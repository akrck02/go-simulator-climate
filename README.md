# go-simulator-climate

A [go-simulator](../go-simulator) plugin that works out the climate of a relief and the biome of
every place, so that temperature, rain and biomes always agree with each other and with the
relief:

- **temperature**: a sea-level mean that falls with height (lapse rate) and, optionally, northwards;
- **rain**: moist air comes from one direction, recharges over the sea and drops its water where it
  climbs the large-scale relief (orographic rain), leaving a rain shadow that it slowly recovers
  from;
- **biomes**: rules over temperature, rain, height, slope, river valleys (from the water that drains
  through every place) and the coast, the first that matches winning; then **regions**, areas of the
  map where a world has biomes of its own (an ellipse or a polygon, with conditions and a ragged edge).

It publishes three fields — `temperature` (°C), `precipitation` (mm a year) and `biome` (an index,
−1 for the sea) — and a `biomes` event with the names of the biomes. Run without a window,
go-simulator saves the fields as grids that [go-scene-builder](../go-scene-builder) reads as a
layer (`-layer biome=…`): it colours the terrain and sows each biome's vegetation.

## Rules file

```yaml
plugins:
  - {name: climate, command: [go, run, .], directory: ~/Code/akrck02/go-simulator-climate}
stages:
  - name: climate
    clock: {from: 0, to: 1, step: 1}
    snapshots: [1]
    rules:
      - rule: climate
        with:
          relief: relieve.json            # the relief format of go-scene-builder (km east and north)
          origin: [6, 4.5]                # x = east − 6, z = 4.5 − north
          cell: 0.1                       # km
          temperature: {sea_level: 12, lapse: 6.5}
          rain: {wind_from: 270, background: 800, orographic: 1800}
          terrain: {river_area: 40, river_slope: 0.03, coast_reach: 0.6, coast_low: 12, coast_slope: 0.04}
          biomes:
            - {biome: snow, when: {temperature: {max: -2.5}}}
            - {biome: rock, when: {slope: {min: 0.6}}}
            - {biome: riverside, when: {river: true}}
            - {biome: forest, when: {precipitation: {min: 1000}}}
            - {biome: steppe}
          regions:
            - {biome: old wood, ellipse: {centre: [3, 6], radii: [2, 1.5]}, ragged: 0.15, when: {height: {min: 150}}}
```

```bash
cd ../go-simulator && go run ./cmd/simulate -rules climate.yaml -output results
```

The climate does not change with time yet: seasons (a moving snow line, the colours of the year)
are the next step.
