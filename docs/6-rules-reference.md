# 📋 Rules of the climate plugin

Version 1.0.0, protocol 2. Yearly temperature and rain over a relief, the biome of every place, and the turning seasons.

This reference is generated from what the plugin itself tells the host (the `describe` message), so it always
matches the code. Parameters go under `with:` in a rules file; see go-simulator's
[rules file reference](https://github.com/akrck02/go-simulator/blob/main/docs/4-guides/1-rules-files.md).

| Rule | What it does |
|---|---|
| [`climate`](#climate) | Works out temperature (fields: temperature, °C), rain (precipitation, mm a year) and biome (biome: index into the names of the "biomes" event, −1 for the sea) at the start, and again when the world's ground changes. |
| [`seasons`](#seasons) | Turns the year: moves the snow line with the season's temperature and gives foliage the season's colour (environment). |

## `climate`

Works out temperature (fields: temperature, °C), rain (precipitation, mm a year) and biome (biome: index into the names of the "biomes" event, −1 for the sea) at the start, and again when the world's ground changes.

- **Reads:** fields `elevation`, `drainage_area`
- **Writes:** `temperature`, `precipitation`, `biome`

| Parameter | Type | Default | Meaning |
|---|---|---|---|
| `relief` | text |  | relief header (relieve.json with relieve.f32 next to it), km east and north; without it, the world's elevation |
| `origin` | list |  | [east, north] of the world's origin: x = east − origin[0], z = origin[1] − north |
| `cell` | number | `0.1` | km: cell of the fields |
| `temperature` | map |  | {sea_level: °C, lapse: °C per km, northward: °C per 10 km} |
| `rain` | map |  | {wind_from: degrees, background: mm, orographic: mm, …} |
| `terrain` | map |  | {river_area: km², river_slope, coast_reach: km, coast_low: m, coast_slope} |
| `biomes` | list |  | rules: [{biome, when: {temperature, precipitation, height, slope: {min, max}, river, coast}}] |
| `regions` | list |  | [{biome, ellipse: {centre, radii} \| polygon, when}]: the world's own biomes |
| `layer` | text |  | biomes.json: also save the biomes as a layer for go-scene-builder (-layer biome=…), with their names, in km east and north |

## `seasons`

Turns the year: moves the snow line with the season's temperature and gives foliage the season's colour (environment).

- **Reads:** the environment
- **Writes:** `environment`

| Parameter | Type | Default | Meaning |
|---|---|---|---|
| `year_length` | number | `240` | units of the stage's time a year lasts |
| `warmest` | number | `0.3` | when it is warmest, as a share of the year |
| `temperature` | map |  | {sea_level: °C, lapse: °C per km, amplitude: °C above and below the mean} |
| `snow_at` | number | `0` | °C under which the ground is snowy |
| `snow_blend` | number | `150` | metres over which the snow fades in |
| `foliage` | list |  | [{at: share of the year, color, amount}], interpolated around the year |
