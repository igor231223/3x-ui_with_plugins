# <Plugin Name>

One-line description for plugin catalog tooltip.

## What It Does

- Briefly describe the plugin purpose.
- Mention key user-facing features.

## Requirements

- 3x-ui plugin API support: required version/range.
- External dependencies (if any), for example `caddy`, `sing-box`, `docker`.

## Install

Use Plugins page in panel and install from catalog or source:

- `manifest_url`: `https://.../manifest.json`
- `git_repo`: `https://...repo.git#plugins/<plugin-folder>`
- `zip`: `https://.../plugin.zip`
- `path`: local directory or manifest path

## Configuration

- List required settings.
- List optional settings.

## Endpoints

- `GET /status`
- `GET /actions`
- `POST /actions/:action`
- `GET /ui-schema` (optional)
- `GET /inbound-protocols` (optional)
- `GET /subscriptions/links` (optional)
- `GET /subscriptions/json` (optional)
- `GET /menu-items` (optional)

## Notes

- Add migration notes if behavior changed.
- Add troubleshooting hints for common errors.
