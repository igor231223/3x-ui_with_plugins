package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/plugins"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"

	"github.com/gin-gonic/gin"
)

// APIController handles the main API routes for the 3x-ui panel, including inbounds and server management.
type APIController struct {
	BaseController
	inboundController *InboundController
	serverController  *ServerController
	pluginManager     *plugins.Manager
	Tgbot             service.Tgbot
}

type pluginSetEnabledReq struct {
	Enabled bool `json:"enabled"`
}

type pluginUninstallReq struct {
	Mode string `json:"mode"`
}

type pluginInstallReq struct {
	SourceType string `json:"sourceType"`
	Source     string `json:"source"`
}

type pluginCatalogReq struct {
	Source string `json:"source"`
}

type pluginInstallPolicyReq struct {
	StrictAuth bool `json:"strictAuth"`
}

// NewAPIController creates a new APIController instance and initializes its routes.
func NewAPIController(g *gin.RouterGroup, customGeo *service.CustomGeoService, pluginManager *plugins.Manager) *APIController {
	a := &APIController{
		pluginManager: pluginManager,
	}
	a.initRouter(g, customGeo)
	return a
}

// checkAPIAuth is a middleware that returns 404 for unauthenticated API requests
// to hide the existence of API endpoints from unauthorized users
func (a *APIController) checkAPIAuth(c *gin.Context) {
	if !session.IsLogin(c) {
		if isAjax(c) {
			pureJsonMsg(c, http.StatusUnauthorized, false, "unauthorized")
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
		return
	}
	c.Next()
}

// initRouter sets up the API routes for inbounds, server, and other endpoints.
func (a *APIController) initRouter(g *gin.RouterGroup, customGeo *service.CustomGeoService) {
	// Main API group
	api := g.Group("/panel/api")
	api.Use(a.checkAPIAuth)

	// Inbounds API
	inbounds := api.Group("/inbounds")
	a.inboundController = NewInboundController(inbounds)

	// Server API
	server := api.Group("/server")
	a.serverController = NewServerController(server)
	if a.pluginManager != nil {
		a.pluginManager.RegisterServerRoutes(server)
	}

	NewCustomGeoController(api.Group("/custom-geo"), customGeo)

	// Extra routes
	api.GET("/backuptotgbot", a.BackuptoTgbot)
	api.GET("/plugins/registry", a.pluginsRegistry)
	api.POST("/plugins/:id/enabled", a.setPluginEnabled)
	api.POST("/plugins/:id/reinstall", a.reinstallPlugin)
	api.POST("/plugins/:id/uninstall", a.uninstallPlugin)
	api.GET("/plugins/:id/status", a.pluginStatus)
	api.GET("/plugins/:id/actions", a.pluginActions)
	api.GET("/plugins/:id/ui-schema", a.pluginUISchema)
	api.POST("/plugins/:id/actions/:action", a.pluginActionExecute)
	api.GET("/plugins/inbound-protocols", a.pluginInboundProtocols)
	api.GET("/plugins/menu-items", a.pluginMenuItems)
	api.GET("/plugins/install/policy", a.getPluginInstallPolicy)
	api.POST("/plugins/install/policy", a.setPluginInstallPolicy)
	api.POST("/plugins/install/precheck", a.precheckPluginInstall)
	api.POST("/plugins/install", a.installPlugin)
	api.POST("/plugins/catalog", a.pluginCatalog)
	api.POST("/plugins/routing/sync", a.syncPluginRouting)
}

// BackuptoTgbot sends a backup of the panel data to Telegram bot admins.
func (a *APIController) BackuptoTgbot(c *gin.Context) {
	a.Tgbot.SendBackupToAdmins()
}

func (a *APIController) pluginsRegistry(c *gin.Context) {
	if a.pluginManager == nil {
		jsonObj(c, []plugins.Status{}, nil)
		return
	}
	jsonObj(c, a.pluginManager.Registry(c.Request.Context()), nil)
}

func (a *APIController) setPluginEnabled(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "set plugin enabled", fmt.Errorf("plugin manager unavailable"))
		return
	}
	var req pluginSetEnabledReq
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "set plugin enabled", err)
		return
	}
	pluginID := c.Param("id")
	if err := a.pluginManager.SetEnabled(c.Request.Context(), pluginID, req.Enabled); err != nil {
		jsonMsg(c, "set plugin enabled", err)
		return
	}
	status, err := a.pluginManager.GetStatus(c.Request.Context(), pluginID)
	if err != nil {
		jsonMsg(c, "set plugin enabled", err)
		return
	}
	jsonObj(c, status, nil)
}

