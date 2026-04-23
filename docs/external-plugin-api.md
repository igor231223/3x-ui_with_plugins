# External Plugin API (MVP)

This document describes how to connect external plugins to the panel without rebuilding core.

## 1. Manifest registration

Core loads external plugins from:

- `DB/plugins/manifest.json`

Manifest entry schema:

```json
{
  "id": "sample-proxy",
  "baseUrl": "http://127.0.0.1:17231",
  "authToken": "",
  "command": "/usr/local/bin/sample-plugind",
  "args": [],
  "workingDir": ".",
  "env": {
    "NAIVE_PLUGIN_PORT": "17231"
  },
  "autostart": false
}
```

Field notes:

- `id`: unique plugin id
- `baseUrl`: plugin HTTP endpoint used by core
- `command`/`args`/`workingDir`/`env`: optional process startup config
- `autostart`: if true, core starts process on panel startup
- `authToken`: optional bearer token for secure plugin API calls

### Recommended plugin repository layout

For plugin catalogs in panel UI, keep repository structure:

```text
plugins/
  <plugin-name>/
    manifest.json
    README.md
```

Catalog scanner behavior:

- scans `plugins/*` folders
- reads `manifest.json` for plugin id/source
- reads `README.md` and shows title/description as tooltip in UI
- installation from catalog uses `git_repo` source with sub-path (`repo.git#plugins/<plugin-name>`)

## 2. Required plugin endpoints

### `GET /status`

Returns plugin status payload:

```json
{
  "id": "sample-proxy",
  "version": "0.1.0",
  "installed": true,
  "enabled": true,
  "canToggle": true,
  "canReinstall": true,
  "canUninstall": true,
  "canConfigure": true,
  "health": "healthy",
  "message": ""
}
```

### `GET /actions`

Returns action list:

```json
[
  { "id": "setEnabled", "title": "Set enabled", "description": "Enable or disable plugin", "needsBody": true },
  { "id": "reinstall", "title": "Reinstall", "description": "Reinstall integration", "needsBody": false }
]
```

### `POST /actions/:actionId`

Executes action with JSON body.

Examples:

- `setEnabled` body: `{ "enabled": true }`
- `uninstall` body: `{ "mode": "integrationOnly" }`

Return HTTP 2xx on success; 4xx/5xx with JSON error on failure.

## 3. Optional UI schema endpoint

### `GET /ui-schema`

If `canConfigure=true`, plugin may provide schema for generic UI modal.

Example:

```json
{
  "title": "Plugin runtime configuration",
  "submitAction": "saveConfig",
  "fields": [
    { "key": "domain", "label": "Domain", "type": "text", "required": true, "placeholder": "proxy.example.com", "default": "proxy.example.com" },
    { "key": "port", "label": "Port", "type": "number", "required": true, "default": 443 },
    { "key": "username", "label": "Username", "type": "text", "required": true, "default": "user" },
    { "key": "password", "label": "Password", "type": "password", "required": true, "default": "pass" }
  ]
}
```

Supported field `type` values in current UI:

- `text`
- `password`
- `number`
- `bool`

## 4. Core API facade

Panel routes used by UI:

- `GET /panel/api/plugins/registry`
- `GET /panel/api/plugins/:id/status`
- `GET /panel/api/plugins/:id/actions`
- `GET /panel/api/plugins/:id/ui-schema`
- `POST /panel/api/plugins/:id/actions/:action`
- `POST /panel/api/plugins/install`
- `GET /panel/api/plugins/menu-items`
- `POST /panel/api/plugins/routing/sync`

Core proxies external plugin operations through the plugin manager.

Plugin install request payload:

```json
{
  "sourceType": "manifest_url",
  "source": "https://example.com/plugins/manifest.json"
}
```

Supported `sourceType` values:

- `manifest_url` (`https://...` or local `.json` path)
- `git_repo` (`https://...git` or local repo path)
- `zip` (`https://...zip` or local `.zip` path)
- `path` (local directory or file path)

Install safety checks in core:

- validates plugin manifest schema and required fields (`id`, `baseUrl`)
- rejects duplicate plugin IDs
- validates `baseUrl` format
- starts plugin and persists manifest atomically only after successful start
- prevents ZIP path traversal when unpacking archives

## Security for remote plugins

For remote plugin deployment (plugin daemon on another host), do not expose plugin API publicly.

Recommended:

- private network (`WireGuard`/`Tailscale`) between panel and plugin host
- `authToken` in manifest + token validation in plugin daemon
- firewall allowlist only panel host

SSH tunnel is also supported operationally:

1. keep plugin daemon on `127.0.0.1:<port>` on remote host
2. create tunnel from panel host:
   - `ssh -N -L 127.0.0.1:17331:127.0.0.1:17231 user@remote-host`
3. set plugin `baseUrl` to `http://127.0.0.1:17331`
4. optionally keep `authToken` enabled for defense in depth

Routing sync behavior:

- panel reads current Xray template routing/outbounds
- panel calls plugin routing apply endpoint for compatible plugins
- endpoint is additive and does not change built-in Xray routing path

## 5. Quick start: new plugin in 5 minutes

This is the shortest path to make a new external plugin visible in Plugin Registry.

### Step 1: create a tiny HTTP daemon

Minimal Go example:

