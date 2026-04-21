package naive

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/config"
)

const (
	NaivePluginStateHealthy  = "healthy"
	NaivePluginStateDegraded = "degraded"
	NaivePluginStateMissing  = "missing"
)

type NaivePluginService struct{}

type NaivePreflightRequest struct {
	RealityListen string `json:"realityListen"`
	RealityPort   int    `json:"realityPort"`
	NaiveListen   string `json:"naiveListen"`
	NaivePort     int    `json:"naivePort"`
	Domain        string `json:"domain"`
	EnableNaive   bool   `json:"enableNaive"`
}

type NaivePreflightResult struct {
	OK          bool     `json:"ok"`
	Errors      []string `json:"errors"`
	Warnings    []string `json:"warnings"`
	Suggestions []string `json:"suggestions"`
}

type NaivePluginState struct {
	Installed         bool   `json:"installed"`
	Enabled           bool   `json:"enabled"`
	State             string `json:"state"`
	RequiresReinstall bool   `json:"requiresReinstall"`
	UIHookPresent     bool   `json:"uiHookPresent"`
	LastError         string `json:"lastError"`
	UpdatedAt         int64  `json:"updatedAt"`
}

type NaiveRuntimeConfig struct {
	Domain   string `json:"domain"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type NaiveRuntimeHealth struct {
	Ready             bool   `json:"ready"`
	RuntimeDir        string `json:"runtimeDir"`
	RuntimeConfigPath string `json:"runtimeConfigPath"`
	InstallScriptPath string `json:"installScriptPath"`
	ReadmePath        string `json:"readmePath"`
	CaddyfilePath     string `json:"caddyfilePath"`
	UnitFilePath      string `json:"unitFilePath"`
	ConfigExists      bool   `json:"configExists"`
	ScriptExists      bool   `json:"scriptExists"`
	ReadmeExists      bool   `json:"readmeExists"`
	CaddyfileExists   bool   `json:"caddyfileExists"`
	UnitFileExists    bool   `json:"unitFileExists"`
	CaddyBinaryExists bool   `json:"caddyBinaryExists"`
	PortReachable     bool   `json:"portReachable"`
	LastError         string `json:"lastError"`
}

type NaiveRuntimeInstallRunResult struct {
	Success   bool   `json:"success"`
	ExitCode  int    `json:"exitCode"`
	Output    string `json:"output"`
	Script    string `json:"script"`
	Timestamp int64  `json:"timestamp"`
}

type NaivePluginUninstallMode string

const (
	NaivePluginUninstallIntegrationOnly NaivePluginUninstallMode = "integrationOnly"
	NaivePluginUninstallDeleteAll       NaivePluginUninstallMode = "deleteAll"
)

func (s *NaivePluginService) ValidatePreflight(req NaivePreflightRequest) NaivePreflightResult {
	res := NaivePreflightResult{
		OK: true,
	}
	if !req.EnableNaive {
		return res
	}

	realityListen := normalizeListen(req.RealityListen)
	naiveListen := normalizeListen(req.NaiveListen)
	sharesAddress := sharesSameAddress(realityListen, naiveListen)

	if req.RealityPort == 443 && req.NaivePort == 443 && sharesAddress {
		res.OK = false
		res.Errors = append(res.Errors,
			"Unable to enable Naive Proxy: IP:443 is already used by VLESS Reality in stable mode.")
		res.Suggestions = append(res.Suggestions,
			"Use a second IP for Naive on 443, or switch Naive to a different port (for example 8443).")
	}

	if strings.TrimSpace(req.Domain) == "" {
		res.OK = false
		res.Errors = append(res.Errors,
			"Domain is required for Naive Proxy in production mode.")
		res.Suggestions = append(res.Suggestions,
			"Set a domain with correct DNS A/AAAA records and obtain a valid TLS certificate first.")
	}
	if req.NaivePort < 1 || req.NaivePort > 65535 {
		res.OK = false
		res.Errors = append(res.Errors, fmt.Sprintf("Invalid Naive port: %d", req.NaivePort))
	}
	if domain := strings.TrimSpace(req.Domain); domain != "" {
		ips, lookupErr := net.LookupHost(domain)
		if lookupErr != nil || len(ips) == 0 {
			res.OK = false
			res.Errors = append(res.Errors, fmt.Sprintf("Domain %q is not resolvable from server.", domain))
			res.Suggestions = append(res.Suggestions, "Verify DNS A/AAAA records and wait for propagation.")
		}
	}
	if conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", req.NaivePort), 350*time.Millisecond); dialErr == nil {
		_ = conn.Close()
		res.Warnings = append(res.Warnings, fmt.Sprintf("Port %d is already reachable on localhost.", req.NaivePort))
		res.Suggestions = append(res.Suggestions, "Ensure this listener is intended for naive-in; otherwise choose another port.")
	}

	if req.NaivePort != 443 {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("Naive is configured on non-standard TLS port %d.", req.NaivePort))
	}

	return res
}

func (s *NaivePluginService) GetState() (*NaivePluginState, error) {
	path := getNaivePluginStatePath()
	state := defaultNaivePluginState()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			state.State = NaivePluginStateMissing
			return state, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}
	if state.State == "" {
		state.State = NaivePluginStateHealthy
	}
	if state.Installed && !integrationHookExists() {
		state.State = NaivePluginStateDegraded
		state.RequiresReinstall = true
		state.UIHookPresent = false
		if state.LastError == "" {
			state.LastError = "Naive plugin integration hook is missing after update."
		}
		_ = s.SetState(state)
	}
	return state, nil
}

func (s *NaivePluginService) SetState(state *NaivePluginState) error {
	if state == nil {
		return fmt.Errorf("plugin state is nil")
	}
	state.UpdatedAt = time.Now().Unix()
	if state.State == "" {
		state.State = NaivePluginStateHealthy
	}

	path := getNaivePluginStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (s *NaivePluginService) ReinstallIntegration() (*NaivePluginState, error) {
	state, err := s.GetState()
	if err != nil {
		return nil, err
	}
	state.Installed = true
	state.UIHookPresent = true
	state.RequiresReinstall = false
	state.State = NaivePluginStateHealthy
	state.LastError = ""
	if err := ensureIntegrationHookMarker(); err != nil {
		state.State = NaivePluginStateDegraded
		state.RequiresReinstall = true
		state.UIHookPresent = false
		state.LastError = err.Error()
		if setErr := s.SetState(state); setErr != nil {
			return nil, setErr
		}
		return state, err
	}
	if err := s.SetState(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *NaivePluginService) Uninstall(mode NaivePluginUninstallMode) (*NaivePluginState, error) {
	if mode != NaivePluginUninstallDeleteAll && mode != NaivePluginUninstallIntegrationOnly {
		return nil, fmt.Errorf("unsupported uninstall mode: %s", mode)
	}
	if inUse, err := s.hasActiveRuntimeConnections(); err == nil && inUse {
		return nil, fmt.Errorf("naive runtime has active connections; disable clients and retry uninstall")
	}

	if mode == NaivePluginUninstallDeleteAll {
		root := filepath.Join(config.GetDBFolderPath(), "plugins", "naive")
		if err := os.RemoveAll(root); err != nil {
			return nil, err
		}
		_ = os.Remove(integrationHookMarkerPath())
		return defaultNaivePluginState(), nil
	}

	state, err := s.GetState()
	if err != nil {
		return nil, err
	}
	state.Enabled = false
	state.State = NaivePluginStateDegraded
	state.RequiresReinstall = true
	state.UIHookPresent = false
	state.LastError = "Integration was removed. Data is preserved."
	_ = os.Remove(integrationHookMarkerPath())
	if err := s.SetState(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *NaivePluginService) SetEnabled(enabled bool) (*NaivePluginState, error) {
	state, err := s.GetState()
	if err != nil {
		return nil, err
	}
	if !enabled {
		if inUse, inUseErr := s.hasActiveRuntimeConnections(); inUseErr == nil && inUse {
			return nil, fmt.Errorf("naive runtime has active connections; disable clients and retry")
		}
	}
	if enabled {
		cfg, cfgErr := s.GetRuntimeConfig()
		if cfgErr != nil {
			return nil, cfgErr
		}
		if cfg == nil ||
			strings.TrimSpace(cfg.Domain) == "" ||
			strings.TrimSpace(cfg.Username) == "" ||
			strings.TrimSpace(cfg.Password) == "" ||
			cfg.Port <= 0 {
			return nil, fmt.Errorf("naive runtime config is incomplete")
		}
	}
	state.Enabled = enabled
	if enabled && state.State == NaivePluginStateMissing {
		state.Installed = true
		state.State = NaivePluginStateHealthy
	}
	if err := s.SetState(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *NaivePluginService) GetRuntimeConfig() (*NaiveRuntimeConfig, error) {
	path := getNaivePluginRuntimeConfigPath()
	cfg := defaultNaiveRuntimeConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Port == 0 {
		cfg.Port = 443
	}
	return cfg, nil
}

func (s *NaivePluginService) SetRuntimeConfig(cfg *NaiveRuntimeConfig) (*NaiveRuntimeConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("runtime config is nil")
	}
	cfg.Domain = strings.TrimSpace(cfg.Domain)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Password = strings.TrimSpace(cfg.Password)
	if cfg.Port == 0 {
		cfg.Port = 443
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}

	path := getNaivePluginRuntimeConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *NaivePluginService) InstallRuntimeArtifacts() (*NaiveRuntimeHealth, error) {
	cfg, err := s.GetRuntimeConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil ||
		strings.TrimSpace(cfg.Domain) == "" ||
		strings.TrimSpace(cfg.Username) == "" ||
		strings.TrimSpace(cfg.Password) == "" ||
		cfg.Port <= 0 {
		return nil, fmt.Errorf("naive runtime config is incomplete")
	}

	runtimeDir := getNaiveRuntimeDir()
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, err
	}

	runtimeCfg := map[string]any{
		"type":           "naive-runtime",
		"domain":         cfg.Domain,
		"authPort":       443,
		"naiveInPort":    cfg.Port,
		"username":       cfg.Username,
		"password":       cfg.Password,
		"fallbackRoot":   getNaiveRuntimeFallbackDir(),
		"caddyfilePath":  getNaiveRuntimeCaddyfilePath(),
		"singboxCfgPath": getNaiveRuntimeSingboxConfigPath(),
		"updated":        time.Now().Unix(),
	}
	cfgPath := getNaiveRuntimeGeneratedConfigPath()
	cfgBytes, _ := json.MarshalIndent(runtimeCfg, "", "  ")
	if err := os.WriteFile(cfgPath, cfgBytes, 0o644); err != nil {
		return nil, err
	}

	authB64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.Username, cfg.Password)))
	caddyfile := fmt.Sprintf(`{
	admin off
}

:443, https://%s {
	tls {
		issuer acme {
			disable_http_challenge
		}
	}

	route {
		@naive {
			method CONNECT
			header Proxy-Authorization "Basic %s"
		}

		handle @naive {
			reverse_proxy h2c://127.0.0.1:%d {
				header_up Proxy-Authorization {header.Proxy-Authorization}
			}
		}

		handle {
			root * %s
			file_server
		}
	}
}
`, cfg.Domain, authB64, cfg.Port, getNaiveRuntimeFallbackDir())
	caddyfilePath := getNaiveRuntimeCaddyfilePath()
	if err := os.WriteFile(caddyfilePath, []byte(caddyfile), 0o644); err != nil {
		return nil, err
	}

	singboxCfg := fmt.Sprintf(`{
  "log": {
    "level": "warn",
    "output": "/var/log/sing-box/sing-box.log"
  },
  "inbounds": [
    {
      "type": "naive",
      "tag": "naive-in",
      "network": "tcp",
      "listen": "127.0.0.1",
      "listen_port": %d,
      "users": [
        {
          "username": "%s",
          "password": "%s"
        }
      ]
    }
  ],
  "outbounds": [
    {
      "type": "direct"
    }
  ]
}
`, cfg.Port, cfg.Username, cfg.Password)
	if err := os.WriteFile(getNaiveRuntimeSingboxConfigPath(), []byte(singboxCfg), 0o644); err != nil {
		return nil, err
	}

	unitFile := fmt.Sprintf(`[Unit]
