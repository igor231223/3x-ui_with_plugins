package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServerPlugin describes an extension that can register server API routes.
type ServerPlugin interface {
	ID() string
	RegisterServerRoutes(serverGroup *gin.RouterGroup)
}

// Status describes a plugin's runtime state in a common shape.
type Status struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Installed    bool   `json:"installed"`
	Enabled      bool   `json:"enabled"`
	CanToggle    bool   `json:"canToggle"`
	CanReinstall bool   `json:"canReinstall"`
	CanUninstall bool   `json:"canUninstall"`
	CanConfigure bool   `json:"canConfigure"`
	Health       string `json:"health"`
	Message      string `json:"message,omitempty"`
}

// Action describes an executable plugin action for dynamic UI/API.
type Action struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	NeedsBody   bool   `json:"needsBody"`
}

type UIField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     any    `json:"default,omitempty"`
}

type UISchema struct {
	Title        string    `json:"title"`
	SubmitAction string    `json:"submitAction"`
	Fields       []UIField `json:"fields"`
}

type InboundProtocol struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	PluginID       string    `json:"pluginId,omitempty"`
	SupportsStream bool      `json:"supportsStream"`
	ClientIDKey    string    `json:"clientIdKey,omitempty"`
	Fields         []UIField `json:"fields,omitempty"`
}

// LifecyclePlugin supports optional startup/shutdown hooks.
type LifecyclePlugin interface {
	OnStart(ctx context.Context) error
	OnStop(ctx context.Context) error
}

// StatusPlugin supports optional status reporting for registry API.
type StatusPlugin interface {
	Status(ctx context.Context) Status
}

// TogglePlugin supports generic enable/disable operations.
type TogglePlugin interface {
	GetEnabled(ctx context.Context) (bool, error)
	SetEnabled(ctx context.Context, enabled bool) error
}

// ReinstallPlugin supports generic reinstall/recover action.
type ReinstallPlugin interface {
	Reinstall(ctx context.Context) error
}

// UninstallPlugin supports generic uninstall action.
type UninstallPlugin interface {
	Uninstall(ctx context.Context, mode string) error
}

// ConfigurePlugin marks plugins that expose configurable settings.
type ConfigurePlugin interface {
	HasConfiguration(ctx context.Context) bool
}

type UISchemaPlugin interface {
	UISchema(ctx context.Context) (UISchema, error)
}

type InboundProtocolPlugin interface {
	InboundProtocols(ctx context.Context) ([]InboundProtocol, error)
}

type SubscriptionProvider interface {
	SubscriptionLinks(ctx context.Context, subID, host string) ([]string, error)
	SubscriptionJSON(ctx context.Context, subID, host string) ([]json.RawMessage, error)
}

// Manager keeps a list of server plugins and mounts their routes.
type Manager struct {
	serverPlugins   []ServerPlugin
	externalPlugins map[string]*externalPlugin
}

func NewManager(serverPlugins ...ServerPlugin) *Manager {
	return &Manager{
		serverPlugins:   serverPlugins,
		externalPlugins: map[string]*externalPlugin{},
	}
}

func (m *Manager) LoadExternalPlugins(manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var manifests []ExternalPluginManifest
	if err := json.Unmarshal(data, &manifests); err != nil {
		return err
	}
	for _, cfg := range manifests {
		if cfg.ID == "" || cfg.BaseURL == "" {
			continue
		}
		m.externalPlugins[cfg.ID] = newExternalPlugin(cfg)
	}
	return nil
}

func (m *Manager) RegisterServerRoutes(serverGroup *gin.RouterGroup) {
	for _, plugin := range m.serverPlugins {
		plugin.RegisterServerRoutes(serverGroup)
	}
}

