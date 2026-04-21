package controller

import (
	"fmt"
	"net/http"

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
		c.AbortWithStatus(http.StatusNotFound)
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
