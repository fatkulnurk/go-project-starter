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
  time every minute and broadcasts a demo pub/sub event on `app.demo.ping`,
  registered via `RegisterSchedule` on `cmd/scheduler`.
- pub/sub handler (pubsub adapter) — subscribes to `app.demo.ping` and logs
  received events, registered via `RegisterPubSub` on `cmd/subscriber`.

## How it works

The module carries no state. `New(deps)` stores the branding `Settings`
(`AppName`, `BaseURL`, `AssetsBaseURL`, `Year`) and the optional pub/sub
`Publisher`; the composition root supplies them once at startup. `RegisterWeb`
and `RegisterAPI` mount the two adapters on their routers:

```text
cmd/web ──► homepage.RegisterWeb(r)
             └─► adapter/web: GET / → welcome page (platform/view layout)
cmd/api ──► homepage.RegisterAPI(r)
             └─► adapter/api: GET / → branding JSON (application.Info)
cmd/scheduler ──► homepage.RegisterSchedule(sched)     (publisher wired in)
             └─► adapter/schedule: homepage.tick → log time + Publish app.demo.ping
cmd/subscriber ──► homepage.RegisterPubSub(bus)
             └─► adapter/pubsub: Subscribe app.demo.ping → log received event
```

The tick→pub/sub→subscriber loop proves the whole broadcast pipeline: the
scheduler produces an event every minute, the configured broker fans it out,
and the subscriber binary logs it. Running only the scheduler logs the tick;
running both binaries (with `PUBSUB_DRIVER=redis`/`rabbitmq`/`kafka`, or the
in-process `memory` bus within one process) shows the event arriving on the
consumer side.

## Layers

`domain/`, `application/` and `infrastructure/` exist only to keep the module
skeleton uniform (see `internal/modules/README.md`) and are intentionally
empty — `application` carries just the `Info` branding value. `adapter/web/`
holds the welcome handler and `template/web/` the embedded page template;
`adapter/api/` holds the JSON handler; `adapter/schedule/` registers the demo
"tick" job; `adapter/pubsub/` registers the demo `app.demo.ping` handler. The
`API` surface (`api.go`) is empty because no other module calls homepage use
cases.
