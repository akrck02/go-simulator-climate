package climate

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// writeLayer saves the biomes as a grid of categories that go-scene-builder reads as a layer
// (-layer biome=<path>): a header with the biome names as labels and the values, little-endian
// float32 row by row from the north, in a .f32 file next to it.
//
// The world's fields cannot be used for this as they are: their x and z move with the origin, and
// the names of their values travel apart, in the "biomes" event. The layer keeps the relief's own
// kilometres east and north, the ones the scene builder's map and relief use, so it lines up with
// them whatever the origin.
func writeLayer(path string, biomes *Grid, names []string) error {
	if !strings.HasSuffix(path, ".json") {
		return fmt.Errorf("layer %q: the header must be a .json file (the values go to a .f32 next to it)", path)
	}
	header, err := json.Marshal(map[string]any{"columns": biomes.Columns, "rows": biomes.Rows, "cell": biomes.Cell,
		"west": biomes.West, "north": biomes.North, "units": "km", "labels": names})
	if err != nil {
		return err
	}
	values := make([]byte, 4*len(biomes.Values))
	for index, value := range biomes.Values {
		binary.LittleEndian.PutUint32(values[4*index:], math.Float32bits(float32(value)))
	}
	if err := os.WriteFile(path, header, 0o644); err != nil {
		return err
	}
	return os.WriteFile(strings.TrimSuffix(path, ".json")+".f32", values, 0o644)
}
