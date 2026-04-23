package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ExternalPluginManifest struct {
	ID         string            `json:"id"`
	BaseURL    string            `json:"baseUrl"`
	AuthToken  string            `json:"authToken,omitempty"`
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	WorkingDir string            `json:"workingDir"`
	Env        map[string]string `json:"env"`
	Autostart  bool              `json:"autostart"`
}

type externalPlugin struct {
	cfg    ExternalPluginManifest
	client *http.Client
	cmd    *exec.Cmd
}

func newExternalPlugin(cfg ExternalPluginManifest) *externalPlugin {
	return &externalPlugin{
		cfg: cfg,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (p *externalPlugin) ID() string {
	return p.cfg.ID
}

func (p *externalPlugin) RegisterServerRoutes(_ *gin.RouterGroup) {
}

func (p *externalPlugin) OnStart(_ context.Context) error {
	if !p.cfg.Autostart || p.cfg.Command == "" {
		return nil
	}
	cmd := exec.Command(p.cfg.Command, p.cfg.Args...)
	if p.cfg.WorkingDir != "" {
		cmd.Dir = p.cfg.WorkingDir
	}
	if len(p.cfg.Env) > 0 {
		env := make([]string, 0, len(p.cfg.Env)+len(os.Environ()))
		env = append(env, os.Environ()...)
		for k, v := range p.cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = append(cmd.Env, env...)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	p.cmd = cmd
	return nil
}

func (p *externalPlugin) OnStop(_ context.Context) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *externalPlugin) Status(ctx context.Context) Status {
	var status Status
	if err := p.getJSON(ctx, "/status", &status); err != nil {
		return Status{
			ID:           p.cfg.ID,
			Version:      "unknown",
			Installed:    false,
			Enabled:      false,
			CanToggle:    false,
			CanReinstall: false,
			CanUninstall: false,
			CanConfigure: false,
			Health:       "degraded",
			Message:      err.Error(),
		}
	}
	return status
}

func (p *externalPlugin) SetEnabled(ctx context.Context, enabled bool) error {
	return p.postAction(ctx, "setEnabled", map[string]any{"enabled": enabled})
}

func (p *externalPlugin) Reinstall(ctx context.Context) error {
	return p.postAction(ctx, "reinstall", map[string]any{})
}

func (p *externalPlugin) Uninstall(ctx context.Context, mode string) error {
	return p.postAction(ctx, "uninstall", map[string]any{"mode": mode})
}

func (p *externalPlugin) Actions(ctx context.Context) ([]Action, error) {
	var actions []Action
	if err := p.getJSON(ctx, "/actions", &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

func (p *externalPlugin) ExecuteAction(ctx context.Context, actionID string, payload map[string]any) error {
	return p.postAction(ctx, actionID, payload)
}

func (p *externalPlugin) UISchema(ctx context.Context) (UISchema, error) {
	var schema UISchema
	if err := p.getJSON(ctx, "/ui-schema", &schema); err != nil {
		return UISchema{}, err
	}
	return schema, nil
}

func (p *externalPlugin) InboundProtocols(ctx context.Context) ([]InboundProtocol, error) {
	var protocols []InboundProtocol
	if err := p.getJSON(ctx, "/inbound-protocols", &protocols); err != nil {
		return nil, err
	}
	return protocols, nil
}

func (p *externalPlugin) SubscriptionLinks(ctx context.Context, subID, host string) ([]string, error) {
	var links []string
	if err := p.getJSON(ctx, fmt.Sprintf("/subscriptions/links?subId=%s&host=%s", urlQueryEscape(subID), urlQueryEscape(host)), &links); err != nil {
		return nil, err
	}
	return links, nil
}

func (p *externalPlugin) SubscriptionJSON(ctx context.Context, subID, host string) ([]json.RawMessage, error) {
	var result []json.RawMessage
	if err := p.getJSON(ctx, fmt.Sprintf("/subscriptions/json?subId=%s&host=%s", urlQueryEscape(subID), urlQueryEscape(host)), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *externalPlugin) MenuItems(ctx context.Context) ([]MenuItem, error) {
	var items []MenuItem
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.BaseURL+"/menu-items", nil)
	if err != nil {
		return nil, err
	}
	p.applyAuth(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return []MenuItem{}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("external plugin request failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (p *externalPlugin) ApplyRouting(ctx context.Context, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/routing/apply", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	p.applyAuth(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("external plugin routing apply failed: %s", resp.Status)
	}
	return nil
}

func urlQueryEscape(v string) string {
	r := strings.NewReplacer("%", "%25", "&", "%26", "=", "%3D", " ", "%20", "?", "%3F", "#", "%23")
	return r.Replace(v)
}

func (p *externalPlugin) postAction(ctx context.Context, actionID string, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/actions/"+actionID, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	p.applyAuth(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("external plugin action failed: %s", resp.Status)
	}
	return nil
}

func (p *externalPlugin) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.BaseURL+path, nil)
	if err != nil {
		return err
	}
	p.applyAuth(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("external plugin request failed: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (p *externalPlugin) applyAuth(req *http.Request) {
	if strings.TrimSpace(p.cfg.AuthToken) == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.cfg.AuthToken))
}
