package plugins

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/config"
)

const (
	pluginSourceManifestURL = "manifest_url"
	pluginSourceGitRepo     = "git_repo"
	pluginSourceZip         = "zip"
	pluginSourcePath        = "path"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

func (m *Manager) InstallExternalFromSource(ctx context.Context, sourceType, source string) ([]string, error) {
	ids, _, err := m.InstallExternalFromSourceWithWarnings(ctx, sourceType, source)
	return ids, err
}

func (m *Manager) PrecheckExternalSource(ctx context.Context, sourceType, source string) ([]string, []string, error) {
	m.installMu.Lock()
	defer m.installMu.Unlock()

	source = strings.TrimSpace(source)
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if source == "" {
		return nil, nil, fmt.Errorf("plugin source is required")
	}

	manifests, baseDir, cleanup, err := m.resolveSource(ctx, sourceType, source)
	if err != nil {
		return nil, nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if len(manifests) == 0 {
		return nil, nil, fmt.Errorf("no plugins found in source")
	}

	for i := range manifests {
		if err := sanitizeManifest(&manifests[i], baseDir); err != nil {
			return nil, nil, err
		}
	}
	warnings, err := m.validateNewManifests(manifests)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(manifests))
	for _, cfg := range manifests {
		ids = append(ids, cfg.ID)
	}
	return ids, warnings, nil
}

func (m *Manager) InstallExternalFromSourceWithWarnings(ctx context.Context, sourceType, source string) ([]string, []string, error) {
	m.installMu.Lock()
	defer m.installMu.Unlock()

	source = strings.TrimSpace(source)
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if source == "" {
		return nil, nil, fmt.Errorf("plugin source is required")
	}

	manifests, baseDir, cleanup, err := m.resolveSource(ctx, sourceType, source)
	if err != nil {
		return nil, nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if len(manifests) == 0 {
		return nil, nil, fmt.Errorf("no plugins found in source")
	}

	for i := range manifests {
		if err := sanitizeManifest(&manifests[i], baseDir); err != nil {
			return nil, nil, err
		}
	}
	warnings, err := m.validateNewManifests(manifests)
	if err != nil {
		return nil, nil, err
	}

	started := make([]*externalPlugin, 0, len(manifests))
	for _, cfg := range manifests {
		plugin := newExternalPlugin(cfg)
		if err := plugin.OnStart(ctx); err != nil {
			for _, p := range started {
				_ = p.OnStop(context.Background())
			}
			return nil, nil, fmt.Errorf("plugin %s failed to start: %w", cfg.ID, err)
		}
		started = append(started, plugin)
	}

	if err := m.appendPersistedManifests(manifests); err != nil {
		for _, p := range started {
			_ = p.OnStop(context.Background())
		}
		return nil, nil, err
	}

	ids := make([]string, 0, len(manifests))
	for i, cfg := range manifests {
		m.externalPlugins[cfg.ID] = started[i]
		ids = append(ids, cfg.ID)
	}
	return ids, warnings, nil
}

func (m *Manager) validateNewManifests(manifests []ExternalPluginManifest) ([]string, error) {
	seen := map[string]struct{}{}
	warnings := make([]string, 0)
	for _, cfg := range manifests {
		if cfg.ID == "" {
			return nil, fmt.Errorf("plugin id is required")
		}
		if !pluginIDPattern.MatchString(cfg.ID) {
			return nil, fmt.Errorf("plugin id %q is invalid", cfg.ID)
		}
		if _, exists := seen[cfg.ID]; exists {
			return nil, fmt.Errorf("duplicate plugin id %q in source", cfg.ID)
		}
		seen[cfg.ID] = struct{}{}
		if _, exists := m.externalPlugins[cfg.ID]; exists {
			return nil, fmt.Errorf("plugin %q already installed", cfg.ID)
		}
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, fmt.Errorf("plugin %q baseUrl is required", cfg.ID)
		}
		u, err := url.Parse(cfg.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("plugin %q has invalid baseUrl", cfg.ID)
		}
		if cfg.Command != "" && !filepath.IsAbs(cfg.Command) {
			return nil, fmt.Errorf("plugin %q command must resolve to absolute path", cfg.ID)
		}
		if !isPrivateOrLocalHost(u.Hostname()) && strings.TrimSpace(cfg.AuthToken) == "" {
			warnings = append(warnings, fmt.Sprintf("plugin %q uses non-local baseUrl without authToken", cfg.ID))
		}
	}
	return warnings, nil
}

func isPrivateOrLocalHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

func sanitizeManifest(cfg *ExternalPluginManifest, baseDir string) error {
	cfg.ID = strings.TrimSpace(cfg.ID)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Command = strings.TrimSpace(cfg.Command)
	cfg.WorkingDir = strings.TrimSpace(cfg.WorkingDir)

	if cfg.Command != "" && !filepath.IsAbs(cfg.Command) {
		if baseDir == "" {
			return fmt.Errorf("plugin %q command must be absolute", cfg.ID)
		}
		cfg.Command = filepath.Join(baseDir, cfg.Command)
	}
	if cfg.WorkingDir != "" && !filepath.IsAbs(cfg.WorkingDir) {
		if baseDir == "" {
			return fmt.Errorf("plugin %q workingDir must be absolute", cfg.ID)
		}
		cfg.WorkingDir = filepath.Join(baseDir, cfg.WorkingDir)
	}
	if cfg.WorkingDir == "" && baseDir != "" {
		cfg.WorkingDir = baseDir
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	return nil
}

func (m *Manager) appendPersistedManifests(extra []ExternalPluginManifest) error {
	manifestPath := m.manifestPath
	if strings.TrimSpace(manifestPath) == "" {
		manifestPath = filepath.Join(config.GetDBFolderPath(), "plugins", "manifest.json")
		m.manifestPath = manifestPath
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return err
	}

	existing := []ExternalPluginManifest{}
	data, err := os.ReadFile(manifestPath)
	if err == nil && strings.TrimSpace(string(data)) != "" {
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("invalid existing plugin manifest: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	existing = append(existing, extra...)
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	tmp := manifestPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, manifestPath)
}

func (m *Manager) resolveSource(ctx context.Context, sourceType, source string) ([]ExternalPluginManifest, string, func(), error) {
	switch sourceType {
	case pluginSourceManifestURL:
		return manifestsFromManifestSource(source)
	case pluginSourceGitRepo:
		return manifestsFromGit(ctx, source)
	case pluginSourceZip:
		return manifestsFromZip(source)
	case pluginSourcePath:
		return manifestsFromLocalPath(source)
	default:
		return nil, "", nil, fmt.Errorf("unsupported plugin source type: %s", sourceType)
	}
}

func manifestsFromManifestSource(source string) ([]ExternalPluginManifest, string, func(), error) {
	if isHTTPSURL(source) {
		data, err := downloadHTTPS(source)
		if err != nil {
			return nil, "", nil, err
		}
		manifests, err := decodeManifestBytes(data)
		return manifests, "", nil, err
	}
	path := filepath.Clean(source)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", nil, err
	}
	manifests, err := decodeManifestBytes(data)
	return manifests, filepath.Dir(path), nil, err
}

func manifestsFromGit(ctx context.Context, source string) ([]ExternalPluginManifest, string, func(), error) {
	repoURL, subPath := splitGitSource(source)
	tmp, err := os.MkdirTemp("", "plugin-git-*")
	if err != nil {
		return nil, "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, "", nil, fmt.Errorf("git clone failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	targetDir := tmp
	if subPath != "" {
		cleanSub := filepath.Clean(subPath)
		if cleanSub == "." || strings.HasPrefix(cleanSub, "..") {
			cleanup()
			return nil, "", nil, fmt.Errorf("invalid git sub path: %s", subPath)
		}
		targetDir = filepath.Join(tmp, cleanSub)
		if _, err := os.Stat(targetDir); err != nil {
			cleanup()
			return nil, "", nil, fmt.Errorf("git sub path not found: %s", subPath)
		}
	}
	manifests, baseDir, err := manifestsFromDirectory(targetDir)
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}
	return manifests, baseDir, cleanup, nil
}

func splitGitSource(source string) (repoURL string, subPath string) {
	source = strings.TrimSpace(source)
	idx := strings.Index(source, "#")
	if idx < 0 {
		return source, ""
	}
	repoURL = strings.TrimSpace(source[:idx])
	subPath = strings.TrimSpace(source[idx+1:])
	return repoURL, subPath
}

func manifestsFromZip(source string) ([]ExternalPluginManifest, string, func(), error) {
	workDir, err := os.MkdirTemp("", "plugin-zip-*")
	if err != nil {
		return nil, "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(workDir) }

	zipPath := source
	if isHTTPSURL(source) {
		data, err := downloadHTTPS(source)
		if err != nil {
			cleanup()
			return nil, "", nil, err
		}
		zipPath = filepath.Join(workDir, "source.zip")
		if err := os.WriteFile(zipPath, data, 0o644); err != nil {
			cleanup()
			return nil, "", nil, err
		}
	} else {
		zipPath = filepath.Clean(source)
	}

	extracted := filepath.Join(workDir, "extracted")
	if err := unzipSafe(zipPath, extracted); err != nil {
		cleanup()
		return nil, "", nil, err
	}
	manifests, baseDir, err := manifestsFromDirectory(extracted)
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}
	return manifests, baseDir, cleanup, nil
}

func manifestsFromLocalPath(source string) ([]ExternalPluginManifest, string, func(), error) {
	path := filepath.Clean(source)
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", nil, err
	}
	if info.IsDir() {
		manifests, baseDir, err := manifestsFromDirectory(path)
		return manifests, baseDir, nil, err
	}
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		return manifestsFromZip(path)
	}
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, "", nil, err
		}
		manifests, err := decodeManifestBytes(data)
		return manifests, filepath.Dir(path), nil, err
	}
	return nil, "", nil, fmt.Errorf("unsupported local plugin source: %s", path)
}

