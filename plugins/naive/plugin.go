package naive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/plugins"
	webservice "github.com/mhsanaei/3x-ui/v2/web/service"
)

type Plugin struct {
	naiveService NaivePluginService
}

type pluginUninstallReq struct {
	Mode string `json:"mode"`
}

type pluginRuntimeConfigReq struct {
	Domain   string `json:"domain"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type pluginSetEnabledReq struct {
	Enabled bool `json:"enabled"`
}

type pluginPreviewReq struct {
	SubID string `json:"subId"`
	Host  string `json:"host"`
}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) ID() string {
	return "naive"
}

func (p *Plugin) OnStart(_ context.Context) error {
	return nil
}

func (p *Plugin) OnStop(_ context.Context) error {
	return nil
}

func (p *Plugin) Status(_ context.Context) plugins.Status {
	state, err := p.naiveService.GetState()
	if err != nil || state == nil {
		return plugins.Status{
			ID:        p.ID(),
			Version:   "0.1.0",
			Installed: false,
			Enabled:   false,
			Health:    "degraded",
			Message:   "naive plugin state unavailable",
		}
	}
	health := "degraded"
	if state.State == NaivePluginStateHealthy {
		health = "healthy"
	}
	return plugins.Status{
		ID:        p.ID(),
		Version:   "0.1.0",
		Installed: state.Installed && state.UIHookPresent,
		Enabled:   state.Enabled,
		Health:    health,
		Message:   state.LastError,
	}
}

func (p *Plugin) GetEnabled(_ context.Context) (bool, error) {
	state, err := p.naiveService.GetState()
	if err != nil {
		return false, err
	}
	return state.Enabled, nil
}

func (p *Plugin) SetEnabled(_ context.Context, enabled bool) error {
	if enabled {
		cfg, cfgErr := p.naiveService.GetRuntimeConfig()
		if cfgErr != nil {
			return cfgErr
		}
		realityListen, realityPort := detectRealityBinding()
		preflight := p.naiveService.ValidatePreflight(NaivePreflightRequest{
			RealityListen: realityListen,
			RealityPort:   realityPort,
			NaiveListen:   "0.0.0.0",
			NaivePort:     cfg.Port,
			Domain:        cfg.Domain,
			EnableNaive:   true,
		})
		if !preflight.OK {
			return fmt.Errorf("naive preflight failed: %s", strings.Join(preflight.Errors, "; "))
		}
	}
	_, err := p.naiveService.SetEnabled(enabled)
	return err
}

func (p *Plugin) Reinstall(_ context.Context) error {
	_, err := p.naiveService.ReinstallIntegration()
	return err
}

func (p *Plugin) Uninstall(_ context.Context, mode string) error {
	if strings.TrimSpace(mode) == "" {
		mode = string(NaivePluginUninstallIntegrationOnly)
	}
	_, err := p.naiveService.Uninstall(NaivePluginUninstallMode(mode))
	return err
}

func (p *Plugin) HasConfiguration(_ context.Context) bool {
	return true
}

func (p *Plugin) RegisterServerRoutes(serverGroup *gin.RouterGroup) {
	serverGroup.GET("/naive/state", p.getState)
	serverGroup.GET("/naive/config", p.getConfig)
	serverGroup.GET("/naive/runtime/health", p.getRuntimeHealth)

	serverGroup.POST("/naive/preflight", p.preflight)
	serverGroup.POST("/naive/reinstall", p.reinstall)
	serverGroup.POST("/naive/uninstall", p.uninstallHandler)
	serverGroup.POST("/naive/config", p.setConfig)
	serverGroup.POST("/naive/enabled", p.setEnabled)
	serverGroup.POST("/naive/preview", p.preview)
	serverGroup.POST("/naive/runtime/install", p.installRuntime)
	serverGroup.POST("/naive/runtime/run-install", p.runRuntimeInstall)
}

func (p *Plugin) preflight(c *gin.Context) {
	var req NaivePreflightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pluginJSONMsg(c, "Naive preflight validation", err)
		return
	}
	pluginJSONObj(c, p.naiveService.ValidatePreflight(req))
}

func (p *Plugin) getState(c *gin.Context) {
	state, err := p.naiveService.GetState()
	if err != nil {
		pluginJSONMsg(c, "Get Naive plugin state", err)
		return
	}
	pluginJSONObj(c, state)
}

func (p *Plugin) reinstall(c *gin.Context) {
	state, err := p.naiveService.ReinstallIntegration()
	if err != nil {
		pluginJSONMsg(c, "Reinstall Naive plugin integration", err)
		return
	}
	pluginJSONObj(c, state)
}

func (p *Plugin) uninstallHandler(c *gin.Context) {
	var req pluginUninstallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pluginJSONMsg(c, "Uninstall Naive plugin integration", err)
		return
	}
	if err := p.Uninstall(c.Request.Context(), req.Mode); err != nil {
		pluginJSONMsg(c, "Uninstall Naive plugin integration", err)
		return
	}
	state, err := p.naiveService.GetState()
	if err != nil {
		pluginJSONMsg(c, "Uninstall Naive plugin integration", err)
		return
	}
	pluginJSONObj(c, state)
}

func (p *Plugin) getConfig(c *gin.Context) {
	cfg, err := p.naiveService.GetRuntimeConfig()
	if err != nil {
		pluginJSONMsg(c, "Get Naive plugin config", err)
		return
	}
	pluginJSONObj(c, cfg)
}

func (p *Plugin) setConfig(c *gin.Context) {
	var req pluginRuntimeConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pluginJSONMsg(c, "Set Naive plugin config", err)
		return
	}
	realityListen, realityPort := detectRealityBinding()
	preflight := p.naiveService.ValidatePreflight(NaivePreflightRequest{
		RealityListen: realityListen,
		RealityPort:   realityPort,
		NaiveListen:   "0.0.0.0",
		NaivePort:     req.Port,
		Domain:        req.Domain,
		EnableNaive:   true,
	})
	if !preflight.OK {
		pluginJSONObj(c, preflight)
		return
	}
	cfg, err := p.naiveService.SetRuntimeConfig(&NaiveRuntimeConfig{
		Domain:   req.Domain,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		pluginJSONMsg(c, "Set Naive plugin config", err)
		return
	}
	pluginJSONObj(c, cfg)
}

func (p *Plugin) setEnabled(c *gin.Context) {
	var req pluginSetEnabledReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pluginJSONMsg(c, "Set Naive plugin enabled", err)
		return
	}
	err := p.SetEnabled(c.Request.Context(), req.Enabled)
	if err != nil {
		pluginJSONMsg(c, "Set Naive plugin enabled", err)
		return
	}
	state, err := p.naiveService.GetState()
	if err != nil {
		pluginJSONMsg(c, "Set Naive plugin enabled", err)
		return
	}
	pluginJSONObj(c, state)
}

func (p *Plugin) getRuntimeHealth(c *gin.Context) {
	health, err := p.naiveService.GetRuntimeHealth()
	if err != nil {
		pluginJSONMsg(c, "Get Naive runtime health", err)
		return
	}
	pluginJSONObj(c, health)
}

func (p *Plugin) installRuntime(c *gin.Context) {
	health, err := p.naiveService.InstallRuntimeArtifacts()
	if err != nil {
		pluginJSONMsg(c, "Install Naive runtime", err)
		return
	}
	pluginJSONObj(c, health)
}

func (p *Plugin) runRuntimeInstall(c *gin.Context) {
	result, err := p.naiveService.RunRuntimeInstaller()
	if err != nil {
		pluginJSONMsg(c, "Run Naive runtime installer", err)
		return
	}
	pluginJSONObj(c, result)
}

func (p *Plugin) preview(c *gin.Context) {
	var req pluginPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pluginJSONMsg(c, "Naive subscription preview", err)
		return
	}
	req.SubID = strings.TrimSpace(req.SubID)
	if req.SubID == "" {
		pluginJSONMsg(c, "Naive subscription preview", fmt.Errorf("subId is required"))
		return
	}

	state, err := p.naiveService.GetState()
	if err != nil {
		pluginJSONMsg(c, "Naive subscription preview", err)
		return
	}
	cfg, err := p.naiveService.GetRuntimeConfig()
	if err != nil {
		pluginJSONMsg(c, "Naive subscription preview", err)
		return
	}

	inboundService := webservice.InboundService{}
	inbounds, err := inboundService.GetAllInbounds()
	if err != nil {
		pluginJSONMsg(c, "Naive subscription preview", err)
		return
	}
	hasSubClient := false
	for _, inbound := range inbounds {
		clients, cErr := inboundService.GetClients(inbound)
		if cErr != nil {
			continue
		}
		for _, client := range clients {
			if client.Enable && strings.TrimSpace(client.SubID) == req.SubID {
				hasSubClient = true
				break
			}
		}
		if hasSubClient {
			break
		}
	}

	hasConfig := cfg != nil &&
		strings.TrimSpace(cfg.Domain) != "" &&
		strings.TrimSpace(cfg.Username) != "" &&
		strings.TrimSpace(cfg.Password) != "" &&
		cfg.Port > 0

	preflight := NaivePreflightResult{OK: true}
	if hasConfig {
		realityListen, realityPort := detectRealityBinding()
		preflight = p.naiveService.ValidatePreflight(NaivePreflightRequest{
			RealityListen: realityListen,
			RealityPort:   realityPort,
			NaiveListen:   "0.0.0.0",
			NaivePort:     cfg.Port,
			Domain:        cfg.Domain,
			EnableNaive:   true,
		})
	}

	reasons := make([]string, 0)
	if state == nil {
		reasons = append(reasons, "stateUnavailable")
	} else {
		if state.State != NaivePluginStateHealthy {
			reasons = append(reasons, "stateNotHealthy")
		}
		if !state.Enabled {
			reasons = append(reasons, "pluginDisabled")
		}
	}
	if !hasConfig {
		reasons = append(reasons, "missingConfig")
	}
	if !hasSubClient {
		reasons = append(reasons, "noSubClient")
	}
	if !preflight.OK {
		reasons = append(reasons, "preflightFailed")
	}
	publishReady := state != nil &&
		state.Enabled &&
		state.State == NaivePluginStateHealthy &&
		hasConfig &&
		hasSubClient &&
		preflight.OK

	sampleLink := ""
	if publishReady {
		sampleLink = fmt.Sprintf("naive+https://%s:%s@%s:%d", cfg.Username, cfg.Password, cfg.Domain, cfg.Port)
	}

	pluginJSONObj(c, gin.H{
		"subId":            req.SubID,
		"hasSubClient":     hasSubClient,
		"naiveInNormalSub": publishReady,
		"naiveInJsonSub":   publishReady,
		"sampleNaiveLink":  sampleLink,
		"state":            state,
		"configPresent":    hasConfig,
		"whyNotPublished":  reasons,
		"preflight":        preflight,
	})
}

func detectRealityBinding() (string, int) {
	inboundService := webservice.InboundService{}
	inbounds, err := inboundService.GetAllInbounds()
	if err != nil {
		return "", 0
	}
	for _, inbound := range inbounds {
		if inbound == nil || !inbound.Enable || !strings.EqualFold(string(inbound.Protocol), "vless") {
			continue
		}
		stream := map[string]any{}
		if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
			continue
		}
		sec, _ := stream["security"].(string)
		if strings.EqualFold(sec, "reality") {
			return inbound.Listen, inbound.Port
		}
	}
	return "", 0
}

func pluginJSONObj(c *gin.Context, obj any) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj":     obj,
	})
}

func pluginJSONMsg(c *gin.Context, msg string, err error) {
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     msg,
			"obj":     err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     msg,
	})
}
