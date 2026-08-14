package domain

import (
	"strings"
	"testing"
)

func TestObjectKeyRejectsTraversal(t *testing.T) {
	bad := [][3]string{
		{"../", "u1", "avatars"},
		{"user", "u1/..", "avatars"},
		{"user", "u1", "../avatars"},
		{"user", "u1", "avatars\\.."},
		{"user\\..", "u1", "avatars"},
	}
	for _, b := range bad {
		if _, err := ObjectKey(b[0], b[1], b[2], "a.jpg"); err == nil {
			t.Fatalf("ObjectKey(%q,%q,%q) succeeded, want error", b[0], b[1], b[2])
		}
	}
}

func TestUniqueFileNameSanitizesSeparators(t *testing.T) {
	name := `..\..\evil file.jpg`
	key, err := ObjectKey("user", "u1", "avatars", name)
	if err != nil {
		t.Fatalf("ObjectKey error: %v", err)
	}
	if strings.Contains(key, "\\") || strings.Contains(key, "/..") {
		t.Fatalf("key contains separator/traversal: %q", key)
	}
	if !strings.HasPrefix(key, "media/user/u1/avatars/") {
		t.Fatalf("key missing prefix: %q", key)
	}
	if !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("key missing ext: %q", key)
	}
	if strings.Contains(key, "\x00") {
		t.Fatalf("key contains NUL: %q", key)
	}
}

func TestUniqueFileNameCapsLength(t *testing.T) {
	long := strings.Repeat("a", 300) + ".png"
	key, err := ObjectKey("user", "u1", "avatars", long)
	if err != nil {
		t.Fatalf("ObjectKey error: %v", err)
	}
	if len(key) > 512 {
		t.Fatalf("key too long (%d): %q", len(key), key)
	}
}
