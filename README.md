# aipencil

A CLI tool that renders structured JSON scene descriptions to SVG and PNG graphics. Designed to give AI systems a deterministic, consistent, and inexpensive way to produce images — from simple diagrams to character illustrations and game assets.

## Why?

AI image generation today relies on diffusion models that are expensive, inconsistent, and hard to control precisely. `aipencil` takes a different approach: describe what you want in structured JSON, get exactly that as output. Same input always produces the same image. No randomness, no hallucinated details, no GPU required.

## Showcase

### Simple shapes with text

```json
{"elements": [{"type": "group", "layout": {"type": "stack"}, "children": [
  {"type": "circle", "r": 50, "style": {"fill": "#4a90d9", "stroke": "#2c5f8a", "strokeWidth": 3}},
  {"type": "text", "text": "Hello\nWorld", "align": "center",
   "style": {"fill": "#ffffff", "fontSize": 18, "fontWeight": "bold"}}
]}]}
```

![Simple](examples/simple.png)

### Architecture diagrams

Full system architecture with styled boxes, labeled arrows, and column layout.

![Architecture](examples/architecture.png)

<details>
<summary>View JSON source</summary>

See [examples/architecture.json](examples/architecture.json)
</details>

### Character patterns

Built-in `person` and `face` patterns with customizable colors, expressions, and labels.

![Characters](examples/characters.png)

<details>
<summary>View JSON source</summary>

See [examples/characters.json](examples/characters.json)
</details>

### Force-directed graph layout

Automatic node positioning with `"layout": {"type": "graph"}`. Supports labeled arrows and multiple curve styles.

![Graph](examples/graph.png)

<details>
<summary>View JSON source</summary>

See [examples/graph.json](examples/graph.json)
</details>

## Install

```bash
go install github.com/KarpelesLab/aipencil@latest
```

For PNG output, one of these must be available on your system:
- `rsvg-convert` (librsvg) — recommended
- `inkscape`
- `magick` (ImageMagick)

## Usage

Flags come before the input file:

```bash
# JSON from stdin to SVG on stdout
echo '{"elements":[...]}' | aipencil

# File input, SVG output
aipencil -o diagram.svg input.json

# PNG output (format inferred from extension)
aipencil -o output.png input.json

# PNG at 2x resolution
aipencil -o output.png -scale 2 input.json

# Validate without rendering
aipencil -validate input.json

# List built-in patterns
aipencil -list-patterns

# Run as MCP server (stdio transport)
aipencil -mcp
```

## Scene Description Format

A scene is a JSON object with optional canvas settings and an array of elements:

```json
{
  "width": 800,
  "height": 600,
  "background": "#ffffff",
  "padding": 20,
  "pixelPerfect": false,
  "styles": {},
  "defs": {},
  "elements": []
}
```

All top-level fields are optional. When `width`/`height` are omitted, the canvas auto-sizes to fit the content.

### Element types

| Type | Description | Key fields |
|------|-------------|------------|
| `rect` | Rectangle | `width`, `height`, `rx` (corner radius) |
| `circle` | Circle | `r` (radius) |
| `ellipse` | Ellipse | `rx`, `ry` |
| `line` | Line segment | `x`, `y`, `x2`, `y2` |
| `path` | SVG path | `d` (path data) |
| `polygon` | Closed polygon | `points` (array of `[x, y]`) |
| `polyline` | Open polyline | `points` |
| `text` | Text label | `text`, `fontSize`, `fontWeight`, `align`, `maxWidth` |
| `image` | Embedded image | `href`, `width`, `height` |
| `group` | Container | `children`, `layout` |
| `arrow` | Connector | `from`, `to`, `label`, `curve`, `headStyle` |
| `use` | Pattern instance | `pattern`, `params` |

### Common element fields

Every element supports: `id`, `x`, `y`, `style`, `class`, `transform`, `children`.

When `x`/`y` are omitted, the layout engine positions the element automatically.

### Styles

Named styles are defined at the scene level and referenced with `class`. Inline `style` overrides class properties:

