# `internal/modules/homepage/`

**Business module** — the module behind the public landing page at `/`. It has
no use cases or data of its own: it only serves the app branding as JSON and
renders a static welcome page from the shared view layout.

## What it does

- `GET /` (JSON adapter) — returns the app branding (`app_name`, `base_url`,
  `assets_base_url`, `year`) so a frontend can render links without hard-coding
  environment values.
- `GET /` (web adapter) — renders the welcome HTML page from the shared
  `platform/view` layout.
- scheduled job (schedule adapter) — a demo "tick" job that logs the current
  time every minute, registered via `RegisterSchedule` on `cmd/scheduler`.

## How it works

The module carries no state. `New(deps)` stores the branding `Settings`
(`AppName`, `BaseURL`, `AssetsBaseURL`, `Year`); the composition root supplies
them once at startup. `RegisterWeb` and `RegisterAPI` mount the two adapters on
their routers:

```text
cmd/web ──► homepage.RegisterWeb(r)
             └─► adapter/web: GET / → welcome page (platform/view layout)
cmd/api ──► homepage.RegisterAPI(r)
             └─► adapter/api: GET / → branding JSON (application.Info)
cmd/scheduler ──► homepage.RegisterSchedule(sched)
             └─► adapter/schedule: homepage.tick → log time every minute
```

## Layers

`domain/`, `application/` and `infrastructure/` exist only to keep the module
skeleton uniform (see `internal/modules/README.md`) and are intentionally
empty — `application` carries just the `Info` branding value. `adapter/web/`
holds the welcome handler and `template/web/` the embedded page template;
`adapter/api/` holds the JSON handler; `adapter/schedule/` registers the demo
"tick" job. The `API` surface (`api.go`) is empty because no other module calls
homepage use cases.
