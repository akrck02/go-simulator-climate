# 📚 go-simulator-climate documentation

| | Page | What is in it |
|---|---|---|
| 🚀 | [Getting started](getting-started.md) | The folders it needs, a whole rules file, running it without a window and in the viewer, the tests. |
| 🌦️ | [The climate model](model.md) | How temperature, rain, river valleys, coasts, biomes, regions and seasons are worked out, with maps and charts of a made-up island. |
| 🔍 | [How it works](how-it-works.md) | How the plugin fits go-simulator, what happens at the start and every step, and how the pictures were made. |
| 📋 | [Rules reference](rules.md) | Reference of the `climate` and `seasons` rules, generated from the plugin's own description. |

New here? Start with the [project README](../README.md).

## 🚧 Not done yet

- The climate does not change with time on its own: the `climate` rule works it out once (and
  again only when the world's ground changes, as the terrain's `erosion` rule makes it), and the
  `seasons` rule moves only the snow line and the foliage tint, not the `temperature` and
  `precipitation` fields. There is no weather yet (no single storms, no daily rain).
- The foliage tint is drawn only on materials marked as foliage; the made-up island in these pages
  has no vegetation, so its pictures show the snow but not the tint.
- The only example rules file, `examples/mountain.yaml`, works out the climate of a synthetic
  mountain without the seasons; the pages show the rest as rules to copy.
