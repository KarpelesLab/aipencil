package layout

import (
	"math"
	"strings"

	"github.com/KarpelesLab/aipencil/font"
	"github.com/KarpelesLab/aipencil/scene"
)

// Layout runs the full layout pipeline on a scene:
// 1. Measure pass (bottom-up) — compute natural sizes
// 2. Place pass (top-down) — assign positions
// 3. Canvas auto-size if needed
func Layout(s *scene.Scene) {
	// Measure all elements
	for _, el := range s.Elements {
		measure(el, s)
	}

	// Place elements that lack explicit positions
	placeChildren(s.Elements, nil, s)

	// Auto-size canvas
	if s.Width == nil || s.Height == nil {
		w, h := canvasBounds(s.Elements)
		pad := 20.0
		if s.Padding != nil {
			pad = *s.Padding
		}
		if s.Width == nil {
			v := w + pad*2
			s.Width = &v
		}
		if s.Height == nil {
			v := h + pad*2
			s.Height = &v
		}
	}

	// Pixel-perfect snapping
	if s.PixelPerfect {
		for _, el := range s.Elements {
			snapToPixels(el)
		}
	}
}

// measure computes natural sizes bottom-up.
func measure(el *scene.Element, s *scene.Scene) {
	// Measure children first
	for _, child := range el.Children {
		measure(child, s)
	}

	switch el.Type {
	case "rect":
		el.ComputedWidth = valOr(el.Width, 100)
		el.ComputedHeight = valOr(el.Height, 60)

	case "circle":
		r := valOr(el.R, 25)
		el.ComputedWidth = r * 2
		el.ComputedHeight = r * 2

	case "ellipse":
		rx := valOr(el.RX, 30)
		ry := valOr(el.RY, 20)
		el.ComputedWidth = rx * 2
		el.ComputedHeight = ry * 2

	case "text":
		fontSize := valOr(el.FontSize, 14)
		fontWeight := el.FontWeight
		style := resolveStyleForLayout(el, s)
		if style != nil {
			if style.FontSize != nil {
				fontSize = *style.FontSize
			}
			if style.FontWeight != "" {
				fontWeight = style.FontWeight
			}
		}

		text := el.Text
		lines := strings.Split(text, "\n")

		// Auto-wrap if maxWidth is set
		if el.MaxWidth != nil && *el.MaxWidth > 0 {
			var wrapped []string
			for _, line := range lines {
				wrapped = append(wrapped, font.WrapText(line, *el.MaxWidth, fontSize, fontWeight)...)
			}
			lines = wrapped
		}

		w, h := font.MeasureLines(lines, fontSize, fontWeight, 1.4)
		el.ComputedWidth = w
		el.ComputedHeight = h

	case "line":
		x1, y1 := valOr(el.X, 0), valOr(el.Y, 0)
		x2, y2 := valOr(el.X2, 0), valOr(el.Y2, 0)
		el.ComputedWidth = math.Abs(x2 - x1)
		el.ComputedHeight = math.Abs(y2 - y1)

	case "image":
		el.ComputedWidth = valOr(el.Width, 100)
		el.ComputedHeight = valOr(el.Height, 100)

	case "group":
		layoutGroup(el, s)

	case "polygon", "polyline":
		if len(el.Points) > 0 {
			var maxX, maxY float64
			for _, p := range el.Points {
				if p[0] > maxX {
					maxX = p[0]
				}
				if p[1] > maxY {
					maxY = p[1]
				}
			}
			el.ComputedWidth = maxX
			el.ComputedHeight = maxY
		}

	case "use":
		// Pattern expansion should happen before layout; if not expanded yet,
		// use declared width/height from the pattern def
		if s.Defs != nil {
			if def, ok := s.Defs[el.Pattern]; ok {
				el.ComputedWidth = valOr(el.Width, def.Width)
				el.ComputedHeight = valOr(el.Height, def.Height)
			}
		}
		if el.ComputedWidth == 0 {
			el.ComputedWidth = valOr(el.Width, 60)
		}
		if el.ComputedHeight == 0 {
			el.ComputedHeight = valOr(el.Height, 60)
		}
	}
}