Description=Naive Proxy (Caddy) managed by 3x-ui MVP
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/caddy run --config %s --adapter caddyfile
ExecReload=/usr/bin/caddy reload --config %s --adapter caddyfile
Restart=always
RestartSec=2
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, caddyfilePath, caddyfilePath)
	unitPath := getNaiveRuntimeSystemdUnitTemplatePath()
	if err := os.WriteFile(unitPath, []byte(unitFile), 0o644); err != nil {
		return nil, err
	}

	installScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
if [[ "$(uname -s)" != "Linux" ]]; then
  echo "[naive-runtime] This installer currently supports Linux only."
  exit 1
fi
if [[ $EUID -ne 0 ]]; then
  echo "[naive-runtime] Please run as root for systemd setup."
  exit 1
fi

detect_pm() {
  if command -v apt-get >/dev/null 2>&1; then echo "apt"; return; fi
  if command -v dnf >/dev/null 2>&1; then echo "dnf"; return; fi
  echo "unknown"
}

ensure_caddy() {
  if command -v caddy >/dev/null 2>&1; then return; fi
  PM=$(detect_pm)
  if [[ "$PM" == "apt" ]]; then
    apt-get update
    apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl gnupg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' > /etc/apt/sources.list.d/caddy-stable.list
    chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    chmod o+r /etc/apt/sources.list.d/caddy-stable.list
    apt-get update
    apt-get install -y caddy
    return
  fi
  echo "[naive-runtime] caddy not found and auto-install is supported only on apt-based distros."
  echo "Install Caddy manually: https://caddyserver.com/docs/install"
  exit 1
}

