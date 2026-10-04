package climate

import (
	"math"
	"sort"
)

// The climate model is the landscape simulator's, kept simple and tunable:
//   - temperature: a sea-level mean that falls with height (lapse rate) and, optionally, northwards;
//   - rain: moist air comes from one direction, recharges over the sea and drops its water where it
//     climbs the large-scale relief (orographic rain), leaving a rain shadow behind that it slowly
//     recovers from; the rain is a background share of the air's humidity plus the orographic part;
//   - biomes come from both, with the slope, the rivers and the coast (see biomes.go).

// TemperatureSettings are the parameters of the temperature.
type TemperatureSettings struct {
	SeaLevel          float64 // °C, mean at sea level
	Lapse             float64 // °C lost per km of height
	NorthwardGradient float64 // °C per 10 km northwards (negative: colder to the north)
}

// temperature is the mean yearly temperature of every cell (sea cells as at sea level).
func temperature(heights *Grid, land []bool, settings TemperatureSettings) *Grid {
	result := heights.like()
	middle := heights.North - float64(heights.Rows)*heights.Cell/2
	for row := range heights.Rows {
		_, north := heights.centre(row, 0)
		for column := range heights.Columns {
			height := 0.0
			if land[row*heights.Columns+column] {
				height = heights.at(row, column)
			}
			value := settings.SeaLevel - settings.Lapse*height/1000 + settings.NorthwardGradient*(north-middle)/10
			result.set(row, column, value)
		}
	}
	return result
}

// RainSettings are the parameters of the rain.
type RainSettings struct {
	WindFrom        float64 // degrees the moist wind comes from, clockwise from north
	Background      float64 // mm a year with saturated air (frontal rain)
	Orographic      float64 // mm a year added on the most exposed slopes
	Recharge        float64 // share of the missing moisture the air takes back per cell over the sea
	Evaporation     float64 // the same over land
	Base            float64 // share of its moisture the air drops per cell over land
	Rise            float64 // extra share per unit of climb (metres up per metre forward)
	CalculationCell float64 // km: the moisture is carried on a coarser grid
	ReliefScale     float64 // km: the relief the air feels is smoothed this much
	Smoothing       float64 // km: the rain is smoothed this much
}

// rain is the yearly rain (mm) of every land cell, with a semi-Lagrangian moisture model.
func rain(heights *Grid, land []bool, settings RainSettings) *Grid {
	landGrid := heights.like()
	for index, isLand := range land {
		if isLand {
			landGrid.Values[index] = 1
		}
	}
	ground := heights.like()
	for index, isLand := range land {
		if isLand {
			ground.Values[index] = heights.Values[index]
		}
	}
	coarse := ground.blurred(settings.ReliefScale).resampled(settings.CalculationCell)
	coarseLand := landGrid.resampled(settings.CalculationCell)
	moisture, dropped := carryMoisture(coarse, coarseLand, settings)
	orographic := dropped.resampledTo(heights).blurred(settings.Smoothing)
	humidity := moisture.resampledTo(heights).blurred(settings.Smoothing)
	reference := percentileOnLand(orographic, land, 0.97)
	if reference <= 0 {
		reference = 1
	}
	result := heights.like()
	for index, isLand := range land {
		if isLand {
			result.Values[index] = settings.Background*humidity.Values[index] + settings.Orographic*orographic.Values[index]/reference
		}
	}
	return result
}