func (m *Manager) Start(ctx context.Context) error {
	for _, plugin := range m.serverPlugins {
		lifecyclePlugin, ok := plugin.(LifecyclePlugin)
		if !ok {
			continue
		}
		if err := lifecyclePlugin.OnStart(ctx); err != nil {
			return err
		}
	}
	for _, plugin := range m.externalPlugins {
		if err := plugin.OnStart(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	for _, plugin := range m.externalPlugins {
		if err := plugin.OnStop(ctx); err != nil {
			return err
		}
	}
	for i := len(m.serverPlugins) - 1; i >= 0; i-- {
		lifecyclePlugin, ok := m.serverPlugins[i].(LifecyclePlugin)
		if !ok {
			continue
		}
		if err := lifecyclePlugin.OnStop(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Registry(ctx context.Context) []Status {
	statuses := make([]Status, 0, len(m.serverPlugins)+len(m.externalPlugins))
	for _, plugin := range m.serverPlugins {
		if statusPlugin, ok := plugin.(StatusPlugin); ok {
			status := statusPlugin.Status(ctx)
			_, canToggle := plugin.(TogglePlugin)
			_, canReinstall := plugin.(ReinstallPlugin)
			_, canUninstall := plugin.(UninstallPlugin)
			configurePlugin, hasConfigure := plugin.(ConfigurePlugin)
			status.CanToggle = canToggle
			status.CanReinstall = canReinstall
			status.CanUninstall = canUninstall
			status.CanConfigure = hasConfigure && configurePlugin.HasConfiguration(ctx)
			statuses = append(statuses, status)
			continue
		}
		_, canToggle := plugin.(TogglePlugin)
		_, canReinstall := plugin.(ReinstallPlugin)
		_, canUninstall := plugin.(UninstallPlugin)
		configurePlugin, hasConfigure := plugin.(ConfigurePlugin)
		statuses = append(statuses, Status{
			ID:           plugin.ID(),
			Version:      "unknown",
			Installed:    true,
			Enabled:      true,
			CanToggle:    canToggle,
			CanReinstall: canReinstall,
			CanUninstall: canUninstall,
			CanConfigure: hasConfigure && configurePlugin.HasConfiguration(ctx),
			Health:       "unknown",
		})
	}
	for _, plugin := range m.externalPlugins {
		statuses = append(statuses, plugin.Status(ctx))
	}
	return statuses
}

func (m *Manager) SetEnabled(ctx context.Context, pluginID string, enabled bool) error {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.SetEnabled(ctx, enabled)
	}
	for _, plugin := range m.serverPlugins {
		if plugin.ID() != pluginID {
			continue
		}
		togglePlugin, ok := plugin.(TogglePlugin)
		if !ok {
			return fmt.Errorf("plugin %s does not support enable/disable", pluginID)
		}
		return togglePlugin.SetEnabled(ctx, enabled)
	}
	return fmt.Errorf("plugin %s not found", pluginID)
}

func (m *Manager) GetStatus(ctx context.Context, pluginID string) (Status, error) {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.Status(ctx), nil
	}
	for _, plugin := range m.serverPlugins {
		if plugin.ID() != pluginID {
			continue
		}
		if statusPlugin, ok := plugin.(StatusPlugin); ok {
			status := statusPlugin.Status(ctx)
			_, canToggle := plugin.(TogglePlugin)
			_, canReinstall := plugin.(ReinstallPlugin)
			_, canUninstall := plugin.(UninstallPlugin)
			configurePlugin, hasConfigure := plugin.(ConfigurePlugin)
			status.CanToggle = canToggle
			status.CanReinstall = canReinstall
			status.CanUninstall = canUninstall
			status.CanConfigure = hasConfigure && configurePlugin.HasConfiguration(ctx)
			return status, nil
		}
		_, canToggle := plugin.(TogglePlugin)
		_, canReinstall := plugin.(ReinstallPlugin)
		_, canUninstall := plugin.(UninstallPlugin)
		configurePlugin, hasConfigure := plugin.(ConfigurePlugin)
		return Status{
			ID:           plugin.ID(),
			Version:      "unknown",
			Installed:    true,
			Enabled:      true,
			CanToggle:    canToggle,
			CanReinstall: canReinstall,
			CanUninstall: canUninstall,
			CanConfigure: hasConfigure && configurePlugin.HasConfiguration(ctx),
			Health:       "unknown",
		}, nil
	}
	return Status{}, fmt.Errorf("plugin %s not found", pluginID)
}

func (m *Manager) Reinstall(ctx context.Context, pluginID string) error {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.Reinstall(ctx)
	}
	for _, plugin := range m.serverPlugins {
		if plugin.ID() != pluginID {
			continue
		}
		reinstallPlugin, ok := plugin.(ReinstallPlugin)
		if !ok {
			return fmt.Errorf("plugin %s does not support reinstall", pluginID)
		}
		return reinstallPlugin.Reinstall(ctx)
	}
	return fmt.Errorf("plugin %s not found", pluginID)
}

func (m *Manager) Uninstall(ctx context.Context, pluginID, mode string) error {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.Uninstall(ctx, mode)
	}
	for _, plugin := range m.serverPlugins {
		if plugin.ID() != pluginID {
			continue
		}
		uninstallPlugin, ok := plugin.(UninstallPlugin)
		if !ok {
			return fmt.Errorf("plugin %s does not support uninstall", pluginID)
		}
		return uninstallPlugin.Uninstall(ctx, mode)
	}
	return fmt.Errorf("plugin %s not found", pluginID)
}

func (m *Manager) Actions(ctx context.Context, pluginID string) ([]Action, error) {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.Actions(ctx)
	}
	status, err := m.GetStatus(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	actions := make([]Action, 0, 4)
	if status.CanToggle {
		actions = append(actions, Action{ID: "setEnabled", Title: "Set enabled", Description: "Enable or disable plugin", NeedsBody: true})
	}
	if status.CanReinstall {
		actions = append(actions, Action{ID: "reinstall", Title: "Reinstall", Description: "Reinstall or recover plugin integration", NeedsBody: false})
	}
	if status.CanUninstall {
		actions = append(actions, Action{ID: "uninstall", Title: "Uninstall", Description: "Uninstall plugin integration", NeedsBody: true})
	}
	return actions, nil
}

func (m *Manager) ExecuteAction(ctx context.Context, pluginID, actionID string, payload map[string]any) error {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.ExecuteAction(ctx, actionID, payload)
	}
	switch actionID {
	case "setEnabled":
		enabled, _ := payload["enabled"].(bool)
		return m.SetEnabled(ctx, pluginID, enabled)
	case "reinstall":
		return m.Reinstall(ctx, pluginID)
	case "uninstall":
		mode, _ := payload["mode"].(string)
		return m.Uninstall(ctx, pluginID, mode)
	default:
		return fmt.Errorf("unsupported action %s for plugin %s", actionID, pluginID)
	}
}

