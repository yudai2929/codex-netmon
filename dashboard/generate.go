package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Generator struct {
	OutputPath string
}

func (generator Generator) Run() error {
	if generator.OutputPath == "" {
		return fmt.Errorf("dashboard output path is required")
	}
	manifest, err := BuildNetmonDashboard()
	if err != nil {
		return fmt.Errorf("build Codex netmon dashboard: %w", err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal Codex netmon dashboard: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(generator.OutputPath), 0o755); err != nil {
		return fmt.Errorf("create dashboard directory: %w", err)
	}
	if err := os.WriteFile(generator.OutputPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write Codex netmon dashboard: %w", err)
	}
	return nil
}
