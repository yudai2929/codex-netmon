package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yudai2929/codex-netmon/dashboard"
)

func main() {
	outputPath := flag.String("output", "grafana/dashboards/codex-netmon.json", "Grafana dashboard JSON output path")
	flag.Parse()
	if err := (dashboard.Generator{OutputPath: *outputPath}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
