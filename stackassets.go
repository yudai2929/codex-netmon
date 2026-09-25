package main

import "embed"

//go:embed compose.yaml grafana/dashboards/codex-netmon.json grafana/provisioning/dashboards/provider.yaml
var localStackAssets embed.FS