func (a *APIController) pluginStatus(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin status", fmt.Errorf("plugin manager unavailable"))
		return
	}
	status, err := a.pluginManager.GetStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		jsonMsg(c, "plugin status", err)
		return
	}
	jsonObj(c, status, nil)
}

func (a *APIController) reinstallPlugin(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin reinstall", fmt.Errorf("plugin manager unavailable"))
		return
	}
	pluginID := c.Param("id")
	if err := a.pluginManager.Reinstall(c.Request.Context(), pluginID); err != nil {
		jsonMsg(c, "plugin reinstall", err)
		return
	}
	status, err := a.pluginManager.GetStatus(c.Request.Context(), pluginID)
	if err != nil {
		jsonMsg(c, "plugin reinstall", err)
		return
	}
	jsonObj(c, status, nil)
}

func (a *APIController) uninstallPlugin(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin uninstall", fmt.Errorf("plugin manager unavailable"))
		return
	}
	var req pluginUninstallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "plugin uninstall", err)
		return
	}
	pluginID := c.Param("id")
	if err := a.pluginManager.Uninstall(c.Request.Context(), pluginID, req.Mode); err != nil {
		jsonMsg(c, "plugin uninstall", err)
		return
	}
	status, err := a.pluginManager.GetStatus(c.Request.Context(), pluginID)
	if err != nil {
		// For external plugins, uninstall can fully remove plugin from runtime/manifest.
		// In this case, "not found" is an expected post-state and should be treated as success.
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			jsonObj(c, gin.H{"id": pluginID, "uninstalled": true}, nil)
			return
		}
		jsonMsg(c, "plugin uninstall", err)
		return
	}
	jsonObj(c, status, nil)
}

func (a *APIController) pluginActions(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin actions", fmt.Errorf("plugin manager unavailable"))
		return
	}
	actions, err := a.pluginManager.Actions(c.Request.Context(), c.Param("id"))
	if err != nil {
		jsonMsg(c, "plugin actions", err)
		return
	}
	jsonObj(c, actions, nil)
}

func (a *APIController) pluginActionExecute(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin action execute", fmt.Errorf("plugin manager unavailable"))
		return
	}
	payload := map[string]any{}
	_ = c.ShouldBindJSON(&payload)
	if err := a.pluginManager.ExecuteAction(c.Request.Context(), c.Param("id"), c.Param("action"), payload); err != nil {
		jsonMsg(c, "plugin action execute", err)
		return
	}
	status, err := a.pluginManager.GetStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		jsonMsg(c, "plugin action execute", err)
		return
	}
	jsonObj(c, status, nil)
}

func (a *APIController) pluginUISchema(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin ui-schema", fmt.Errorf("plugin manager unavailable"))
		return
	}
	schema, err := a.pluginManager.UISchema(c.Request.Context(), c.Param("id"))
	if err != nil {
		jsonMsg(c, "plugin ui-schema", err)
		return
	}
	jsonObj(c, schema, nil)
}

func (a *APIController) pluginInboundProtocols(c *gin.Context) {
	if a.pluginManager == nil {
		jsonObj(c, []plugins.InboundProtocol{}, nil)
		return
	}
	jsonObj(c, a.pluginManager.InboundProtocols(c.Request.Context()), nil)
}

