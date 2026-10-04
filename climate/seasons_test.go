package climate

import (
	"math"
	"testing"

	"github.com/akrck02/go-simulator/plugin"
)

func TestSeasonsMoveTheSnowAndTheLeaves(t *testing.T) {
	rule, err := newSeasons(plugin.Parameters{
		"year_length": 100.0, "warmest": 0.5,
		"temperature": map[string]any{"sea_level": 10.0, "lapse": 5.0, "amplitude": 10.0},
		"foliage": []any{
			map[string]any{"at": 0.0, "color": "#ffffff", "amount": 1.0},
			map[string]any{"at": 0.5, "color": "#000000", "amount": 0.0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	seasons := rule.(*seasons)
	// coldest at the start: 10 − 10 = 0 °C at sea level → snow from sea level; warmest at mid-year: 20 °C → 4 km
	for _, check := range []struct{ share, snowLine float64 }{{0, 0}, {0.5, 4}} {
		mean := seasons.seaLevel + seasons.amplitude*math.Cos(2*math.Pi*(check.share-seasons.warmest))
		if line := (mean - seasons.snowAt) / seasons.lapse; math.Abs(line-check.snowLine) > 1e-9 {
			t.Errorf("snow line at %v of the year: %v km, want %v", check.share, line, check.snowLine)
		}
	}
	if tint := seasons.foliageAt(0.25); math.Abs(float64(tint[3])-0.5) > 1e-6 {
		t.Errorf("a quarter into the year the foliage is halfway between the stops: %v", tint)
	}
	if tint := seasons.foliageAt(0.75); math.Abs(float64(tint[3])-0.5) > 1e-6 {
		t.Errorf("and it comes back around the year: %v", tint)
	}
}
