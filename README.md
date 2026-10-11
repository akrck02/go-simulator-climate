# go-simulator-climate

![Temperature, rain and biomes of a made-up island](docs/img/climate.jpg)


A [go-simulator](https://github.com/akrck02/go-simulator) plugin that works out the climate of a relief and the biome of
every place: temperature falls with height, moist wind drops its rain on the slopes it climbs and
leaves a rain shadow behind, and biome rules over temperature, rain, slope, rivers and coast give
every place its biome. A `seasons` rule turns the year: snow line and foliage colour.

```yaml
- rule: climate
  with:
    relief: relieve.json
    origin: [6, 4.5]
    temperature: {sea_level: 12, lapse: 6.5}
    rain: {wind_from: 270, background: 800, orographic: 1800}
    biomes:
      - {biome: snow, when: {temperature: {max: -2.5}}}
      - {biome: forest, when: {precipitation: {min: 1000}}}
      - {biome: steppe}
```

Documentation in [docs/](docs/README.md): [the model](docs/model.md) and the
[rules reference](docs/rules.md).