ensure_singbox() {
  if command -v sing-box >/dev/null 2>&1; then return; fi
  PM=$(detect_pm)
  if [[ "$PM" == "apt" ]]; then
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://sing-box.app/gpg.key -o /etc/apt/keyrings/sagernet.asc
    chmod a+r /etc/apt/keyrings/sagernet.asc
    cat >/etc/apt/sources.list.d/sagernet.sources <<'EOF'
Types: deb
URIs: https://deb.sagernet.org/
Suites: *
Components: *
Enabled: yes
Signed-By: /etc/apt/keyrings/sagernet.asc
EOF
    apt-get update
    apt-get install -y sing-box
    return
  fi
  echo "[naive-runtime] sing-box not found and auto-install is supported only on apt-based distros."
  echo "Install sing-box manually: https://sing-box.sagernet.org/installation/package-manager/"
  exit 1
}

check_443_conflict() {
  if ! command -v ss >/dev/null 2>&1; then
    return
  fi
  local listeners
  listeners=$(ss -ltnp '( sport = :443 )' 2>/dev/null || true)
  if [[ -z "$listeners" ]]; then
    return
  fi
  if echo "$listeners" | grep -qi "caddy"; then
    return
  fi
  echo "[naive-runtime] port 443 is already used by a non-caddy process."
  echo "[naive-runtime] Refusing to overwrite production listener."
  echo "[naive-runtime] Set NAIVE_FORCE=1 only if you really want to continue."
  if [[ "${NAIVE_FORCE:-0}" != "1" ]]; then
    exit 1
  fi
}

