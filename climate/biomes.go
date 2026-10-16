package climate

import (
	"fmt"
	"math"
)

// Biomes are chosen per land cell by rules, the first that matches winning, then regions (areas of
// the map where the world has a biome of its own) override them. Conditions:
//
//	temperature, precipitation (mm), height (m), slope (rise over run), river_area (km² drained):
//	    {min: …, max: …}
//	river: true        on a valley floor of a large river (drained area over river_area_km2, flat)
//	coast: true        low and flat, close to the sea
//
//	biomes:
//	  - {biome: snow, when: {temperature: {max: -2.5}}}
//	  - {biome: rock, when: {slope: {min: 0.8}}}
//	  - {biome: steppe}                             # no conditions: everything left
//	regions:
//	  - {biome: old_forest, ellipse: {centre: [12, 8.5], radii: [4, 2.5]}, ragged: 0.15, when: {height: {min: 150, max: 1900}}}
//	  - {biome: orchards, polygon: [[…], …], when: {slope: {max: 0.06}}}

// BiomeRule is one rule of the biome table.
type BiomeRule struct {
	Biome string
	When  map[string]any
}

// Region gives an area of the map a biome of its own. Ragged (a share of the ellipse's size)
// makes its edge irregular, like a real forest's, with smooth noise.
type Region struct {
	Biome   string
	Ellipse *Ellipse
	Polygon [][2]float64
	When    map[string]any
	Ragged  float64
}

// Ellipse is an area: centre (km east, north) and radii (km east, north).
type Ellipse struct {
	Centre [2]float64
	Radii  [2]float64
}

// cellFacts are what the rules look at in one cell.
type cellFacts struct {
	temperature, precipitation, height, slope, riverArea float64
	river, coast                                         bool
}

// TerrainSettings say what counts as a river valley and a coast.
type TerrainSettings struct {
	RiverArea  float64 // km² drained
	RiverSlope float64 // flatter than this
	CoastReach float64 // km from the sea
	CoastLow   float64 // metres
	CoastSlope float64
}

// classify gives every land cell the index of its biome in names (−1 for the sea). drained is the
// area (km²) draining through every cell when the world has it; nil works it out here.
func classify(heights, temperatures, rains, drained *Grid, land []bool, rules []BiomeRule, regions []Region, terrain TerrainSettings) (*Grid, []string, error) {
	names, index := []string{}, map[string]int{}
	register := func(name string) int {
		if position, known := index[name]; known {
			return position
		}
		index[name] = len(names)
		names = append(names, name)
		return index[name]
	}
	for _, rule := range rules {
		register(rule.Biome)
	}
	for _, region := range regions {
		register(region.Biome)
	}
	ground := heights.like()
	for position, isLand := range land {
		if isLand {
			ground.Values[position] = heights.Values[position]
		}
	}
	slopes := ground.blurred(heights.Cell).slope()
	if drained == nil {
		drained = drainage(ground, land)
	}
	seaDistance := distanceToSea(heights, land)
	riverValleys := valleys(drained, slopes, land, terrain)
	result := heights.like()
	for row := range heights.Rows {
		for column := range heights.Columns {
			position := row*heights.Columns + column
			if !land[position] {
				result.Values[position] = -1
				continue
			}
			facts := cellFacts{temperature: temperatures.Values[position], precipitation: rains.Values[position], height: heights.Values[position],
				slope: slopes.Values[position], riverArea: drained.Values[position], river: riverValleys[position],
				coast: seaDistance.Values[position] < terrain.CoastReach && heights.Values[position] < terrain.CoastLow && slopes.Values[position] < terrain.CoastSlope}
			chosen := -1
			for _, rule := range rules {
				matches, err := holds(rule.When, facts)
				if err != nil {
					return nil, nil, fmt.Errorf("biome %q: %w", rule.Biome, err)
				}
				if matches {
					chosen = index[rule.Biome]
					break
				}
			}
			east, north := heights.centre(row, column)
			for _, region := range regions {
				matches, err := holds(region.When, facts)
				if err != nil {
					return nil, nil, fmt.Errorf("region %q: %w", region.Biome, err)
				}
				if matches && region.contains(east, north) {
					chosen = index[region.Biome]
				}
			}
			result.Values[position] = float64(chosen)
		}
	}
	return result, names, nil
}