// layoutGroup computes size based on layout type and positions children.
func layoutGroup(el *scene.Element, s *scene.Scene) {
	layoutType := "free"
	gap := 20.0
	align := "center"
	if el.Layout != nil {
		layoutType = el.Layout.Type
		if el.Layout.Gap > 0 {
			gap = el.Layout.Gap
		}
		if el.Layout.Align != "" {
			align = el.Layout.Align
		}
	}

	switch layoutType {
	case "row":
		layoutRow(el, gap, align)
	case "column":
		layoutColumn(el, gap, align)
	case "grid":
		layoutGrid(el, s)
	case "stack":
		layoutStack(el)
	case "graph":
		layoutGraph(el, s.Elements)
	default: // "free"
		layoutFree(el)
	}
}

func layoutRow(el *scene.Element, gap float64, align string) {
	x := 0.0
	var maxH float64

	for _, child := range el.Children {
		if child.ComputedHeight > maxH {
			maxH = child.ComputedHeight
		}
	}

	for _, child := range el.Children {
		if child.X == nil {
			child.ComputedX = x
			child.Positioned = true
		}
		if child.Y == nil {
			switch align {
			case "top":
				child.ComputedY = 0
			case "bottom":
				child.ComputedY = maxH - child.ComputedHeight
			default: // center
				child.ComputedY = (maxH - child.ComputedHeight) / 2
			}
			child.Positioned = true
		}
		x += child.ComputedWidth + gap
	}

	el.ComputedWidth = x - gap
	if el.ComputedWidth < 0 {
		el.ComputedWidth = 0
	}
	el.ComputedHeight = maxH
}

func layoutColumn(el *scene.Element, gap float64, align string) {
	y := 0.0
	var maxW float64

	for _, child := range el.Children {
		if child.ComputedWidth > maxW {
			maxW = child.ComputedWidth
		}
	}

	for _, child := range el.Children {
		if child.Y == nil {
			child.ComputedY = y
			child.Positioned = true
		}
		if child.X == nil {
			switch align {
			case "left":
				child.ComputedX = 0
			case "right":
				child.ComputedX = maxW - child.ComputedWidth
			default: // center
				child.ComputedX = (maxW - child.ComputedWidth) / 2
			}
			child.Positioned = true
		}
		y += child.ComputedHeight + gap
	}

	el.ComputedWidth = maxW
	el.ComputedHeight = y - gap
	if el.ComputedHeight < 0 {
		el.ComputedHeight = 0
	}
}

func layoutGrid(el *scene.Element, s *scene.Scene) {
	cols := 3
	gap := 20.0
	if el.Layout != nil {
		if el.Layout.Columns > 0 {
			cols = el.Layout.Columns
		}
		if el.Layout.Gap > 0 {
			gap = el.Layout.Gap
		}
	}

	// Compute cell size
	cellW, cellH := 0.0, 0.0
	if el.Layout != nil && el.Layout.CellWidth > 0 {
		cellW = el.Layout.CellWidth
	}
	if el.Layout != nil && el.Layout.CellHeight > 0 {
		cellH = el.Layout.CellHeight
	}

	// Auto-compute cell size from largest child
	if cellW == 0 || cellH == 0 {
		for _, child := range el.Children {
			if child.ComputedWidth > cellW {
				cellW = child.ComputedWidth
			}
			if child.ComputedHeight > cellH {
				cellH = child.ComputedHeight
			}
		}
	}

	for i, child := range el.Children {
		col := i % cols
		row := i / cols
		if child.X == nil {
			child.ComputedX = float64(col) * (cellW + gap)
			child.Positioned = true
		}
		if child.Y == nil {
			child.ComputedY = float64(row) * (cellH + gap)
			child.Positioned = true
		}
	}

	rows := (len(el.Children) + cols - 1) / cols
	el.ComputedWidth = float64(cols)*(cellW+gap) - gap
	el.ComputedHeight = float64(rows)*(cellH+gap) - gap
}

