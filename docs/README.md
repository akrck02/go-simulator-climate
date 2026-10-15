# 📚 go-simulator-climate documentation

| | Page | What is in it |
|---|---|---|
| 🚀 | [Getting started](getting-started.md) | The folders it needs, a whole rules file, running it without a window and in the viewer, the tests. |
| 🌦️ | [The climate model](model.md) | How temperature, rain, river valleys, coasts, biomes, regions and seasons are worked out, with maps and charts of a made-up island. |
| 🔍 | [How it works](how-it-works.md) | How the plugin fits go-simulator, what happens at the start and every step, and how the pictures were made. |
| 📋 | [Rules reference](rules.md) | Reference of the `climate` and `seasons` rules, generated from the plugin's own description. |

New here? Start with the [project README](../README.md).

## 🚧 Not done yet

- The climate does not change with time: the `climate` rule works it out once, and the `seasons`
  rule moves only the snow line and the foliage tint, not the `temperature` and `precipitation`
  fields. There is no weather yet (no single storms, no daily rain).
- A snapshot of the `biome` field is not yet a layer go-scene-builder reads as it is: its header
  has no `labels` (the names are in the `biomes` event), and its `west` and `north` are the world's,
  so they match the relief's only with `origin: [0, 0]` (see
  [getting started](getting-started.md#in-the-viewer)).
- The foliage tint is drawn only on materials marked as foliage; the made-up island in these pages
  has no vegetation, so its pictures show the snow but not the tint.
- There is no example rules file in the repository yet; the pages show rules to copy.
