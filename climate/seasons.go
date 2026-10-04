package climate

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// The seasons rule turns the year: the temperature swings around its yearly mean, so the snow line
// (where the season's temperature falls below "snow_at") climbs in summer and comes down in winter,
// and foliage takes the colour of the season. Any clock works: the year lasts "year_length" units
// of the stage's time (seconds, with a real-time clock).
//
//	- rule: seasons
//	  with:
//	    year_length: 240         # seconds a year lasts on a real-time clock
//	    warmest: 0.3             # when in the year it is warmest (0 = the start)
//	    temperature: {sea_level: 12, lapse: 6.5, amplitude: 9}
//	    snow_at: 0               # °C
//	    foliage:
//	      - {at: 0.0, color: "#9cc35a", amount: 0.15}     # spring
//	      - {at: 0.3, color: "#5d7a3c", amount: 0}        # summer
//	      - {at: 0.6, color: "#c8752f", amount: 0.6}      # autumn
//	      - {at: 0.8, color: "#7a6e5f", amount: 0.6}      # winter

func seasonsRule() plugin.Definition {
	return plugin.Definition{
		Description: protocol.RuleDescription{
			Name:    "seasons",
			Summary: "Turns the year: moves the snow line with the season's temperature and gives foliage the season's colour (environment).",
			Parameters: []protocol.ParameterDescription{
				{Name: "year_length", Type: "number", Default: 240.0, Summary: "units of the stage's time a year lasts"},
				{Name: "warmest", Type: "number", Default: 0.3, Summary: "when it is warmest, as a share of the year"},
				{Name: "temperature", Type: "map", Summary: "{sea_level: °C, lapse: °C per km, amplitude: °C above and below the mean}"},
				{Name: "snow_at", Type: "number", Default: 0.0, Summary: "°C under which the ground is snowy"},
				{Name: "snow_blend", Type: "number", Default: 150.0, Summary: "metres over which the snow fades in"},
				{Name: "foliage", Type: "list", Summary: "[{at: share of the year, color, amount}], interpolated around the year"},
			},
			Reads: world.Interest{Environment: true}, Writes: []string{"environment"},
		},
		New: newSeasons,
	}
}

type seasons struct {
	yearLength, warmest, seaLevel, lapse, amplitude, snowAt, snowBlend float64
	foliage                                                            []foliageStop
}

type foliageStop struct {
	at     float64
	color  [4]float32
	amount float64
}

func newSeasons(parameters plugin.Parameters) (plugin.Rule, error) {
	temperature := parameters.Map("temperature")
	rule := &seasons{yearLength: math.Max(parameters.Number("year_length", 240), 1e-9), warmest: parameters.Number("warmest", 0.3),
		seaLevel: temperature.Number("sea_level", 12), lapse: math.Max(temperature.Number("lapse", 6.5), 1e-9),
		amplitude: temperature.Number("amplitude", 9), snowAt: parameters.Number("snow_at", 0), snowBlend: parameters.Number("snow_blend", 150)}
	entries, _ := parameters["foliage"].([]any)
	for _, entry := range entries {
		values := plugin.Parameters(asMap(entry))
		rule.foliage = append(rule.foliage, foliageStop{at: values.Number("at", 0), color: linearColor(values.Text("color", "#ffffff")),
			amount: values.Number("amount", 0)})
	}
	sort.Slice(rule.foliage, func(first, second int) bool { return rule.foliage[first].at < rule.foliage[second].at })
	return rule, nil
}

// Start shows the season of the start.
func (rule *seasons) Start(context *plugin.Context) error { return rule.Step(context) }

// Step shows the season of the current time.
func (rule *seasons) Step(context *plugin.Context) error {
	share := math.Mod(context.Time/rule.yearLength, 1)
	if share < 0 {
		share++
	}
	meanAtSea := rule.seaLevel + rule.amplitude*math.Cos(2*math.Pi*(share-rule.warmest))
	snowLine := (meanAtSea - rule.snowAt) / rule.lapse // km
	snow := &world.Snow{Level: float32(math.Max(snowLine, 0)), Blend: float32(rule.snowBlend / 1000)}
	tint := rule.foliageAt(share)
	context.SetEnvironment(&world.Environment{Snow: snow, FoliageTint: &tint})
	return nil
}

// foliageAt interpolates the foliage stops around the year.
func (rule *seasons) foliageAt(share float64) [4]float32 {
	if len(rule.foliage) == 0 {
		return [4]float32{}
	}
	next := sort.Search(len(rule.foliage), func(index int) bool { return rule.foliage[index].at > share })
	before, after := rule.foliage[(next-1+len(rule.foliage))%len(rule.foliage)], rule.foliage[next%len(rule.foliage)]
	span := math.Mod(after.at-before.at+1, 1)
	weight := 0.0
	if span > 0 {
		weight = math.Mod(share-before.at+1, 1) / span
	}
	var tint [4]float32
	for channel := range 3 {
		tint[channel] = before.color[channel] + float32(weight)*(after.color[channel]-before.color[channel])
	}
	tint[3] = float32(before.amount + weight*(after.amount-before.amount))
	return tint
}

// linearColor turns "#rrggbb" (sRGB) into linear RGB.
func linearColor(text string) [4]float32 {
	hex := strings.TrimPrefix(text, "#")
	color := [4]float32{1, 1, 1, 1}
	for channel := 0; channel < 3 && len(hex) >= 2*channel+2; channel++ {
		value, err := strconv.ParseUint(hex[2*channel:2*channel+2], 16, 8)
		if err != nil {
			continue
		}
		srgb := float64(value) / 255
		if srgb <= 0.04045 {
			color[channel] = float32(srgb / 12.92)
		} else {
			color[channel] = float32(math.Pow((srgb+0.055)/1.055, 2.4))
		}
	}
	return color
}
