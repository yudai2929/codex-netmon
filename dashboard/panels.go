package dashboard

import (
	"fmt"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	viz "github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboardv2"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/resource"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
)

type Position struct {
	X, Y, Width, Height int64
}

type PanelDefinition struct {
	ID              int
	Title           string
	Description     string
	Position        Position
	Visualization   cog.Builder[dashboardv2.VizConfigKind]
	Queries         []cog.Builder[dashboardv2.PanelQueryKind]
	Transformations []cog.Builder[dashboardv2.TransformationKind]
	TimeFrom        string
}

type Definition struct {
	UID         string
	Title       string
	Description string
	Tags        []string
	Panels      []PanelDefinition
}

func (definition Definition) Build() (resource.Manifest, error) {
	grid := dashboardv2.Grid()
	builder := dashboardv2.NewDashboardBuilder(definition.Title).
		Description(definition.Description).
		Tags(definition.Tags).
		TimeSettings(dashboardv2.NewTimeSettingsBuilder().
			Timezone("browser").From("now-24h").To("now").AutoRefresh("30s"))

	for _, item := range definition.Panels {
		key := fmt.Sprintf("panel-%d", item.ID)
		queries := dashboardv2.NewQueryGroupBuilder()
		for _, query := range item.Queries {
			queries.Target(query)
		}
		for _, transformation := range item.Transformations {
			queries.Transformation(transformation)
		}
		if item.TimeFrom != "" {
			queries.QueryOptions(dashboardv2.NewQueryOptionsSpecBuilder().TimeFrom(item.TimeFrom))
		}
		builder.Panel(key, dashboardv2.NewPanelBuilder().
			Id(float64(item.ID)).Title(item.Title).Description(item.Description).
			Data(queries).Visualization(item.Visualization))
		grid.Item(dashboardv2.GridItem(key).
			X(item.Position.X).Y(item.Position.Y).
			Width(item.Position.Width).Height(item.Position.Height))
	}
	return dashboardv2.Manifest(definition.UID, builder.GridLayout(grid)).Build()
}

func promTableQuery(expr, refID string) *dashboardv2.TargetBuilder {
	query := prometheus.NewQueryV2Builder().
		Datasource(dashboardv2.NewDashboardv2DataQueryKindDatasourceBuilder().Name("prometheus")).
		Expr(expr).EditorMode(prometheus.QueryEditorModeCode).
		Format(prometheus.PromQueryFormatTable).Instant(true).Range(false)
	return dashboardv2.NewTargetBuilder().RefId(refID).Query(query)
}

func promQuery(expr, refID, legend string, instant bool) *dashboardv2.TargetBuilder {
	query := prometheus.NewQueryV2Builder().
		Datasource(dashboardv2.NewDashboardv2DataQueryKindDatasourceBuilder().Name("prometheus")).
		Expr(expr).EditorMode(prometheus.QueryEditorModeCode).
		Instant(instant).Range(!instant)
	if legend != "" {
		query.LegendFormat(legend)
	}
	return dashboardv2.NewTargetBuilder().RefId(refID).Query(query)
}

func statPanel(id int, position Position, title, expr, unit, description string) PanelDefinition {
	return PanelDefinition{
		ID: id, Title: title, Description: description, Position: position,
		Visualization: stat.NewVisualizationV2Builder().Unit(unit).ColorMode(viz.BigValueColorModeNone).NoValue("No measurements"),
		Queries:       []cog.Builder[dashboardv2.PanelQueryKind]{promQuery(expr, "A", "", true)},
	}
}
