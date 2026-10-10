// Package climate holds the rules of the climate plugin: from a relief, the yearly temperature and
// rain and, from them, the biome of every place, published as fields of the world.
package climate

import (
	"fmt"

	"github.com/akrck02/go-scene-builder/relief"
	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// Version of the plugin.
const Version = "1.0.0"

// Plugin returns the climate plugin.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{
		Name:    "climate",
		Version: Version,
		Summary: "Yearly temperature and rain over a relief, the biome of every place, and the turning seasons.",
		Rules:   []plugin.Definition{climateRule(), seasonsRule()},
	}
}

func climateRule() plugin.Definition {
	return plugin.Definition{
		Description: protocol.RuleDescription{
			Name: "climate",
			Summary: "Works out temperature (fields: temperature, °C), rain (precipitation, mm a year) and biome (biome: index into the " +
				"names of the \"biomes\" event, −1 for the sea) once, at the start.",
			Parameters: []protocol.ParameterDescription{
				{Name: "relief", Type: "text", Summary: "relief header (relieve.json with relieve.f32 next to it), km east and north"},
				{Name: "origin", Type: "list", Summary: "[east, north] of the world's origin: x = east − origin[0], z = origin[1] − north"},
				{Name: "cell", Type: "number", Default: 0.1, Summary: "km: cell of the fields"},
				{Name: "temperature", Type: "map", Summary: "{sea_level: °C, lapse: °C per km, northward: °C per 10 km}"},
				{Name: "rain", Type: "map", Summary: "{wind_from: degrees, background: mm, orographic: mm, …}"},
				{Name: "terrain", Type: "map", Summary: "{river_area: km², river_slope, coast_reach: km, coast_low: m, coast_slope}"},
				{Name: "biomes", Type: "list", Summary: "rules: [{biome, when: {temperature, precipitation, height, slope: {min, max}, river, coast}}]"},
				{Name: "regions", Type: "list", Summary: "[{biome, ellipse: {centre, radii} | polygon, when}]: the world's own biomes"},
			},
			Writes: []string{"temperature", "precipitation", "biome"},
		},
		New: newClimate,
	}
}

type climateRuleInstance struct {
	parameters plugin.Parameters
	done       bool
}

func newClimate(parameters plugin.Parameters) (plugin.Rule, error) {
	if parameters.Text("relief", "") == "" {
		return nil, fmt.Errorf("climate needs a relief")
	}
	return &climateRuleInstance{parameters: parameters}, nil
}

// Start works the climate out and publishes it.
func (rule *climateRuleInstance) Start(context *plugin.Context) error {
	rule.done = true
	return rule.publish(context)
}

// Step publishes the climate if Start did not (the climate does not change with time yet).
func (rule *climateRuleInstance) Step(context *plugin.Context) error {
	if rule.done {
		return nil
	}
	rule.done = true
	return rule.publish(context)
}

func (rule *climateRuleInstance) publish(context *plugin.Context) error {
	result, err := Compute(rule.parameters)
	if err != nil {
		return err
	}
	origin := pair(rule.parameters["origin"], [2]float64{0, 0})
	for _, field := range []struct {
		name string
		grid *Grid
	}{{"temperature", result.Temperature}, {"precipitation", result.Precipitation}, {"biome", result.Biome}} {
		context.SetField(fieldOf(field.name, field.grid, origin))
	}
	context.Emit("biomes", map[string]any{"names": result.Names})
	return nil
}

// Result is a worked-out climate.
type Result struct {
	Temperature, Precipitation, Biome *Grid
	Names                             []string
}

// Compute works out the climate from a rule's parameters (the plugin's work, without the host).
func Compute(parameters plugin.Parameters) (*Result, error) {
	grid, err := relief.Load(plugin.Path(parameters.Text("relief", "")))
	if err != nil {
		return nil, err
	}
	heights := gridOf(grid).resampled(parameters.Number("cell", 0.1))
	land := make([]bool, len(heights.Values))
	for index, height := range heights.Values {
		land[index] = height > 0
	}
	temperatures := temperature(heights, land, temperatureSettings(parameters.Map("temperature")))
	rains := rain(heights, land, rainSettings(parameters.Map("rain")))
	rules, regions, err := biomeTable(parameters)
	if err != nil {
		return nil, err
	}
	biomes, names, err := classify(heights, temperatures, rains, land, rules, regions, terrainSettings(parameters.Map("terrain")))
	if err != nil {
		return nil, err
	}
	return &Result{Temperature: temperatures, Precipitation: rains, Biome: biomes, Names: names}, nil
}

