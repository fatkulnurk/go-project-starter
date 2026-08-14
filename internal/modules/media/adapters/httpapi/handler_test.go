package httpapi

import (
	"context"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
)

// fakeURLGen returns a canned URL or delegates to the error.
type fakeURLGen struct {
	url string
	err error
}

func (f *fakeURLGen) URL(context.Context, string) (string, error) { return f.url, f.err }

func testMedia() *domain.Media {
	return &domain.Media{ID: "m1", FileName: "media/user/u1/avatars/a.jpg"}
}

func TestMediaURLUsesGenerator(t *testing.T) {
	h := &handler{deps: Deps{
		URLGenerator: &fakeURLGen{url: "https://cdn.example.com/a.jpg"},
		BaseURL:      "https://app.example.com",
	}}
	if got := h.mediaURL(context.Background(), testMedia()); got != "https://cdn.example.com/a.jpg" {
		t.Fatalf("mediaURL = %q, want generated URL", got)
	}
}

func TestMediaURLFallsBackToDownloadEndpoint(t *testing.T) {
	cases := []struct {
		name string
		gen  storage.URLGenerator
	}{
		{name: "nil generator"},
		{name: "errnoURL", gen: &fakeURLGen{err: storage.ErrNoURL}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &handler{deps: Deps{URLGenerator: tc.gen, BaseURL: "https://app.example.com/"}}
			want := "https://app.example.com/api/v1/media/m1/download"
			if got := h.mediaURL(context.Background(), testMedia()); got != want {
				t.Fatalf("mediaURL = %q, want %q", got, want)
			}
		})
	}
}

func TestMediaURLEmptyGeneratorURLFallsBack(t *testing.T) {
	h := &handler{deps: Deps{URLGenerator: &fakeURLGen{url: ""}, BaseURL: "https://app.example.com"}}
	want := "https://app.example.com/api/v1/media/m1/download"
	if got := h.mediaURL(context.Background(), testMedia()); got != want {
		t.Fatalf("mediaURL = %q, want %q", got, want)
	}
}
