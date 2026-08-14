# Public assets

Files in this directory are served statically at `/assets/*` by the API and
web servers (see `PUBLIC_DIR` in `.env.example`).

Put shared static files here so both email HTML and web pages can reference
them with an absolute URL:

- `logo.png` — brand logo used by the email layout
  (`{{.Common.AssetsBaseURL}}/assets/logo.png`).

The base URL of these assets is `ASSETS_BASE_URL` (defaults to `APP_BASE_URL`),
so it can point at a CDN in production.

> Note: most email clients block external images by default — recipients may
> need to click "show images" for logos to render.