check_singbox_reality_conflict() {
  if [[ ! -f /etc/sing-box/config.json ]]; then
    return
  fi
  if grep -Eq "\"type\"[[:space:]]*:[[:space:]]*\"vless\"" /etc/sing-box/config.json && \
     grep -Eq "\"security\"[[:space:]]*:[[:space:]]*\"reality\"" /etc/sing-box/config.json && \
     grep -Eq "\"listen_port\"[[:space:]]*:[[:space:]]*443" /etc/sing-box/config.json; then
    echo "[naive-runtime] existing sing-box Reality listener on :443 detected."
    echo "[naive-runtime] Configure second IP or move one of listeners off 443."
    if [[ "${NAIVE_FORCE:-0}" != "1" ]]; then
      exit 1
    fi
  fi
}

ensure_caddy
ensure_singbox
check_443_conflict
check_singbox_reality_conflict

install -D -m 0644 "%s" "/etc/systemd/system/naive-proxy.service"
install -D -m 0644 "%s" "/etc/sing-box/config-naive.json"
install -D -m 0644 "%s" "/etc/caddy/Caddyfile"
mkdir -p "%s"
if [[ ! -f "%s/index.html" ]]; then
  cat >"%s/index.html" <<'EOF'
<!doctype html><html><head><meta charset="utf-8"><title>Welcome</title></head><body><h1>It works</h1></body></html>
EOF
fi
if [[ -f /etc/sing-box/config.json ]]; then
  cp -f /etc/sing-box/config.json /etc/sing-box/config.json.bak.naive
