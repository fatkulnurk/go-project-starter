// Package application is the homepage module's application layer. The homepage
// has no use cases (no commands, no queries) beyond rendering a static page, so
// this package only carries the branding view model shared by its adapters.
package application

// Info is the branding a frontend may need to render links and chrome.
type Info struct {
	AppName       string `json:"app_name"`
	BaseURL       string `json:"base_url"`
	AssetsBaseURL string `json:"assets_base_url"`
	Year          int    `json:"year"`
}
