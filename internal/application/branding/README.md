# `internal/application/branding/`

**Cross-cutting capability** — the shared branding data and template-render
helper used by every module that renders user-facing copy (web pages, email,
SMS). Keeping it in the application layer gives web and email layouts one source
of truth for the brand data instead of duplicating a `Common` struct in each
platform package.

## Key types

| Symbol   | Purpose                                                        |
|----------|----------------------------------------------------------------|
| `Common` | brand data embedded by every view model: `AppName`, `BaseURL`, `AssetsBaseURL`, `Year` |

## Functions

- `Render(exec, name, data) (string, error)` — executes the named template with
  `data` and returns the output. `exec` is satisfied by both
  `html/template.Template` and `text/template.Template`, so it works with either
  template set.

## Usage

```go
type welcomeView struct {
    branding.Common
    Message string
}

out, err := branding.Render(layout, "content", welcomeView{
    Common:  branding.Common{AppName: "Go Project Starter", BaseURL: cfg.BaseURL, AssetsBaseURL: cfg.AssetsBaseURLOrDefault(), Year: 2026},
    Message: "Hello",
})
```

Templates read `{{.Common.AppName}}`, `{{.Common.BaseURL}}`,
`{{.Common.AssetsBaseURL}}` and `{{.Common.Year}}`.

## Dependency rules

Vendor-free contract; imports stdlib only.
