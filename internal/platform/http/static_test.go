package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMountStaticServesFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "logo.png"), []byte("png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	MountStatic(r, dir, "/assets")

	req := httptest.NewRequest(http.MethodGet, "/assets/logo.png", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /assets/logo.png = %d, want 200 (body: %q)", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "png-bytes" {
		t.Fatalf("body = %q, want file contents", rr.Body.String())
	}
}

func TestMountStaticMissingFile404(t *testing.T) {
	r := chi.NewRouter()
	MountStatic(r, t.TempDir(), "/assets")

	req := httptest.NewRequest(http.MethodGet, "/assets/nope.png", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /assets/nope.png = %d, want 404", rr.Code)
	}
}

func TestMountStaticRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("top-secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	MountStatic(r, dir, "/assets")

	for _, path := range []string{"/assets/../../secret.txt", "/assets/%2e%2e/secret.txt"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404 (path traversal must be rejected)", path, rr.Code)
		}
	}
}