func gridOf(source *relief.Relief) *Grid {
	grid := newGrid(source.Columns, source.Rows, source.Cell, source.West, source.North)
	for index, height := range source.Heights {
		grid.Values[index] = float64(height)
	}
	return grid
}

func temperatureSettings(values plugin.Parameters) TemperatureSettings {
	return TemperatureSettings{SeaLevel: values.Number("sea_level", 14), Lapse: values.Number("lapse", 6.5),
		NorthwardGradient: values.Number("northward", 0)}
}

func rainSettings(values plugin.Parameters) RainSettings {
	return RainSettings{WindFrom: values.Number("wind_from", 270), Background: values.Number("background", 900),
		Orographic: values.Number("orographic", 2200), Recharge: values.Number("recharge", 0.08), Evaporation: values.Number("evaporation", 0.008),
		Base: values.Number("base", 0.004), Rise: values.Number("rise", 0.35), CalculationCell: values.Number("calculation_cell", 0.5),
		ReliefScale: values.Number("relief_scale", 2.5), Smoothing: values.Number("smoothing", 2)}
}

func terrainSettings(values plugin.Parameters) TerrainSettings {
	return TerrainSettings{RiverArea: values.Number("river_area", 40), RiverSlope: values.Number("river_slope", 0.03),
		CoastReach: values.Number("coast_reach", 0.6), CoastLow: values.Number("coast_low", 12), CoastSlope: values.Number("coast_slope", 0.04)}
}

func biomeTable(parameters plugin.Parameters) ([]BiomeRule, []Region, error) {
	var rules []BiomeRule
	entries, _ := parameters["biomes"].([]any)
	for _, entry := range entries {
		values, _ := entry.(map[string]any)
		name, _ := values["biome"].(string)
		if name == "" {
			return nil, nil, fmt.Errorf("a biome rule has no biome")
		}
		when, _ := values["when"].(map[string]any)
		rules = append(rules, BiomeRule{Biome: name, When: when})
	}
	var regions []Region
	entries, _ = parameters["regions"].([]any)
	for _, entry := range entries {
		values := plugin.Parameters(asMap(entry))
		region := Region{Biome: values.Text("biome", ""), Ragged: values.Number("ragged", 0)}
		region.When, _ = values["when"].(map[string]any)
		if ellipse := values.Map("ellipse"); len(ellipse) > 0 {
			region.Ellipse = &Ellipse{Centre: pair(ellipse["centre"], [2]float64{}), Radii: pair(ellipse["radii"], [2]float64{1, 1})}
		}
		points, _ := values["polygon"].([]any)
		for _, point := range points {
			region.Polygon = append(region.Polygon, pair(point, [2]float64{}))
		}
		if region.Biome == "" || (region.Ellipse == nil && len(region.Polygon) < 3) {
			return nil, nil, fmt.Errorf("a region needs a biome and an ellipse or a polygon")
		}
		regions = append(regions, region)
	}
	return rules, regions, nil
}

func asMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func pair(value any, fallback [2]float64) [2]float64 {
	values, _ := value.([]any)
	if len(values) != 2 {
		return fallback
	}
	first, firstIsNumber := numberOf(values[0])
	second, secondIsNumber := numberOf(values[1])
	if !firstIsNumber || !secondIsNumber {
		return fallback
	}
	return [2]float64{first, second}
}

// fieldOf is a grid as a world field: x = east − origin east, z = origin north − north; rows go
// from the smallest z (the north), like the grid's.
func fieldOf(name string, grid *Grid, origin [2]float64) *world.FieldPatch {
	values := make([]float32, len(grid.Values))
	for index, value := range grid.Values {
		values[index] = float32(value)
	}
	east, south := grid.West+float64(grid.Columns)*grid.Cell, grid.North-float64(grid.Rows)*grid.Cell
	extent := &world.Extent{MinimumX: grid.West - origin[0], MaximumX: east - origin[0],
		MinimumZ: origin[1] - grid.North, MaximumZ: origin[1] - south}
	return &world.FieldPatch{Name: name, Columns: grid.Columns, Rows: grid.Rows, Extent: extent, Values: values}
}
