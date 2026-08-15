package application

// Info is the branding a frontend may need to render links and chrome.
type Info struct {
	AppName       string `json:"app_name"`
	BaseURL       string `json:"base_url"`
	AssetsBaseURL string `json:"assets_base_url"`
	Year          int    `json:"year"`
}