func layoutStack(el *scene.Element) {
	var maxW, maxH float64
	for _, child := range el.Children {
		if child.ComputedWidth > maxW {
			maxW = child.ComputedWidth
		}
		if child.ComputedHeight > maxH {
			maxH = child.ComputedHeight
		}
	}

	// Center all children
	for _, child := range el.Children {
		if child.X == nil {
			child.ComputedX = (maxW - child.ComputedWidth) / 2
			child.Positioned = true
		}
		if child.Y == nil {
			child.ComputedY = (maxH - child.ComputedHeight) / 2
			child.Positioned = true
		}
	}

	el.ComputedWidth = maxW
	el.ComputedHeight = maxH
}

func layoutFree(el *scene.Element) {
	var maxX, maxY float64
	for _, child := range el.Children {
		cx := valOr(child.X, child.ComputedX)
		cy := valOr(child.Y, child.ComputedY)
		right := cx + child.ComputedWidth
		bottom := cy + child.ComputedHeight
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}
	el.ComputedWidth = maxX
	el.ComputedHeight = maxY
}

// placeChildren assigns default positions to unpositioned top-level elements.
func placeChildren(elements []*scene.Element, parent *scene.Element, s *scene.Scene) {
	pad := 20.0
	if s.Padding != nil {
		pad = *s.Padding
	}

	x, y := pad, pad
	rowH := 0.0
	maxW := 800.0 // default wrap width
	if s.Width != nil {
		maxW = *s.Width
	}

	for _, el := range elements {
		if el.Type == "arrow" {
			continue // arrows don't need positions
		}

		if el.X == nil && !el.Positioned {
			// Check if we need to wrap to next row
			if x+el.ComputedWidth > maxW-pad && x > pad {
				x = pad
				y += rowH + 20
				rowH = 0
			}
			el.ComputedX = x
			el.Positioned = true
			x += el.ComputedWidth + 20
		}
		if el.Y == nil && !el.Positioned {
			el.ComputedY = y
			el.Positioned = true
		}

		if el.ComputedHeight > rowH {
			rowH = el.ComputedHeight
		}
	}
}

// canvasBounds computes the total bounding box of all elements.
func canvasBounds(elements []*scene.Element) (width, height float64) {
	for _, el := range elements {
		x := valOr(el.X, el.ComputedX)
		y := valOr(el.Y, el.ComputedY)

		right := x + el.ComputedWidth
		bottom := y + el.ComputedHeight

		if right > width {
			width = right
		}
		if bottom > height {
			height = bottom
		}

		// Check children (for groups with offset)
		if len(el.Children) > 0 {
			cw, ch := canvasBounds(el.Children)
			if x+cw > width {
				width = x + cw
			}
			if y+ch > height {
				height = y + ch
			}
		}
	}
	return
}

func snapToPixels(el *scene.Element) {
	el.ComputedX = math.Round(el.ComputedX)
	el.ComputedY = math.Round(el.ComputedY)
	el.ComputedWidth = math.Round(el.ComputedWidth)
	el.ComputedHeight = math.Round(el.ComputedHeight)
	for _, child := range el.Children {
		snapToPixels(child)
	}
}

func resolveStyleForLayout(el *scene.Element, s *scene.Scene) *scene.Style {
	var classStyle *scene.Style
	if el.Class != "" && s.Styles != nil {
		classStyle = s.Styles[el.Class]
	}
	return scene.ResolveStyle(classStyle, el.Style)
}

func valOr(p *float64, def float64) float64 {
	if p != nil {
		return *p
	}
	return def
}