// valleys are the flat floors along large rivers, widened by two cells.
func valleys(drained, slopes *Grid, land []bool, terrain TerrainSettings) []bool {
	core := make([]bool, len(land))
	for position, isLand := range land {
		core[position] = isLand && drained.Values[position] > terrain.RiverArea && slopes.Values[position] < terrain.RiverSlope
	}
	widened := make([]bool, len(land))
	for row := range drained.Rows {
		for column := range drained.Columns {
			position := row*drained.Columns + column
			if !core[position] {
				continue
			}
			for rowOffset := -2; rowOffset <= 2; rowOffset++ {
				for columnOffset := -2; columnOffset <= 2; columnOffset++ {
					nearRow, nearColumn := row+rowOffset, column+columnOffset
					if nearRow >= 0 && nearColumn >= 0 && nearRow < drained.Rows && nearColumn < drained.Columns {
						near := nearRow*drained.Columns + nearColumn
						widened[near] = widened[near] || (land[near] && slopes.Values[near] < terrain.RiverSlope*5/3)
					}
				}
			}
		}
	}
	return widened
}

// holds tells whether every condition of a rule holds in a cell.
func holds(when map[string]any, facts cellFacts) (bool, error) {
	for key, condition := range when {
		switch key {
		case "river", "coast":
			wanted, isBoolean := condition.(bool)
			if !isBoolean {
				return false, fmt.Errorf("%s must be true or false", key)
			}
			if (key == "river" && facts.river != wanted) || (key == "coast" && facts.coast != wanted) {
				return false, nil
			}
		case "temperature", "precipitation", "height", "slope", "river_area":
			value := map[string]float64{"temperature": facts.temperature, "precipitation": facts.precipitation, "height": facts.height,
				"slope": facts.slope, "river_area": facts.riverArea}[key]
			bounds, isMap := condition.(map[string]any)
			if !isMap {
				return false, fmt.Errorf("%s needs {min, max}", key)
			}
			if minimum, present := numberOf(bounds["min"]); present && value < minimum {
				return false, nil
			}
			if maximum, present := numberOf(bounds["max"]); present && value > maximum {
				return false, nil
			}
		default:
			return false, fmt.Errorf("unknown condition %q", key)
		}
	}
	return true, nil
}

func (region Region) contains(east, north float64) bool {
	if region.Ellipse != nil {
		across := (east - region.Ellipse.Centre[0]) / region.Ellipse.Radii[0]
		along := (north - region.Ellipse.Centre[1]) / region.Ellipse.Radii[1]
		reach := 1.0
		if region.Ragged > 0 {
			scale := math.Min(region.Ellipse.Radii[0], region.Ellipse.Radii[1]) / 4
			reach += region.Ragged * (2*smoothNoise(east/scale, north/scale) - 1)
		}
		return math.Sqrt(across*across+along*along) <= reach
	}
	return len(region.Polygon) >= 3 && polygonContains(region.Polygon, east, north)
}

// smoothNoise is value noise in 0–1 with two octaves, the same for the same place.
func smoothNoise(east, north float64) float64 {
	return 0.65*valueNoise(east, north) + 0.35*valueNoise(east*2.3+17, north*2.3+5)
}

func valueNoise(east, north float64) float64 {
	cellEast, cellNorth := math.Floor(east), math.Floor(north)
	insideEast, insideNorth := east-cellEast, north-cellNorth
	easeEast, easeNorth := insideEast*insideEast*(3-2*insideEast), insideNorth*insideNorth*(3-2*insideNorth)
	corner := func(column, row float64) float64 {
		value := math.Sin(column*127.1+row*311.7) * 43758.5453
		return value - math.Floor(value)
	}
	top := corner(cellEast, cellNorth)*(1-easeEast) + corner(cellEast+1, cellNorth)*easeEast
	bottom := corner(cellEast, cellNorth+1)*(1-easeEast) + corner(cellEast+1, cellNorth+1)*easeEast
	return top*(1-easeNorth) + bottom*easeNorth
}

func polygonContains(polygon [][2]float64, east, north float64) bool {
	inside := false
	for index, previous := 0, len(polygon)-1; index < len(polygon); previous, index = index, index+1 {
		first, second := polygon[index], polygon[previous]
		if (first[1] > north) != (second[1] > north) && east < (second[0]-first[0])*(north-first[1])/(second[1]-first[1])+first[0] {
			inside = !inside
		}
	}
	return inside
}

func numberOf(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	}
	return math.NaN(), false
}
