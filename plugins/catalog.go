package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CatalogEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	SourceType       string `json:"sourceType"`
	Source           string `json:"source"`
	ManifestPathHint string `json:"manifestPathHint,omitempty"`
	ReadmePresent    bool   `json:"readmePresent"`
}

func (m *Manager) CatalogFromSource(ctx context.Context, source string) ([]CatalogEntry, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("catalog source is required")
	}
	root, repoURL, cleanup, err := resolveCatalogRoot(ctx, source)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	pluginsDir := root
	if st, err := os.Stat(filepath.Join(root, "plugins")); err == nil && st.IsDir() {
		pluginsDir = filepath.Join(root, "plugins")
	}
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogEntry, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pdir := filepath.Join(pluginsDir, e.Name())
		manifest := filepath.Join(pdir, "manifest.json")
		if _, err := os.Stat(manifest); err != nil {
			continue
		}
		manifests, _, _, err := manifestsFromManifestSource(manifest)
		if err != nil || len(manifests) == 0 {
			continue
		}
		id := strings.TrimSpace(manifests[0].ID)
		if id == "" {
			id = e.Name()
		}
		name, desc, hasReadme := extractReadmeInfo(filepath.Join(pdir, "README.md"))
		if name == "" {
			name = e.Name()
		}
		srcType := pluginSourcePath
		src := manifest
		if repoURL != "" {
			srcType = pluginSourceGitRepo
			src = repoURL + "#plugins/" + e.Name()
		}
		out = append(out, CatalogEntry{
			ID:               id,
			Name:             name,
			Description:      desc,
			SourceType:       srcType,
			Source:           src,
			ManifestPathHint: filepath.ToSlash(filepath.Join("plugins", e.Name(), "manifest.json")),
			ReadmePresent:    hasReadme,
		})
	}
	return out, nil
}

func resolveCatalogRoot(ctx context.Context, source string) (string, string, func(), error) {
	if isHTTPSURL(source) || strings.HasSuffix(strings.ToLower(source), ".git") || strings.Contains(source, "github.com/") {
		repoURL, _ := splitGitSource(source)
		tmp, err := os.MkdirTemp("", "plugin-catalog-*")
		if err != nil {
			return "", "", nil, err
		}
		cleanup := func() { _ = os.RemoveAll(tmp) }
		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, tmp)
		if out, err := cmd.CombinedOutput(); err != nil {
			cleanup()
			return "", "", nil, fmt.Errorf("git clone failed: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		return tmp, repoURL, cleanup, nil
	}
	local := filepath.Clean(source)
	st, err := os.Stat(local)
	if err != nil {
		return "", "", nil, err
	}
	if !st.IsDir() {
		return "", "", nil, fmt.Errorf("catalog source must be a repository directory or git url")
	}
	return local, "", nil, nil
}

func extractReadmeInfo(path string) (name string, desc string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	ok = true
	lines := strings.Split(string(data), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") && name == "" {
			name = strings.TrimSpace(strings.TrimLeft(line, "#"))
			continue
		}
		if desc == "" {
			desc = line
		}
		if name != "" && desc != "" {
			break
		}
	}
	return name, desc, ok
}
