package climate

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-scene-builder/relief"
	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/world"
)

// island writes a relief: a round island 20 km across with a ridge running north to south.
func island(t *testing.T) string {
	folder := t.TempDir()
	columns, rows, cell := 120, 120, 0.25
	values := make([]byte, 4*columns*rows)
	for row := range rows {
		for column := range columns {
			east, north := (float64(column)+0.5)*cell-15, 15-(float64(row)+0.5)*cell
			height := -50.0
			if math.Hypot(east, north) < 10 {
				height = 20 + 1800*math.Exp(-east*east/4)*(1-math.Hypot(east, north)/10)
			}
			binary.LittleEndian.PutUint32(values[4*(row*columns+column):], math.Float32bits(float32(height)))
		}
	}
	header, _ := json.Marshal(map[string]any{"columns": columns, "rows": rows, "cell": cell, "west": -15, "north": 15, "units": "km"})
	os.WriteFile(filepath.Join(folder, "relieve.json"), header, 0o644)
	os.WriteFile(filepath.Join(folder, "relieve.f32"), values, 0o644)
	return filepath.Join(folder, "relieve.json")
}

func parameters(path string) plugin.Parameters {
	return plugin.Parameters{
		"relief":      path,
		"cell":        0.25,
		"temperature": map[string]any{"sea_level": 14.0, "lapse": 6.5},
		"rain":        map[string]any{"wind_from": 270.0, "calculation_cell": 0.5, "relief_scale": 1.0, "smoothing": 0.5},
		"biomes": []any{
			map[string]any{"biome": "snow", "when": map[string]any{"temperature": map[string]any{"max": 5.0}}},
			map[string]any{"biome": "forest", "when": map[string]any{"precipitation": map[string]any{"min": 1000.0}}},
			map[string]any{"biome": "steppe"},
		},
		"regions": []any{map[string]any{"biome": "orchard", "ellipse": map[string]any{"centre": []any{0.0, -8.0}, "radii": []any{1.0, 1.0}},
			"when": map[string]any{"height": map[string]any{"max": 300.0}}}},
	}
}

func TestClimateOfARidge(t *testing.T) {
	result, err := Compute(parameters(island(t)))
	if err != nil {
		t.Fatal(err)
	}
	at := func(grid *Grid, east, north float64) float64 { return grid.sample(east, north) }
	if peak := at(result.Temperature, 0, 0); math.Abs(peak-(14-6.5*1.82)) > 0.5 {
		t.Errorf("temperature on the ridge %.1f °C, want about %.1f", peak, 14-6.5*1.82)
	}
	windward, leeward := at(result.Precipitation, -3, 3), at(result.Precipitation, 3, 3)
	if windward <= leeward*1.2 {
		t.Errorf("the slope facing the west wind should be wetter: windward %.0f mm, leeward %.0f mm", windward, leeward)
	}
	names := map[float64]string{-1: "sea"}
	for index, name := range result.Names {
		names[float64(index)] = name
	}
	for _, check := range []struct {
		east, north float64
		want        string
	}{{0, 0, "snow"}, {14, 14, "sea"}, {0, -8.6, "orchard"}} {
		if got := names[result.Biome.nearest(check.east, check.north)]; got != check.want {
			t.Errorf("biome at %v, %v: %s, want %s", check.east, check.north, got, check.want)
		}
	}
}

func TestTheBiomeLayerLinesUpWithTheReliefWhateverTheOrigin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "biomes.json")
	settings := parameters(island(t))
	settings["origin"], settings["layer"] = []any{6.0, 4.0}, path
	rule, err := newClimate(settings)
	if err != nil {
		t.Fatal(err)
	}
	if err := rule.(*climateRuleInstance).Start(&plugin.Context{World: world.New()}); err != nil {
		t.Fatal(err)
	}
	layer, err := relief.Load(path)
	if err != nil {
		t.Fatalf("go-scene-builder should read the layer: %v", err)
	}
	// the same places as TestClimateOfARidge, in the relief's km east and north: the origin moves the
	// world's fields, not the layer
	for _, check := range []struct {
		east, north float64
		want        string
	}{{0, 0, "snow"}, {14, 14, ""}, {0, -8.6, "orchard"}} {
		category, inside := layer.Category(check.east, check.north)
		if !inside {
			t.Fatalf("%v, %v should be inside the layer", check.east, check.north)
		}
		if got := layer.Label(category); got != check.want {
			t.Errorf("biome of the layer at %v, %v: %q, want %q", check.east, check.north, got, check.want)
		}
	}
}

func TestTheLayerNeedsAJsonHeader(t *testing.T) {
	if err := writeLayer(filepath.Join(t.TempDir(), "biomes.f32"), newGrid(2, 2, 1, 0, 2), nil); err == nil {
		t.Fatal("a layer path that is not a .json header must be refused")
	}
}
