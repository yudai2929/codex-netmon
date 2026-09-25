package dashboard

import (
	"fmt"
	"strings"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	viz "github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboardv2"
	"github.com/grafana/grafana-foundation-sdk/go/piechart"
	"github.com/grafana/grafana-foundation-sdk/go/resource"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/table"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
	"github.com/grafana/grafana-foundation-sdk/go/units"
)

func wifiUsage(direction, interval string) string {
	return fmt.Sprintf("sum(increase(codex_wifi_%s_bytes_total[%s])) or vector(0)", direction, interval)
}

func wifiTotal(interval string) string {
	return fmt.Sprintf("(%s) + (%s)", wifiUsage("received", interval), wifiUsage("sent", interval))
}

func wifiUsageByKind(direction, kindMatcher, interval string) string {
	return fmt.Sprintf("sum(increase(codex_wifi_%s_bytes_total{%s}[%s])) or vector(0)", direction, kindMatcher, interval)
}

func wifiTotalByKind(kindMatcher, interval string) string {
	return fmt.Sprintf("(%s) + (%s)",
		wifiUsageByKind("received", kindMatcher, interval),
		wifiUsageByKind("sent", kindMatcher, interval))
}

func wifiMatrixValue(expr, source, direction string) string {
	return fmt.Sprintf(`label_replace(label_replace((%s), "source", %q, "__name__", ".*"), "direction", %q, "__name__", ".*")`, expr, source, direction)
}

func wifiMatrixQuery() string {
	type source struct {
		name, kindMatcher string
	}
	sources := []source{
		{name: "All Codex traffic"},
		{name: "Codex processes", kindMatcher: `process_kind!="command"`},
		{name: "Commands launched by Codex", kindMatcher: `process_kind="command"`},
	}
	var parts []string
	for _, source := range sources {
		received := wifiUsage("received", "$__range")
		sent := wifiUsage("sent", "$__range")
		total := wifiTotal("$__range")
		if source.kindMatcher != "" {
			received = wifiUsageByKind("received", source.kindMatcher, "$__range")
			sent = wifiUsageByKind("sent", source.kindMatcher, "$__range")
			total = wifiTotalByKind(source.kindMatcher, "$__range")
		}
		parts = append(parts,
			wifiMatrixValue(total, source.name, "Total"),
			wifiMatrixValue(received, source.name, "Received"),
			wifiMatrixValue(sent, source.name, "Sent"),
		)
	}
	return strings.Join(parts, " or ")
}

