package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/application"
	"github.com/go-chi/chi/v5"
)

func TestInfoReturnsBranding(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Info: application.Info{
			AppName:       "My App",
			BaseURL:       "https://app.test",
			AssetsBaseURL: "https://cdn.test",
			Year:          2026,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}

	var body struct {
		Data application.Info `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.AppName != "My App" || body.Data.BaseURL != "https://app.test" ||
		body.Data.AssetsBaseURL != "https://cdn.test" || body.Data.Year != 2026 {
		t.Fatalf("unexpected branding: %+v", body.Data)
	}
}