func (m *Manager) UISchema(ctx context.Context, pluginID string) (UISchema, error) {
	if plugin, ok := m.externalPlugins[pluginID]; ok {
		return plugin.UISchema(ctx)
	}
	for _, plugin := range m.serverPlugins {
		if plugin.ID() != pluginID {
			continue
		}
		if schemaPlugin, ok := plugin.(UISchemaPlugin); ok {
			return schemaPlugin.UISchema(ctx)
		}
		return UISchema{}, fmt.Errorf("plugin %s does not provide ui-schema", pluginID)
	}
	return UISchema{}, fmt.Errorf("plugin %s not found", pluginID)
}

func (m *Manager) InboundProtocols(ctx context.Context) []InboundProtocol {
	protocols := make([]InboundProtocol, 0)
	for _, plugin := range m.serverPlugins {
		protocolPlugin, ok := plugin.(InboundProtocolPlugin)
		if !ok {
			continue
		}
		items, err := protocolPlugin.InboundProtocols(ctx)
		if err != nil {
			continue
		}
		for i := range items {
			if items[i].PluginID == "" {
				items[i].PluginID = plugin.ID()
			}
		}
		protocols = append(protocols, items...)
	}
	for pluginID, plugin := range m.externalPlugins {
		items, err := plugin.InboundProtocols(ctx)
		if err != nil {
			continue
		}
		for i := range items {
			if items[i].PluginID == "" {
				items[i].PluginID = pluginID
			}
		}
		protocols = append(protocols, items...)
	}
	return protocols
}

func (m *Manager) protocolOwner(ctx context.Context, protocol string) (string, bool) {
	target := strings.TrimSpace(strings.ToLower(protocol))
	if target == "" {
		return "", false
	}
	for _, p := range m.InboundProtocols(ctx) {
		if strings.ToLower(strings.TrimSpace(p.ID)) == target {
			return p.PluginID, p.PluginID != ""
		}
	}
	return "", false
}

func (m *Manager) ProtocolOwner(ctx context.Context, protocol string) (string, bool) {
	return m.protocolOwner(ctx, protocol)
}

func (m *Manager) IsPluginProtocol(ctx context.Context, protocol string) bool {
	_, ok := m.protocolOwner(ctx, protocol)
	return ok
}

func (m *Manager) ClientIDKey(ctx context.Context, protocol string) string {
	target := strings.TrimSpace(strings.ToLower(protocol))
	if target == "" {
		return ""
	}
	for _, p := range m.InboundProtocols(ctx) {
		if strings.ToLower(strings.TrimSpace(p.ID)) == target {
			return strings.TrimSpace(strings.ToLower(p.ClientIDKey))
		}
	}
	return ""
}

func (m *Manager) SubscriptionLinks(ctx context.Context, subID, host string) []string {
	links := make([]string, 0)
	for _, plugin := range m.serverPlugins {
		provider, ok := plugin.(SubscriptionProvider)
		if !ok {
			continue
		}
		items, err := provider.SubscriptionLinks(ctx, subID, host)
		if err != nil {
			continue
		}
		links = append(links, items...)
	}
	for _, plugin := range m.externalPlugins {
		items, err := plugin.SubscriptionLinks(ctx, subID, host)
		if err != nil {
			continue
		}
		links = append(links, items...)
	}
	return links
}

func (m *Manager) SubscriptionJSON(ctx context.Context, subID, host string) []json.RawMessage {
	items := make([]json.RawMessage, 0)
	for _, plugin := range m.serverPlugins {
		provider, ok := plugin.(SubscriptionProvider)
		if !ok {
			continue
		}
		part, err := provider.SubscriptionJSON(ctx, subID, host)
		if err != nil {
			continue
		}
		items = append(items, part...)
	}
	for _, plugin := range m.externalPlugins {
		part, err := plugin.SubscriptionJSON(ctx, subID, host)
		if err != nil {
			continue
		}
		items = append(items, part...)
	}
	return items
}