// carryMoisture sweeps the grid downwind: every cell takes the air of the cell upwind, which loses
// what it drops over the climbing land and recovers over the sea (and, slowly, over flat land).
func carryMoisture(heights, landShare *Grid, settings RainSettings) (moisture, dropped *Grid) {
	angle := settings.WindFrom * math.Pi / 180
	towardsEast, towardsNorth := -math.Sin(angle), -math.Cos(angle) // where the air goes
	columnStep, rowStep := towardsEast, -towardsNorth               // in cells
	type cell struct {
		row, column int
		order       float64
	}
	cells := make([]cell, 0, heights.Rows*heights.Columns)
	for row := range heights.Rows {
		for column := range heights.Columns {
			cells = append(cells, cell{row, column, float64(column)*columnStep + float64(row)*rowStep})
		}
	}
	sort.SliceStable(cells, func(first, second int) bool { return cells[first].order < cells[second].order })
	moisture, dropped = heights.like(), heights.like()
	for index := range moisture.Values {
		moisture.Values[index] = 1
	}
	spacing := heights.Cell * 1000
	for start := 0; start < len(cells); {
		end := start
		for end < len(cells) && cells[end].order < cells[start].order+0.7 {
			end++
		}
		incoming := make([]float64, end-start)
		upwindHeight := make([]float64, end-start)
		for index, current := range cells[start:end] {
			upwindRow, upwindColumn := float64(current.row)-rowStep, float64(current.column)-columnStep
			if upwindRow < 0 || upwindColumn < 0 || upwindRow > float64(heights.Rows-1) || upwindColumn > float64(heights.Columns-1) {
				incoming[index] = 1
			} else {
				incoming[index] = moisture.sampleCell(upwindRow, upwindColumn)
			}
			upwindHeight[index] = heights.sampleCell(upwindRow, upwindColumn)
		}
		for index, current := range cells[start:end] {
			onLand := landShare.at(current.row, current.column) > 0.5
			climb := math.Max(0, (heights.at(current.row, current.column)-upwindHeight[index])/spacing)
			drop := 0.0
			if onLand {
				drop = clamp(incoming[index]*(settings.Base+settings.Rise*climb), 0, 0.5*incoming[index])
			}
			left := incoming[index] - drop
			if onLand {
				left += settings.Evaporation * (0.5 - left)
			} else {
				left += settings.Recharge * (1 - left)
			}
			moisture.set(current.row, current.column, left)
			dropped.set(current.row, current.column, drop)
		}
		start = end
	}
	return moisture, dropped
}

// resampledTo is the grid sampled at the cells of another grid.
func (grid *Grid) resampledTo(other *Grid) *Grid {
	result := other.like()
	for row := range other.Rows {
		for column := range other.Columns {
			east, north := other.centre(row, column)
			result.set(row, column, grid.sample(east, north))
		}
	}
	return result
}

func percentileOnLand(grid *Grid, land []bool, share float64) float64 {
	var values []float64
	for index, isLand := range land {
		if isLand {
			values = append(values, grid.Values[index])
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	return values[min(len(values)-1, int(share*float64(len(values))))]
}

// drainage is the area (km²) that drains through every cell: water flows to the lowest of the eight
// neighbours.
func drainage(heights *Grid, land []bool) *Grid {
	order := make([]int, len(heights.Values))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(first, second int) bool { return heights.Values[order[first]] > heights.Values[order[second]] })
	area := heights.like()
	cellArea := heights.Cell * heights.Cell
	for _, index := range order {
		if !land[index] {
			continue
		}
		area.Values[index] += cellArea
		row, column := index/heights.Columns, index%heights.Columns
		lowest, lowestHeight := -1, heights.Values[index]
		for rowOffset := -1; rowOffset <= 1; rowOffset++ {
			for columnOffset := -1; columnOffset <= 1; columnOffset++ {
				neighbourRow, neighbourColumn := row+rowOffset, column+columnOffset
				if (rowOffset == 0 && columnOffset == 0) || neighbourRow < 0 || neighbourColumn < 0 || neighbourRow >= heights.Rows || neighbourColumn >= heights.Columns {
					continue
				}
				neighbour := neighbourRow*heights.Columns + neighbourColumn
				if heights.Values[neighbour] < lowestHeight {
					lowest, lowestHeight = neighbour, heights.Values[neighbour]
				}
			}
		}
		if lowest >= 0 {
			area.Values[lowest] += area.Values[index]
		}
	}
	return area
}

// distanceToSea is how far (km) every land cell is from the nearest sea cell (8-neighbour steps).
func distanceToSea(grid *Grid, land []bool) *Grid {
	result := grid.like()
	queue := make([]int, 0, len(land))
	for index, isLand := range land {
		if isLand {
			result.Values[index] = math.Inf(1)
		} else {
			queue = append(queue, index)
		}
	}
	diagonal := math.Sqrt2 * grid.Cell
	for len(queue) > 0 {
		next := queue[:0:0]
		for _, index := range queue {
			row, column := index/grid.Columns, index%grid.Columns
			for rowOffset := -1; rowOffset <= 1; rowOffset++ {
				for columnOffset := -1; columnOffset <= 1; columnOffset++ {
					neighbourRow, neighbourColumn := row+rowOffset, column+columnOffset
					if (rowOffset == 0 && columnOffset == 0) || neighbourRow < 0 || neighbourColumn < 0 || neighbourRow >= grid.Rows || neighbourColumn >= grid.Columns {
						continue
					}
					step := grid.Cell
					if rowOffset != 0 && columnOffset != 0 {
						step = diagonal
					}
					neighbour := neighbourRow*grid.Columns + neighbourColumn
					if result.Values[index]+step < result.Values[neighbour] {
						result.Values[neighbour] = result.Values[index] + step
						next = append(next, neighbour)
					}
				}
			}
		}
		queue = next
	}
	return result
}
