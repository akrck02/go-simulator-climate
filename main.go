// Command go-simulator-climate is a go-simulator plugin that works out the climate of a relief
// (temperature, rain) and, from it, the biome of every place. The host starts it; it talks over
// standard input and output.
package main

import (
	"log"

	"github.com/akrck02/go-simulator-climate/climate"
)

func main() {
	log.SetFlags(0)
	if err := climate.Plugin().Serve(); err != nil {
		log.Fatal(err)
	}
}
