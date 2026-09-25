package localstack

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const composeProject = "codex-netmon"

type Config struct {
	GrafanaPort int
	OTLPPort    int
	ConfigDir   string
}

type Stack struct {
	Config Config
}

type Info struct {
	DashboardURL string
	OTLPEndpoint string
	StopCommand  string
}

func (stack Stack) Start(ctx context.Context, assets fs.FS, stdout, stderr io.Writer) (Info, error) {
	if err := stack.Config.validate(); err != nil {
		return Info{}, err
	}
	configDir, err := stack.Config.configDir()
	if err != nil {
		return Info{}, err
	}
	if err := installAssets(configDir, assets); err != nil {
		return Info{}, err
	}

	composePath := filepath.Join(configDir, "compose.yaml")
	command := exec.CommandContext(ctx, "docker", "compose", "--project-name", composeProject, "--file", composePath, "up", "-d")
	command.Dir = configDir
	command.Env = append(os.Environ(),
		fmt.Sprintf("GRAFANA_PORT=%d", stack.Config.GrafanaPort),
		fmt.Sprintf("OTLP_PORT=%d", stack.Config.OTLPPort),
	)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return Info{}, fmt.Errorf("start local LGTM stack with Docker Compose: %w", err)
	}

	dashboardURL := fmt.Sprintf("http://127.0.0.1:%d/d/codex-netmon/codex-netmon?from=now-24h&to=now", stack.Config.GrafanaPort)
	return Info{
		DashboardURL: dashboardURL,
		OTLPEndpoint: fmt.Sprintf("http://127.0.0.1:%d/v1/metrics", stack.Config.OTLPPort),
		StopCommand:  fmt.Sprintf("docker compose --project-name %s --file %s down", composeProject, shellQuote(composePath)),
	}, nil
}

func (config Config) validate() error {
	if config.GrafanaPort < 1 || config.GrafanaPort > 65535 {
		return fmt.Errorf("Grafana port must be between 1 and 65535")
	}
	if config.OTLPPort < 1 || config.OTLPPort > 65535 {
		return fmt.Errorf("OTLP port must be between 1 and 65535")
	}
	return nil
}

func (config Config) configDir() (string, error) {
	if config.ConfigDir != "" {
		return config.ConfigDir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(base, "codex-netmon", "local-stack"), nil
}

func installAssets(configDir string, assets fs.FS) error {
	files := []struct {
		name string
		mode fs.FileMode
	}{
		{name: "compose.yaml", mode: 0o644},
		{name: "grafana/dashboards/codex-netmon.json", mode: 0o644},
		{name: "grafana/provisioning/dashboards/provider.yaml", mode: 0o644},
	}
	for _, file := range files {
		data, err := fs.ReadFile(assets, file.name)
		if err != nil {
			return fmt.Errorf("read embedded local stack file %s: %w", file.name, err)
		}
		path := filepath.Join(configDir, file.name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create local stack directory: %w", err)
		}
		if err := os.WriteFile(path, data, file.mode); err != nil {
			return fmt.Errorf("write local stack file %s: %w", file.name, err)
		}
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