fi
if [[ -f /etc/caddy/Caddyfile ]]; then
  cp -f /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak.naive
fi
cp -f /etc/sing-box/config-naive.json /etc/sing-box/config.json
systemctl daemon-reload
systemctl enable sing-box || true
systemctl restart sing-box
systemctl enable caddy || true
systemctl reload caddy || systemctl restart caddy
systemctl enable --now naive-proxy.service
systemctl status naive-proxy.service --no-pager || true
systemctl status sing-box --no-pager || true
systemctl status caddy --no-pager || true
echo "[naive-runtime] Provisioning complete."
`, unitPath, getNaiveRuntimeSingboxConfigPath(), caddyfilePath, getNaiveRuntimeFallbackDir(), getNaiveRuntimeFallbackDir(), getNaiveRuntimeFallbackDir())
	scriptPath := getNaiveRuntimeInstallScriptPath()
	if err := os.WriteFile(scriptPath, []byte(installScript), 0o755); err != nil {
		return nil, err
	}

	readme := `Naive Runtime (MVP)

Files:
- runtime.json: generated runtime payload from panel config
- Caddyfile: Caddy config (CONNECT auth + h2c reverse_proxy)
- singbox-naive-server.json: sing-box naive inbound config
- naive-proxy.service: systemd unit template for Caddy runtime
- install-runtime.sh: Linux installer (installs deps, configures caddy + sing-box + systemd)

