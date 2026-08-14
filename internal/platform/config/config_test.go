package config

import "testing"

func TestAssetsBaseURLOrDefault(t *testing.T) {
	if got := (Config{AssetsBaseURL: "", BaseURL: "http://localhost:8080/"}).AssetsBaseURLOrDefault(); got != "http://localhost:8080" {
		t.Fatalf("fallback = %q, want trimmed BaseURL", got)
	}
	if got := (Config{AssetsBaseURL: "  ", BaseURL: "http://x.test"}).AssetsBaseURLOrDefault(); got != "http://x.test" {
		t.Fatalf("blank ASSETS_BASE_URL should fall back, got %q", got)
	}
	if got := (Config{AssetsBaseURL: "https://cdn.example.com/", BaseURL: "http://x.test"}).AssetsBaseURLOrDefault(); got != "https://cdn.example.com" {
		t.Fatalf("custom ASSETS_BASE_URL = %q, want trailing slash trimmed", got)
	}
}
