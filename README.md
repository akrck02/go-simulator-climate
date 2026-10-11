# go-simulator-climate

> **Note:** this project was written with an AI assistant and is pending thorough review. See the [AI disclosure](#ai-disclosure).

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

## AI disclosure

I write most of my code by hand, so I want to be clear about this project: **it was written by an
AI assistant (Claude, by Anthropic) under my direction.** I decided what to build and how it should
behave; the code, the tests and the documentation were written by the AI. Every commit made this
way carries a `Co-Authored-By: Claude` line (all of them).

**Status: pending thorough review.** The code compiles and its tests pass, but I have not yet
reviewed it line by line. Treat it as a working draft, not as code I vouch for.
