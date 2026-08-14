package mailer

import (
	"strings"
	"testing"

	htmltemplate "html/template"
)

func TestEmailLayoutRendersLogoFromAssetsBaseURL(t *testing.T) {
	layout, err := NewEmailLayout()
	if err != nil {
		t.Fatalf("NewEmailLayout error: %v", err)
	}
	content := htmltemplate.Must(layout.Parse("{{define \"content\"}}body{{end}}"))

	type data struct {
		Common
		Body string
	}
	got, err := renderEmailLayout(content, data{
		Common: Common{AppName: "App", BaseURL: "https://app.test", AssetsBaseURL: "https://cdn.test"},
	})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(got, `src="https://cdn.test/assets/logo.png"`) {
		t.Fatalf("expected logo URL from AssetsBaseURL in output:\n%s", got)
	}
	if strings.Contains(got, "#ZgotmplZ") {
		t.Fatalf("html/template sanitized the asset URL:\n%s", got)
	}
}

func TestEmailLayoutFallsBackToTextBrand(t *testing.T) {
	layout, err := NewEmailLayout()
	if err != nil {
		t.Fatalf("NewEmailLayout error: %v", err)
	}
	content := htmltemplate.Must(layout.Parse("{{define \"content\"}}body{{end}}"))

	type data struct {
		Common
		Body string
	}
	got, err := renderEmailLayout(content, data{
		Common: Common{AppName: "Brand", BaseURL: "https://app.test"},
	})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(got, `<span class="brand">Brand</span>`) {
		t.Fatalf("expected text brand fallback when no AssetsBaseURL:\n%s", got)
	}
	if strings.Contains(got, "logo.png") {
		t.Fatalf("unexpected logo reference without AssetsBaseURL:\n%s", got)
	}
}

func renderEmailLayout(t *htmltemplate.Template, d any) (string, error) {
	return renderTemplateString(t, "layout", d)
}

func renderTemplateString(t *htmltemplate.Template, name string, d any) (string, error) {
	var b strings.Builder
	if err := t.ExecuteTemplate(&b, name, d); err != nil {
		return "", err
	}
	return b.String(), nil
}