func (a *APIController) installPlugin(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin install", fmt.Errorf("plugin manager unavailable"))
		return
	}
	var req pluginInstallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "plugin install", err)
		return
	}
	settingSvc := service.SettingService{}
	strictAuth, err := settingSvc.GetPluginInstallStrictAuth()
	if err != nil {
		jsonMsg(c, "plugin install", err)
		return
	}
	_, precheckWarnings, err := a.pluginManager.PrecheckExternalSource(c.Request.Context(), req.SourceType, req.Source)
	if err != nil {
		jsonMsg(c, "plugin install", err)
		return
	}
	if strictAuth && len(precheckWarnings) > 0 {
		jsonMsg(c, "plugin install", fmt.Errorf("blocked by install policy: remote plugin requires authToken"))
		return
	}
	ids, warnings, err := a.pluginManager.InstallExternalFromSourceWithWarnings(c.Request.Context(), req.SourceType, req.Source)
	if err != nil {
		jsonMsg(c, "plugin install", err)
		return
	}
	jsonObj(c, gin.H{"installed": ids, "warnings": warnings}, nil)
}

func (a *APIController) precheckPluginInstall(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin install precheck", fmt.Errorf("plugin manager unavailable"))
		return
	}
	var req pluginInstallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "plugin install precheck", err)
		return
	}
	ids, warnings, err := a.pluginManager.PrecheckExternalSource(c.Request.Context(), req.SourceType, req.Source)
	if err != nil {
		jsonMsg(c, "plugin install precheck", err)
		return
	}
	settingSvc := service.SettingService{}
	strictAuth, err := settingSvc.GetPluginInstallStrictAuth()
	if err != nil {
		jsonMsg(c, "plugin install precheck", err)
		return
	}
	blocked := strictAuth && len(warnings) > 0
	jsonObj(c, gin.H{"detected": ids, "warnings": warnings, "strictAuth": strictAuth, "blocked": blocked}, nil)
}

func (a *APIController) getPluginInstallPolicy(c *gin.Context) {
	settingSvc := service.SettingService{}
	strictAuth, err := settingSvc.GetPluginInstallStrictAuth()
	if err != nil {
		jsonMsg(c, "plugin install policy", err)
		return
	}
	jsonObj(c, gin.H{"strictAuth": strictAuth}, nil)
}

func (a *APIController) setPluginInstallPolicy(c *gin.Context) {
	var req pluginInstallPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "plugin install policy", err)
		return
	}
	settingSvc := service.SettingService{}
	if err := settingSvc.SetPluginInstallStrictAuth(req.StrictAuth); err != nil {
		jsonMsg(c, "plugin install policy", err)
		return
	}
	jsonObj(c, gin.H{"strictAuth": req.StrictAuth}, nil)
}

func (a *APIController) pluginMenuItems(c *gin.Context) {
	if a.pluginManager == nil {
		jsonObj(c, []plugins.MenuItem{}, nil)
		return
	}
	jsonObj(c, a.pluginManager.MenuItems(c.Request.Context()), nil)
}

func (a *APIController) pluginCatalog(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin catalog", fmt.Errorf("plugin manager unavailable"))
		return
	}
	var req pluginCatalogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "plugin catalog", err)
		return
	}
	items, err := a.pluginManager.CatalogFromSource(c.Request.Context(), req.Source)
	if err != nil {
		jsonMsg(c, "plugin catalog", err)
		return
	}
	jsonObj(c, items, nil)
}

func (a *APIController) syncPluginRouting(c *gin.Context) {
	if a.pluginManager == nil {
		jsonMsg(c, "plugin routing sync", fmt.Errorf("plugin manager unavailable"))
		return
	}
	settingSvc := service.SettingService{}
	tpl, err := settingSvc.GetXrayConfigTemplate()
	if err != nil {
		jsonMsg(c, "plugin routing sync", err)
		return
	}
	payload := map[string]any{}
	if strings.TrimSpace(tpl) != "" {
		if err := json.Unmarshal([]byte(tpl), &payload); err != nil {
			jsonMsg(c, "plugin routing sync", fmt.Errorf("invalid xray template config: %w", err))
			return
		}
	}
	result := a.pluginManager.SyncRouting(c.Request.Context(), payload)
	failed := 0
	for _, v := range result {
		if strings.TrimSpace(v) != "" {
			failed++
		}
	}
	jsonObj(c, gin.H{
		"results": result,
		"failed":  failed,
		"total":   len(result),
	}, nil)
}
