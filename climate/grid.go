package climate

import "math"

// Grid is a regular grid of values over kilometres east and north: West and North are its edges,
// each value sits at the centre of its cell, rows go from the north.
type Grid struct {
	Columns, Rows int
	Cell          float64 // km
	West, North   float64
	Values        []float64
}

func newGrid(columns, rows int, cell, west, north float64) *Grid {
	return &Grid{Columns: columns, Rows: rows, Cell: cell, West: west, North: north, Values: make([]float64, columns*rows)}
}

// like is an empty grid with the same layout.
func (grid *Grid) like() *Grid {
	return newGrid(grid.Columns, grid.Rows, grid.Cell, grid.West, grid.North)
}

func (grid *Grid) at(row, column int) float64 { return grid.Values[row*grid.Columns+column] }

func (grid *Grid) set(row, column int, value float64) { grid.Values[row*grid.Columns+column] = value }

// centre is the position of a cell centre.
func (grid *Grid) centre(row, column int) (east, north float64) {
	return grid.West + (float64(column)+0.5)*grid.Cell, grid.North - (float64(row)+0.5)*grid.Cell
}

// sample is the value at a position, bilinear between cell centres (clamped at the edges).
func (grid *Grid) sample(east, north float64) float64 {
	column := clamp((east-grid.West)/grid.Cell-0.5, 0, float64(grid.Columns-1))
	row := clamp((grid.North-north)/grid.Cell-0.5, 0, float64(grid.Rows-1))
	return grid.sampleCell(row, column)
}

// nearest is the value of the cell a position falls in (for categories, which do not blend).
func (grid *Grid) nearest(east, north float64) float64 {
	column := clampInteger(int((east-grid.West)/grid.Cell), 0, grid.Columns-1)
	row := clampInteger(int((grid.North-north)/grid.Cell), 0, grid.Rows-1)
	return grid.at(row, column)
}

// sampleCell is the value at fractional cell coordinates, bilinear (clamped at the edges).
func (grid *Grid) sampleCell(row, column float64) float64 {
	row, column = clamp(row, 0, float64(grid.Rows-1)), clamp(column, 0, float64(grid.Columns-1))
	top, left := int(row), int(column)
	bottom, right := min(top+1, grid.Rows-1), min(left+1, grid.Columns-1)
	down, across := row-float64(top), column-float64(left)
	upper := grid.at(top, left)*(1-across) + grid.at(top, right)*across
	lower := grid.at(bottom, left)*(1-across) + grid.at(bottom, right)*across
	return upper*(1-down) + lower*down
}

// resampled is the grid at another cell size over the same area.
func (grid *Grid) resampled(cell float64) *Grid {
	columns := max(1, int(math.Round(float64(grid.Columns)*grid.Cell/cell)))
	rows := max(1, int(math.Round(float64(grid.Rows)*grid.Cell/cell)))
	result := newGrid(columns, rows, cell, grid.West, grid.North)
	for row := range rows {
		for column := range columns {
			east, north := result.centre(row, column)
			result.set(row, column, grid.sample(east, north))
		}
	}
	return result
}

// blurred is the grid smoothed with a Gaussian of sigma kilometres (separable, clamped edges).
func (grid *Grid) blurred(sigma float64) *Grid {
	radius := sigma / grid.Cell
	if radius < 0.3 {
		return grid
	}
	reach := int(math.Ceil(3 * radius))
	weights := make([]float64, 2*reach+1)
	total := 0.0
	for offset := -reach; offset <= reach; offset++ {
		weights[offset+reach] = math.Exp(-float64(offset*offset) / (2 * radius * radius))
		total += weights[offset+reach]
	}
	for index := range weights {
		weights[index] /= total
	}
	across := grid.like()
	for row := range grid.Rows {
		for column := range grid.Columns {
			sum := 0.0
			for offset := -reach; offset <= reach; offset++ {
				sum += weights[offset+reach] * grid.at(row, clampInteger(column+offset, 0, grid.Columns-1))
			}
			across.set(row, column, sum)
		}
	}
	result := grid.like()
	for row := range grid.Rows {
		for column := range grid.Columns {
			sum := 0.0
			for offset := -reach; offset <= reach; offset++ {
				sum += weights[offset+reach] * across.at(clampInteger(row+offset, 0, grid.Rows-1), column)
			}
			result.set(row, column, sum)
		}
	}
	return result
}

// slope is the steepness of a grid of heights (metres) at every cell, rise over run.
func (grid *Grid) slope() *Grid {
	result := grid.like()
	spacing := grid.Cell * 1000
	for row := range grid.Rows {
		for column := range grid.Columns {
			east := grid.at(row, clampInteger(column+1, 0, grid.Columns-1)) - grid.at(row, clampInteger(column-1, 0, grid.Columns-1))
			north := grid.at(clampInteger(row-1, 0, grid.Rows-1), column) - grid.at(clampInteger(row+1, 0, grid.Rows-1), column)
			result.set(row, column, math.Hypot(east, north)/(2*spacing))
		}
	}
	return result
}

func clamp(value, low, high float64) float64 { return math.Max(low, math.Min(high, value)) }

func clampInteger(value, low, high int) int { return max(low, min(high, value)) }