This directory is persisted under DB folder to survive panel updates.
`
	readmePath := getNaiveRuntimeReadmePath()
	if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
		return nil, err
	}

	state, sErr := s.GetState()
	if sErr == nil {
		state.Installed = true
		if state.State == NaivePluginStateMissing {
			state.State = NaivePluginStateHealthy
		}
		_ = s.SetState(state)
	}

	return s.GetRuntimeHealth()
}

func (s *NaivePluginService) GetRuntimeHealth() (*NaiveRuntimeHealth, error) {
	cfg, err := s.GetRuntimeConfig()
	if err != nil {
		return nil, err
	}
	h := &NaiveRuntimeHealth{
		RuntimeDir:        getNaiveRuntimeDir(),
		RuntimeConfigPath: getNaiveRuntimeGeneratedConfigPath(),
		InstallScriptPath: getNaiveRuntimeInstallScriptPath(),
		ReadmePath:        getNaiveRuntimeReadmePath(),
		CaddyfilePath:     getNaiveRuntimeCaddyfilePath(),
		UnitFilePath:      getNaiveRuntimeSystemdUnitTemplatePath(),
	}
	if _, err := os.Stat(h.RuntimeConfigPath); err == nil {
		h.ConfigExists = true
	}
	if _, err := os.Stat(h.InstallScriptPath); err == nil {
		h.ScriptExists = true
	}
	if _, err := os.Stat(h.ReadmePath); err == nil {
		h.ReadmeExists = true
	}
	if _, err := os.Stat(h.CaddyfilePath); err == nil {
		h.CaddyfileExists = true
	}
	if _, err := os.Stat(h.UnitFilePath); err == nil {
		h.UnitFileExists = true
	}
	if _, err := os.Stat("/usr/bin/caddy"); err == nil {
		h.CaddyBinaryExists = true
	}

	if cfg != nil && cfg.Port > 0 {
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port), 500*time.Millisecond)
		if dialErr == nil {
			h.PortReachable = true
			_ = conn.Close()
		}
	}

	h.Ready = h.ConfigExists && h.ScriptExists && h.ReadmeExists && h.CaddyfileExists && h.UnitFileExists
	if !h.Ready {
		h.LastError = "Naive runtime artifacts are missing. Run install runtime."
	}
	return h, nil
}

func (s *NaivePluginService) RunRuntimeInstaller() (*NaiveRuntimeInstallRunResult, error) {
	scriptPath := getNaiveRuntimeInstallScriptPath()
	if _, err := os.Stat(scriptPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("runtime install script not found; install runtime artifacts first")
		}
		return nil, err
	}

	result := &NaiveRuntimeInstallRunResult{
		Success:   false,
		ExitCode:  -1,
		Script:    scriptPath,
		Timestamp: time.Now().Unix(),
	}

	if runtime.GOOS != "linux" {
		result.Output = "runtime installer execution is supported on Linux only"
		return result, nil
	}

	cmd := exec.Command("bash", scriptPath)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	result.Output = output.String()
	if err == nil {
		result.Success = true
		result.ExitCode = 0
		return result, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.Output = strings.TrimSpace(result.Output + "\n" + err.Error())
	}
	return result, nil
}

func defaultNaivePluginState() *NaivePluginState {
	return &NaivePluginState{
		Installed:         false,
		Enabled:           false,
		State:             NaivePluginStateMissing,
		RequiresReinstall: false,
		UIHookPresent:     false,
	}
}

func getNaivePluginStatePath() string {
	return filepath.Join(config.GetDBFolderPath(), "plugins", "naive", "state.json")
}

func getNaivePluginRuntimeConfigPath() string {
	return filepath.Join(config.GetDBFolderPath(), "plugins", "naive", "runtime.json")
}

func getNaiveRuntimeDir() string {
	return filepath.Join(config.GetDBFolderPath(), "plugins", "naive", "runtime")
}

func getNaiveRuntimeGeneratedConfigPath() string {
	return filepath.Join(getNaiveRuntimeDir(), "runtime.json")
}

func getNaiveRuntimeInstallScriptPath() string {
	return filepath.Join(getNaiveRuntimeDir(), "install-runtime.sh")
}

func getNaiveRuntimeReadmePath() string {
	return filepath.Join(getNaiveRuntimeDir(), "README.txt")
}

func getNaiveRuntimeCaddyfilePath() string {
	return filepath.Join(getNaiveRuntimeDir(), "Caddyfile")
}

func getNaiveRuntimeSystemdUnitTemplatePath() string {
	return filepath.Join(getNaiveRuntimeDir(), "naive-proxy.service")
}

func getNaiveRuntimeSingboxConfigPath() string {
	return filepath.Join(getNaiveRuntimeDir(), "singbox-naive-server.json")
}

func getNaiveRuntimeFallbackDir() string {
	return filepath.Join(getNaiveRuntimeDir(), "fallback-site")
}

func integrationHookMarkerPath() string {
	return filepath.Join(getRunBaseDir(), "web", "assets", "js", "naive_plugin_hook.marker")
}

func integrationHookExists() bool {
	_, err := os.Stat(integrationHookMarkerPath())
	return err == nil
}

func ensureIntegrationHookMarker() error {
	path := integrationHookMarkerPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := []byte("naive plugin integration marker\n")
	return os.WriteFile(path, content, 0o644)
}

func getRunBaseDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	exeDir := filepath.Dir(exePath)
	exeDirLower := strings.ToLower(filepath.ToSlash(exeDir))
	if runtime.GOOS == "windows" &&
		(strings.Contains(exeDirLower, "/appdata/local/temp/") || strings.Contains(exeDirLower, "/go-build")) {
		wd, wErr := os.Getwd()
		if wErr == nil {
			return wd
		}
	}
	return exeDir
}

func normalizeListen(listen string) string {
	v := strings.TrimSpace(strings.ToLower(listen))
	switch v {
	case "", "0.0.0.0", "::", "::0":
		return "all"
	default:
		return v
	}
}

func defaultNaiveRuntimeConfig() *NaiveRuntimeConfig {
	return &NaiveRuntimeConfig{
		Port: 443,
	}
}

func sharesSameAddress(a, b string) bool {
	if a == "all" || b == "all" {
		return true
	}
	return a == b
}

func (s *NaivePluginService) hasActiveRuntimeConnections() (bool, error) {
	cfg, err := s.GetRuntimeConfig()
	if err != nil {
		return false, err
	}
	if cfg == nil || cfg.Port <= 0 {
		return false, nil
	}
	if runtime.GOOS != "linux" {
		return false, nil
	}
	if has, err := hasActiveTCPConnectionsLinux("/proc/net/tcp", cfg.Port); err == nil && has {
		return true, nil
	}
	return hasActiveTCPConnectionsLinux("/proc/net/tcp6", cfg.Port)
}

func hasActiveTCPConnectionsLinux(path string, port int) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	needle := fmt.Sprintf(":%04X", port)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "sl") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		localAddr := fields[1]
		state := fields[3]
		// 01 = ESTABLISHED
		if strings.HasSuffix(localAddr, needle) && state == "01" {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
