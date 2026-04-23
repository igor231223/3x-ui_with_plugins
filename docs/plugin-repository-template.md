# Plugin Repository Template

Use this structure in your external plugin repository (for example `3x-ui_plugins`):

```text
plugins/
  <plugin-folder>/
    manifest.json
    README.md
    bin/
      <plugin-daemon-binary>
    assets/            # optional
    scripts/           # optional
```

## Required Files

- `plugins/<plugin-folder>/manifest.json`
- `plugins/<plugin-folder>/README.md`

Catalog scanner reads each plugin folder under `plugins/` and:

- uses `manifest.json` to detect install source and plugin id
- uses `README.md` title + first non-empty line as description in UI

## Minimal `manifest.json` Example

```json
{
  "id": "example-plugin",
  "baseUrl": "http://127.0.0.1:17301",
  "command": "./bin/example-plugind",
  "args": [],
  "workingDir": ".",
  "env": {
    "EXAMPLE_PLUGIN_PORT": "17301"
  },
  "autostart": true
}
```

## README Style Rules (for consistent UI)

1. First line: `# Plugin Name`
2. Second non-empty line: one short description sentence
3. Keep first description under ~120 chars
4. Put details below in sections (`What It Does`, `Install`, `Configuration`)
