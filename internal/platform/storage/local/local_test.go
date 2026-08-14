package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

func TestResolveRejectsTraversal(t *testing.T) {
	base := filepath.Join(t.TempDir(), "media")
	s := NewLocal(config.LocalStorageConfig{Dir: base})

	evil := []string{
		"../secret.txt",
		"../../etc/passwd",
		"media/../../x",
		`media\..\..\escape`,
		`D:\secret\x`,
		"media/user/../../../outside",
	}
	for _, key := range evil {
		if err := s.Put(context.Background(), key, strings.NewReader("x")); !errors.Is(err, storage.ErrInvalidKey) {
			t.Fatalf("Put(%q) error = %v, want storage.ErrInvalidKey", key, err)
		}
	}
}

func TestURLReturnsErrNoURL(t *testing.T) {
	s := NewLocal(config.LocalStorageConfig{Dir: filepath.Join(t.TempDir(), "media")})
	if _, err := s.URL(context.Background(), "media/user/u1/a.jpg"); !errors.Is(err, storage.ErrNoURL) {
		t.Fatalf("URL error = %v, want storage.ErrNoURL", err)
	}
}

func TestResolveStaysInsideBaseDir(t *testing.T) {
	base := filepath.Join(t.TempDir(), "media")
	s := NewLocal(config.LocalStorageConfig{Dir: base})

	key := "media/user/u1/avatars/file-1.jpg"
	if err := s.Put(context.Background(), key, strings.NewReader("hello")); err != nil {
		t.Fatalf("Put(%q) error = %v", key, err)
	}

	rc, err := s.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", key, err)
	}
	defer rc.Close()
	b, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(key)))
	if err != nil {
		t.Fatalf("expected file under base dir: %v", err)
	}
	if string(b) != "hello" {
		t.Fatalf("content = %q, want %q", b, "hello")
	}
}
