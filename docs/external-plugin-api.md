# External Plugin API (MVP)

This document describes how to connect external plugins to the panel without rebuilding core.

## 1. Manifest registration

Core loads external plugins from:

- `DB/plugins/manifest.json`

Manifest entry schema:

```json
{
  "id": "naive",
  "baseUrl": "http://127.0.0.1:17231",
  "command": "/usr/local/bin/naive-plugind",
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

## 2. Required plugin endpoints

### `GET /status`

Returns plugin status payload:

```json
{
  "id": "naive",
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
  "title": "Naive runtime configuration",
  "submitAction": "saveConfig",
  "fields": [
    { "key": "domain", "label": "Domain", "type": "text", "required": true, "placeholder": "naive.example.com", "default": "naive.example.com" },
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

Core proxies external plugin operations through the plugin manager.

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

## 7. Protocol provider endpoints (for dynamic inbound protocol UI)

If plugin wants to add a protocol to `Create inbound` protocol dropdown, implement:

### `GET /inbound-protocols`

Returns protocol definitions:

```json
[
  {
    "id": "naive",
    "title": "Naive",
    "supportsStream": false,
    "clientIdKey": "id",
    "fields": [
      { "key": "domain", "label": "Domain", "type": "text", "required": true, "default": "naive.example.com" },
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
["naive+https://user:pass@naive.example.com:443#naive-subid"]
```

### `GET /subscriptions/json?subId=<id>&host=<host>`

Returns an array of JSON subscription objects:

```json
[
  {
    "remarks": "naive-plugin",
    "outbounds": [
      {
        "protocol": "naive",
        "tag": "proxy",
        "settings": {
          "server": "naive.example.com",
          "server_port": 443,
          "username": "user",
          "password": "pass"
        }
      }
    ]
  }
]
```