```go
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()

	r.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"id":           "hello-plugin",
			"version":      "0.1.0",
			"installed":    true,
			"enabled":      true,
			"canToggle":    true,
			"canReinstall": false,
			"canUninstall": false,
			"canConfigure": true,
			"health":       "healthy",
			"message":      "",
		})
	})

	r.GET("/actions", func(c *gin.Context) {
		c.JSON(200, []gin.H{
			{"id": "setEnabled", "title": "Set enabled", "description": "Enable or disable plugin", "needsBody": true},
			{"id": "saveConfig", "title": "Save config", "description": "Save plugin config", "needsBody": true},
		})
	})

	r.GET("/ui-schema", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"title":        "Hello plugin config",
			"submitAction": "saveConfig",
			"fields": []gin.H{
				{"key": "greeting", "label": "Greeting", "type": "text", "required": true, "default": "hello"},
				{"key": "enabled", "label": "Enabled", "type": "bool", "required": true, "default": true},
			},
		})
	})

	r.POST("/actions/:actionId", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "action": c.Param("actionId")})
	})

	_ = r.Run("127.0.0.1:17241")
}
```

### Step 2: register plugin in manifest

Add item to `DB/plugins/manifest.json`:

```json
{
  "id": "hello-plugin",
  "baseUrl": "http://127.0.0.1:17241",
  "command": "/usr/local/bin/hello-plugind",
  "args": [],
  "workingDir": ".",
  "env": {},
  "autostart": true
}
```

Important:

- `id` in manifest and `GET /status` should match.
- `baseUrl` must point to daemon address.
- If `autostart=false`, you must run daemon manually.

### Step 3: restart panel and verify

After panel restart, open dashboard:

- Plugin appears in Plugin Registry.
- Health and message come from `GET /status`.
- Action buttons come from `GET /actions`.
- Configure modal comes from `GET /ui-schema`.

## 6. Compatibility notes

- Keep action IDs stable; UI and automation may rely on them.
- Return JSON errors for failed actions with non-2xx status.
- Keep endpoint latency low (`/status` is called often).
- Prefer loopback bind (`127.0.0.1`) unless remote access is required.
- On `disable` or `uninstall`, menu entries are hidden automatically.

## 7. Protocol provider endpoints (for dynamic inbound protocol UI)

If plugin wants to add a protocol to `Create inbound` protocol dropdown, implement:

### `GET /inbound-protocols`

Returns protocol definitions:

```json
[
  {
    "id": "sample-proxy",
    "title": "Sample Proxy",
    "supportsStream": false,
    "clientIdKey": "id",
    "fields": [
      { "key": "domain", "label": "Domain", "type": "text", "required": true, "default": "proxy.example.com" },
      { "key": "port", "label": "Port", "type": "number", "required": true, "default": 443 }
    ]
  }
]
```

UI behavior:

- `id` appears in inbound protocol dropdown.
- Built-in protocols keep existing forms.
- Plugin protocol fields are rendered dynamically from `fields`.

`clientIdKey` controls how core identifies client rows for client operations. Supported values:

- `id`
- `password`
- `email`
- `auth`

For full plugin-owned client lifecycle, plugin should expose action IDs:

- `inboundClientAdd`
- `inboundClientUpdate`
- `inboundClientDelete`
- `inboundClientDeleteByEmail`

Core dispatches these actions for plugin protocols instead of applying built-in `settings.clients` mutation logic.

## 8. Subscription provider endpoints (optional, full-stack plugin path)

To contribute plugin-specific subscription outputs without core edits:

### `GET /subscriptions/links?subId=<id>&host=<host>`

Returns an array of subscription links:

```json
["https+proxy://user:pass@proxy.example.com:443#plugin-subid"]
```

### `GET /subscriptions/json?subId=<id>&host=<host>`

Returns an array of JSON subscription objects:

```json
[
  {
    "remarks": "sample-plugin",
    "outbounds": [
      {
        "protocol": "sample-proxy",
        "tag": "proxy",
        "settings": {
          "server": "proxy.example.com",
          "server_port": 443,
          "username": "user",
          "password": "pass"
        }
      }
    ]
  }
]
```

## 9. Dynamic sidebar menu (optional)

Plugins can add native-looking left sidebar entries by implementing:

### `GET /menu-items`

Returns:

```json
[
  {
    "key": "/panel/plugins/hello",
    "icon": "appstore",
    "title": "Hello plugin"
  },
  {
    "key": "https://docs.example.com/hello-plugin",
    "icon": "book",
    "title": "Plugin docs",
    "external": true
  }
]
```

Rules:

- internal routes must use absolute path keys (`/panel/...`)
- external links should be full `http/https` URLs
- duplicate or invalid menu items are ignored by core
- if endpoint returns `404`, core treats it as "no menu items"

## 10. Plugin routing integration (optional)

To participate in unified routing sync from Plugins page, implement:

### `POST /routing/apply`

Input payload is full current Xray template config (`routing`, `outbounds`, etc).

Plugin should:

- map relevant routing data to plugin runtime format
- validate and persist rules safely
- return `2xx` on success
- return `404` if routing sync is not supported (core ignores)

Example plugin behavior:

- routing rules are translated and stored for plugin runtime
- rules are applied on next runtime install/reinstall