```json
{
  "styles": {
    "box": {"fill": "#e8e8ff", "stroke": "#555", "strokeWidth": 2, "rx": 10}
  },
  "elements": [
    {"type": "rect", "width": 100, "height": 50, "class": "box"},
    {"type": "rect", "width": 100, "height": 50, "class": "box", "style": {"fill": "#ffe8e8"}}
  ]
}
```

Style properties: `fill`, `stroke`, `strokeWidth`, `opacity`, `fontSize`, `fontWeight`, `fontFamily`, `textAnchor`, `rx`, `ry`, `strokeDasharray`, `strokeLinecap`, `strokeLinejoin`, `filter`, `imageRendering`.

### Layouts

Groups can specify how their children are arranged:

| Layout | Description | Options |
|--------|-------------|---------|
| `free` | Explicit positions (default) | — |
| `row` | Horizontal left-to-right | `gap`, `align` (top/center/bottom) |
| `column` | Vertical top-to-bottom | `gap`, `align` (left/center/right) |
| `grid` | Grid with wrapping | `columns`, `gap`, `cellWidth`, `cellHeight` |
| `stack` | All children centered (layered) | — |
| `graph` | Force-directed automatic layout | — |

```json
{"type": "group", "layout": {"type": "row", "gap": 20, "align": "center"}, "children": [...]}
```

### Arrows

Arrows connect elements by `id`. They support anchor points and multiple routing styles:

```json
{"type": "arrow", "from": "nodeA", "to": "nodeB", "label": "HTTP", "curve": "smooth"}
```

- **Anchors**: `"nodeA.right"`, `"nodeA.top"`, etc.
- **Curves**: `straight` (default), `smooth` (bezier), `orthogonal` (right-angle)
- **Head styles**: `filled` (default), `open`, `none`, `diamond`, `circle`

### Patterns / Templates

Reusable components defined in `defs` or loaded from built-in patterns:

```json
{
  "defs": {
    "badge": {
      "params": {
        "label": {"type": "string", "default": "OK"},
        "color": {"type": "color", "default": "#4CAF50"}
      },
      "width": 60, "height": 30,
      "elements": [
        {"type": "rect", "width": 60, "height": 30, "style": {"fill": "{{color}}", "rx": 15}},
        {"type": "text", "x": 30, "y": 15, "text": "{{label}}", "align": "center",
         "style": {"fill": "#fff", "fontSize": 12}}
      ]
    }
  },
  "elements": [
    {"type": "use", "pattern": "badge", "params": {"label": "NEW", "color": "#e74c3c"}}
  ]
}
```

Parameters use `{{paramName}}` substitution. Conditional elements use `"if": "expression == 'value'"`.

#### Built-in patterns

- **`face`** (60x80) — Head with customizable skin/hair/eye colors and expressions (neutral, happy, sad, surprised)
- **`person`** (50x120) — Full body with customizable clothing and skin colors
- **`box-with-label`** (120x50) — Labeled rectangle for diagrams

Run `aipencil -list-patterns` to see all parameters.

### Pixel-perfect mode

For game assets and pixel art, set `"pixelPerfect": true` at the scene level. This snaps all computed coordinates to integers. Combine with `-scale` for supersampled rasterization:

```bash
aipencil -o sprite.png -scale 4 sprite.json
```

## MCP Server

Run as an [MCP](https://modelcontextprotocol.io) tool server for direct AI integration:

```bash
aipencil -mcp
```

Exposes three tools:
- **`render`** — Render a scene to SVG or PNG
- **`validate`** — Check scene JSON for errors
- **`list_patterns`** — List available built-in patterns

## Regenerating examples

All example images can be regenerated from their JSON sources:

```bash
aipencil -o examples/simple.png -scale 2 examples/simple.json
aipencil -o examples/architecture.png -scale 2 examples/architecture.json
aipencil -o examples/characters.png -scale 2 examples/characters.json
aipencil -o examples/graph.png -scale 2 examples/graph.json
```

## License

MIT License. See [LICENSE](LICENSE).