func manifestsFromDirectory(dir string) ([]ExternalPluginManifest, string, error) {
	manifestPath, err := findManifestFile(dir)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}
	manifests, err := decodeManifestBytes(data)
	if err != nil {
		return nil, "", err
	}
	return manifests, filepath.Dir(manifestPath), nil
}

func findManifestFile(root string) (string, error) {
	candidates := []string{
		filepath.Join(root, "manifest.json"),
		filepath.Join(root, "plugins", "manifest.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(root, e.Name(), "manifest.json")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("manifest.json not found in source")
}

func decodeManifestBytes(data []byte) ([]ExternalPluginManifest, error) {
	var list []ExternalPluginManifest
	if err := json.Unmarshal(data, &list); err == nil {
		return list, nil
	}
	var single ExternalPluginManifest
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest json")
	}
	return []ExternalPluginManifest{single}, nil
}

func isHTTPSURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func downloadHTTPS(addr string) ([]byte, error) {
	if !isHTTPSURL(addr) {
		return nil, fmt.Errorf("only https URLs are allowed: %s", addr)
	}
	resp, err := http.Get(addr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 20<<20))
}

func unzipSafe(zipPath, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	base := filepath.Clean(dst) + string(os.PathSeparator)

	for _, f := range r.File {
		target := filepath.Join(dst, f.Name)
		clean := filepath.Clean(target)
		if !strings.HasPrefix(clean, base) && clean != filepath.Clean(dst) {
			return fmt.Errorf("zip contains invalid path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(clean, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(clean, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			_ = src.Close()
			return err
		}
		if _, err := io.Copy(out, src); err != nil {
			_ = out.Close()
			_ = src.Close()
			return err
		}
		_ = out.Close()
		_ = src.Close()
	}
	return nil
}