func BuildWiFiDashboard() (resource.Manifest, error) {
	definition := Definition{
		UID:         "codex-netmon-wifi",
		Title:       "Codex Wi-Fi usage",
		Description: "Wi-Fi usage attributed to Codex processes and commands launched by Codex, measured with macOS nettop. Change the time range in the upper right. Zero can mean no measurement in that range.",
		Tags:        []string{"codex", "wifi", "local"},
		Panels: []PanelDefinition{
			{
				ID: 1, Title: "Total Wi-Fi usage", Position: Position{0, 0, 6, 7},
				Description: "Bytes sent and received by Codex processes and their commands over the selected time range.",
				Visualization: stat.NewVisualizationV2Builder().Unit(units.BytesSI).
					ColorMode(viz.BigValueColorModeNone).NoValue("No measurements"),
				Queries: []cog.Builder[dashboardv2.PanelQueryKind]{
					promQuery(wifiTotal("$__range"), "A", "", true),
				},
			},
			{
				ID: 11, Title: "Usage breakdown", Position: Position{6, 0, 18, 7},
				Description:   "Wi-Fi bytes by source and direction over the selected time range. All Codex traffic is the sum of Codex processes and commands. Historical unclassified traffic is included with Codex processes.",
				Visualization: table.NewVisualizationV2Builder().Unit(units.BytesSI),
				Queries: []cog.Builder[dashboardv2.PanelQueryKind]{
					promTableQuery(wifiMatrixQuery(), "A"),
				},
				Transformations: []cog.Builder[dashboardv2.TransformationKind]{
					dashboardv2.NewTransformationBuilder().Group("groupingToMatrix").
						Options(map[string]any{"rowField": "source", "columnField": "direction", "valueField": "Value"}),
					dashboardv2.NewTransformationBuilder().Group("organize").
						Options(map[string]any{"renameByName": map[string]string{"source\\direction": "Source"}}),
				},
			},
			{
				ID: 7, Title: "Wi-Fi transfer rate", Position: Position{0, 23, 12, 9},
				Description: "Receive and send rate for Codex processes and their commands combined.",
				Visualization: timeseries.NewVisualizationV2Builder().Unit(units.BytesPerSecondSI).
					Legend(viz.NewVizLegendOptionsBuilder().ShowLegend(true)),
				Queries: []cog.Builder[dashboardv2.PanelQueryKind]{
					promQuery(`sum(rate(codex_wifi_received_bytes_total[1m])) or vector(0)`, "A", "Received", false),
					promQuery(`sum(rate(codex_wifi_sent_bytes_total[1m])) or vector(0)`, "B", "Sent", false),
				},
			},
			{
				ID: 9, Title: "Traffic over time by direction", Position: Position{0, 15, 24, 8},
				Description: "Estimated received, sent, and total bytes in each interval. The interval adapts to the selected time range.",
				Visualization: timeseries.NewVisualizationV2Builder().Unit(units.BytesSI).Min(0).
					Legend(viz.NewVizLegendOptionsBuilder().ShowLegend(true)),
				Queries: []cog.Builder[dashboardv2.PanelQueryKind]{
					promQuery(wifiUsage("received", "$__rate_interval"), "A", "Received", false),
					promQuery(wifiUsage("sent", "$__rate_interval"), "B", "Sent", false),
					promQuery(wifiTotal("$__rate_interval"), "C", "Total", false),
				},
			},
			{
				ID: 10, Title: "Traffic over time by source", Position: Position{0, 7, 24, 8},
				Description: "Bytes sent and received in each interval, split by source. Historical unclassified traffic is included with Codex processes.",
				Visualization: timeseries.NewVisualizationV2Builder().Unit(units.BytesSI).Min(0).
					Legend(viz.NewVizLegendOptionsBuilder().ShowLegend(true)),
				Queries: []cog.Builder[dashboardv2.PanelQueryKind]{
					promQuery(wifiTotalByKind(`process_kind!="command"`, "$__rate_interval"), "A", "Codex processes", false),
					promQuery(wifiTotalByKind(`process_kind="command"`, "$__rate_interval"), "B", "Commands launched by Codex", false),
				},
			},
			{
				ID: 8, Title: "Wi-Fi usage by program", Position: Position{12, 23, 12, 9},
				Description: "Share of Wi-Fi bytes sent and received by executable in the selected range, including commands launched by Codex.",
				Visualization: piechart.NewVisualizationV2Builder().Unit(units.BytesSI).
					PieType(piechart.PieChartTypePie).
					ReduceOptions(viz.NewReduceDataOptionsBuilder().Calcs([]string{"lastNotNull"})).
					Legend(piechart.NewPieChartLegendOptionsBuilder().ShowLegend(true).
						DisplayMode(viz.LegendDisplayModeTable).Placement(viz.LegendPlacementRight).
						Values([]piechart.PieChartLegendValues{piechart.PieChartLegendValuesValue, piechart.PieChartLegendValuesPercent})),
				Queries: []cog.Builder[dashboardv2.PanelQueryKind]{
					promQuery(`(sum by (process) (increase(codex_wifi_received_bytes_total[$__range])) + sum by (process) (increase(codex_wifi_sent_bytes_total[$__range]))) > 0`, "A", "{{process}}", true),
				},
			},
		},
	}
	return definition.Build()
}
