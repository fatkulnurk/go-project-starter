# `internal/platform/view/`

**Technical infrastructure** — renders HTML pages (browser views) from embedded
Go templates. It owns the shared base layout — the *how* of every page — while
business modules attach their own content templates via `ParseFS`, mirroring the
email layout in `internal/platform/mailer`.

## Key types

| Symbol   | Purpose                                                    |
|----------|------------------------------------------------------------|
| `Common` | branding (`branding.Common`) embedded by the base layout    |

## Functions

- `NewLayout() (*html/template.Template, error)` — parses the embedded
  `base.html` into a template set that defines the `layout` shell and a
  `content` block. Modules attach their own content templates to it with
  `(*html/template.Template).ParseFS`.

## Usage

```go
layout, err := view.NewLayout()
if err != nil { return err }

tpl, err := layout.ParseFS(moduleTemplates, "welcome/*.html")
// then render with branding data:
out, err := branding.Render(tpl, "content", welcomeView{Common: branding.Common{...}})
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
