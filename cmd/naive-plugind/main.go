package main

import (
	"context"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"

	naiveplugin "github.com/mhsanaei/3x-ui/v2/plugins/naive"

	"github.com/gin-gonic/gin"
)

type actionReq struct {
	Enabled  *bool  `json:"enabled,omitempty"`
	Mode     string `json:"mode,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func main() {
	if gin.Mode() == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	svc := naiveplugin.NaivePluginService{}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.GET("/status", func(c *gin.Context) {
		state, err := svc.GetState()
		if err != nil || state == nil {
			c.JSON(http.StatusOK, gin.H{
				"id":           "naive",
				"version":      "0.1.0",
				"installed":    false,
				"enabled":      false,
				"canToggle":    true,
				"canReinstall": true,
				"canUninstall": true,
				"canConfigure": true,
				"health":       "degraded",
				"message":      "naive plugin state unavailable",
			})
			return
		}
		health := "degraded"
		if state.State == naiveplugin.NaivePluginStateHealthy {
			health = "healthy"
		}
		c.JSON(http.StatusOK, gin.H{
			"id":           "naive",
			"version":      "0.1.0",
			"installed":    state.Installed && state.UIHookPresent,
			"enabled":      state.Enabled,
			"canToggle":    true,
			"canReinstall": true,
			"canUninstall": true,
			"canConfigure": true,
			"health":       health,
			"message":      state.LastError,
		})
	})

	r.GET("/actions", func(c *gin.Context) {
		c.JSON(http.StatusOK, []gin.H{
			{"id": "setEnabled", "title": "Set enabled", "description": "Enable or disable plugin", "needsBody": true},
			{"id": "reinstall", "title": "Reinstall", "description": "Reinstall or recover plugin integration", "needsBody": false},
			{"id": "uninstall", "title": "Uninstall", "description": "Uninstall plugin integration", "needsBody": true},
			{"id": "saveConfig", "title": "Save config", "description": "Save runtime configuration", "needsBody": true},
			{"id": "installRuntime", "title": "Install runtime", "description": "Generate runtime artifacts", "needsBody": false},
			{"id": "runRuntimeInstall", "title": "Run runtime installer", "description": "Execute installer script", "needsBody": false},
			{"id": "inboundClientAdd", "title": "Inbound client add", "description": "Handle plugin inbound client add", "needsBody": true},
			{"id": "inboundClientUpdate", "title": "Inbound client update", "description": "Handle plugin inbound client update", "needsBody": true},
			{"id": "inboundClientDelete", "title": "Inbound client delete", "description": "Handle plugin inbound client delete", "needsBody": true},
			{"id": "inboundClientDeleteByEmail", "title": "Inbound client delete by email", "description": "Handle plugin inbound client delete by email", "needsBody": true},
		})
	})

	r.GET("/ui-schema", func(c *gin.Context) {
		cfg, _ := svc.GetRuntimeConfig()
		c.JSON(http.StatusOK, gin.H{
			"title":        "Naive runtime configuration",
			"submitAction": "saveConfig",
			"fields": []gin.H{
				{"key": "domain", "label": "Domain", "type": "text", "required": true, "placeholder": "naive.example.com", "default": cfg.Domain},
				{"key": "port", "label": "Port", "type": "number", "required": true, "default": cfg.Port},
				{"key": "username", "label": "Username", "type": "text", "required": true, "default": cfg.Username},
				{"key": "password", "label": "Password", "type": "password", "required": true, "default": cfg.Password},
			},
		})
	})

	r.GET("/inbound-protocols", func(c *gin.Context) {
		cfg, _ := svc.GetRuntimeConfig()
		c.JSON(http.StatusOK, []gin.H{
			{
				"id":             "naive",
				"title":          "Naive",
				"supportsStream": false,
				"clientIdKey":    "id",
				"fields": []gin.H{
					{"key": "domain", "label": "Domain", "type": "text", "required": true, "placeholder": "naive.example.com", "default": cfg.Domain},
					{"key": "port", "label": "Port", "type": "number", "required": true, "default": cfg.Port},
					{"key": "username", "label": "Username", "type": "text", "required": true, "default": cfg.Username},
					{"key": "password", "label": "Password", "type": "password", "required": true, "default": cfg.Password},
				},
			},
		})
	})

	r.GET("/subscriptions/links", func(c *gin.Context) {
		cfg, _ := svc.GetRuntimeConfig()
		if cfg == nil {
			c.JSON(http.StatusOK, []string{})
			return
		}
		domain := strings.TrimSpace(cfg.Domain)
		username := strings.TrimSpace(cfg.Username)
		password := strings.TrimSpace(cfg.Password)
		if domain == "" || username == "" || password == "" || cfg.Port <= 0 {
			c.JSON(http.StatusOK, []string{})
			return
		}
		u := &neturl.URL{
			Scheme: "naive+https",
			User:   neturl.UserPassword(username, password),
			Host:   fmt.Sprintf("%s:%d", domain, cfg.Port),
		}
		if sid := strings.TrimSpace(c.Query("subId")); sid != "" {
			u.Fragment = "naive-" + sid
		}
		c.JSON(http.StatusOK, []string{u.String()})
	})

	r.GET("/subscriptions/json", func(c *gin.Context) {
		cfg, _ := svc.GetRuntimeConfig()
		if cfg == nil {
			c.JSON(http.StatusOK, []gin.H{})
			return
		}
		domain := strings.TrimSpace(cfg.Domain)
		username := strings.TrimSpace(cfg.Username)
		password := strings.TrimSpace(cfg.Password)
		if domain == "" || username == "" || password == "" || cfg.Port <= 0 {
			c.JSON(http.StatusOK, []gin.H{})
			return
		}
		c.JSON(http.StatusOK, []gin.H{
			{
				"remarks": "naive-plugin",
				"outbounds": []gin.H{
					{
						"protocol": "naive",
						"tag":      "proxy",
						"settings": gin.H{
							"server":      domain,
							"server_port": cfg.Port,
							"username":    username,
							"password":    password,
						},
					},
				},
			},
		})
	})

	r.POST("/actions/:action", func(c *gin.Context) {
		var req actionReq
		_ = c.ShouldBindJSON(&req)
		ctx := c.Request.Context()
		switch c.Param("action") {
		case "setEnabled":
			enabled := req.Enabled != nil && *req.Enabled
			if _, err := svc.SetEnabled(enabled); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "reinstall":
			if _, err := svc.ReinstallIntegration(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "uninstall":
			mode := strings.TrimSpace(req.Mode)
			if mode == "" {
				mode = string(naiveplugin.NaivePluginUninstallIntegrationOnly)
			}
			if _, err := svc.Uninstall(naiveplugin.NaivePluginUninstallMode(mode)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "saveConfig":
			if _, err := svc.SetRuntimeConfig(&naiveplugin.NaiveRuntimeConfig{
				Domain:   strings.TrimSpace(req.Domain),
				Port:     req.Port,
				Username: strings.TrimSpace(req.Username),
				Password: strings.TrimSpace(req.Password),
			}); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "installRuntime":
			if _, err := svc.InstallRuntimeArtifacts(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "runRuntimeInstall":
			if _, err := svc.RunRuntimeInstaller(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		case "inboundClientAdd", "inboundClientUpdate", "inboundClientDelete", "inboundClientDeleteByEmail":
			// Naive plugin does not maintain per-inbound clients in core xray schema.
			// Accept action to keep plugin-protocol client lifecycle fully decoupled from core.
			c.JSON(http.StatusOK, gin.H{"ok": true, "action": c.Param("action"), "noop": true})
			return
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "unsupported action"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "action": c.Param("action"), "ctx": contextName(ctx)})
	})

	port := strings.TrimSpace(os.Getenv("NAIVE_PLUGIN_PORT"))
	if port == "" {
		port = "17231"
	}
	if _, err := strconv.Atoi(port); err != nil {
		port = "17231"
	}
	_ = r.Run("127.0.0.1:" + port)
}

func contextName(_ context.Context) string {
	return "ok"
}
