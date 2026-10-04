module github.com/akrck02/go-simulator-climate

go 1.25.6

require (
	github.com/akrck02/go-scene-builder v0.0.0
	github.com/akrck02/go-simulator v0.0.0
)

// developed next to these projects
replace (
	github.com/akrck02/go-renderer => ../go-renderer
	github.com/akrck02/go-scene-builder => ../go-scene-builder
	github.com/akrck02/go-simulator => ../go-simulator
)
