package render

import (
	"fmt"
	"math"
	"strings"

	"github.com/KarpelesLab/aipencil/scene"
)

// collectIDs builds a map of element ID → *Element for arrow resolution.
func collectIDs(elements []*scene.Element) map[string]*scene.Element {
	ids := make(map[string]*scene.Element)
	for _, el := range elements {
		collectIDsRecursive(el, ids)
	}
	return ids
}

func collectIDsRecursive(el *scene.Element, ids map[string]*scene.Element) {
	if el.ID != "" {
		ids[el.ID] = el
	}
	for _, child := range el.Children {
		collectIDsRecursive(child, ids)
	}
}

// elementCenter returns the absolute center point of an element,
// walking up the tree to accumulate group translations.
func elementCenter(el *scene.Element, elements []*scene.Element) (cx, cy float64) {
	// Get the element's local center
	localCX := el.EffectiveX() + el.ComputedWidth/2
	localCY := el.EffectiveY() + el.ComputedHeight/2

	// Find parent chain and accumulate translations
	ox, oy := findParentOffset(el, elements)
	return localCX + ox, localCY + oy
}

// findParentOffset finds the accumulated translation offset for an element
// by walking the element tree.
func findParentOffset(target *scene.Element, roots []*scene.Element) (x, y float64) {
	for _, root := range roots {
		if ox, oy, found := findOffsetRecursive(target, root, 0, 0); found {
			return ox, oy
		}
	}
	return 0, 0
}

func findOffsetRecursive(target, current *scene.Element, ox, oy float64) (float64, float64, bool) {
	if current == target {
		return ox, oy, true
	}

	// Accumulate this group's position
	childOX := ox + current.EffectiveX()
	childOY := oy + current.EffectiveY()

	for _, child := range current.Children {
		if rx, ry, found := findOffsetRecursive(target, child, childOX, childOY); found {
			return rx, ry, true
		}
	}
	return 0, 0, false
}

// elementAnchor returns a specific anchor point on an element.
// anchorName can be "center", "top", "bottom", "left", "right".
func elementAnchor(el *scene.Element, anchorName string, elements []*scene.Element) (ax, ay float64) {
	cx, cy := elementCenter(el, elements)
	hw := el.ComputedWidth / 2
	hh := el.ComputedHeight / 2

	switch anchorName {
	case "top":
		return cx, cy - hh
	case "bottom":
		return cx, cy + hh
	case "left":
		return cx - hw, cy
	case "right":
		return cx + hw, cy
	default: // center
		return cx, cy
	}
}

// clipToEdge computes the point where a line from (cx,cy) to (tx,ty)
// intersects the bounding rectangle of the element.
func clipToEdge(cx, cy, hw, hh, tx, ty float64) (float64, float64) {
	dx := tx - cx
	dy := ty - cy

	if dx == 0 && dy == 0 {
		return cx, cy
	}

	// For circles, use the angle directly
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist == 0 {
		return cx, cy
	}

	// Use rectangular clipping
	var scale float64
	if hw == 0 || hh == 0 {
		return cx, cy
	}

	scaleX := math.Abs(dx) / hw
	scaleY := math.Abs(dy) / hh

	if scaleX > scaleY {
		scale = hw / math.Abs(dx)
	} else {
		scale = hh / math.Abs(dy)
	}

	return cx + dx*scale, cy + dy*scale
}

// parseAnchorRef parses "nodeId" or "nodeId.anchor" into (id, anchor).
func parseAnchorRef(ref string) (id, anchor string) {
	if idx := strings.LastIndex(ref, "."); idx >= 0 {
		candidate := ref[idx+1:]
		switch candidate {
		case "top", "bottom", "left", "right", "center":
			return ref[:idx], candidate
		}
	}
	return ref, ""
}

// renderArrows renders all arrow elements in the scene.
func renderArrows(sb *strings.Builder, elements []*scene.Element, s *scene.Scene, indent string) {
	ids := collectIDs(elements)

	for _, el := range elements {
		if el.Type == "arrow" {
			renderArrow(sb, el, ids, elements, s, indent)
		}
		// Check children too
		if len(el.Children) > 0 {
			renderArrowsInChildren(sb, el.Children, ids, elements, s, indent)
		}
	}
}

func renderArrowsInChildren(sb *strings.Builder, children []*scene.Element, ids map[string]*scene.Element, allElements []*scene.Element, s *scene.Scene, indent string) {
	for _, el := range children {
		if el.Type == "arrow" {
			renderArrow(sb, el, ids, allElements, s, indent)
		}
		if len(el.Children) > 0 {
			renderArrowsInChildren(sb, el.Children, ids, allElements, s, indent)
		}
	}
}

func renderArrow(sb *strings.Builder, el *scene.Element, ids map[string]*scene.Element, allElements []*scene.Element, s *scene.Scene, indent string) {
	fromID, fromAnchor := parseAnchorRef(el.From)
	toID, toAnchor := parseAnchorRef(el.To)

	fromEl, fromOK := ids[fromID]
	toEl, toOK := ids[toID]
	if !fromOK || !toOK {
		return // can't resolve endpoints
	}

	var x1, y1, x2, y2 float64

	if fromAnchor != "" {
		x1, y1 = elementAnchor(fromEl, fromAnchor, allElements)
	} else {
		fcx, fcy := elementCenter(fromEl, allElements)
		tcx, tcy := elementCenter(toEl, allElements)
		x1, y1 = clipToEdge(fcx, fcy, fromEl.ComputedWidth/2, fromEl.ComputedHeight/2, tcx, tcy)
	}

	if toAnchor != "" {
		x2, y2 = elementAnchor(toEl, toAnchor, allElements)
	} else {
		tcx, tcy := elementCenter(toEl, allElements)
		x2, y2 = clipToEdge(tcx, tcy, toEl.ComputedWidth/2, toEl.ComputedHeight/2, x1, y1)
	}

	style := resolveStyle(el, s)
	strokeColor := "#333"
	strokeWidth := 1.5
	if style != nil {
		if style.Stroke != "" {
			strokeColor = style.Stroke
		}
		if style.StrokeWidth != nil {
			strokeWidth = *style.StrokeWidth
		}
	}

	markerID := arrowMarkerID(el.HeadStyle, strokeColor)

	switch el.Curve {
	case "smooth":
		renderSmoothArrow(sb, x1, y1, x2, y2, strokeColor, strokeWidth, markerID, indent)
	case "orthogonal":
		renderOrthogonalArrow(sb, x1, y1, x2, y2, strokeColor, strokeWidth, markerID, indent)
	default: // straight
		renderStraightArrow(sb, x1, y1, x2, y2, strokeColor, strokeWidth, markerID, indent)
	}

	// Label
	if el.Label != "" {
		mx, my := (x1+x2)/2, (y1+y2)/2
		// Offset label slightly above the line
		my -= 8
		sb.WriteString(indent)
		fmt.Fprintf(sb, `<text x="%s" y="%s" font-size="12" text-anchor="middle" dominant-baseline="auto" fill="%s">%s</text>`,
			ff(mx), ff(my), escAttr(strokeColor), escText(el.Label))
		sb.WriteByte('\n')
	}
}

func renderStraightArrow(sb *strings.Builder, x1, y1, x2, y2 float64, stroke string, strokeWidth float64, markerID string, indent string) {
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s"`,
		ff(x1), ff(y1), ff(x2), ff(y2), escAttr(stroke), ff(strokeWidth))
	if markerID != "" {
		fmt.Fprintf(sb, ` marker-end="url(#%s)"`, markerID)
	}
	sb.WriteString("/>\n")
}

func renderSmoothArrow(sb *strings.Builder, x1, y1, x2, y2 float64, stroke string, strokeWidth float64, markerID string, indent string) {
	// Compute control points for a smooth curve
	dx := x2 - x1
	dy := y2 - y1
	dist := math.Sqrt(dx*dx + dy*dy)
	offset := dist * 0.3

	// Perpendicular offset for the control point
	nx, ny := -dy/dist, dx/dist

	cx1 := x1 + dx*0.25 + nx*offset*0.5
	cy1 := y1 + dy*0.25 + ny*offset*0.5
	cx2 := x1 + dx*0.75 + nx*offset*0.5
	cy2 := y1 + dy*0.75 + ny*offset*0.5

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<path d="M%s,%s C%s,%s %s,%s %s,%s" fill="none" stroke="%s" stroke-width="%s"`,
		ff(x1), ff(y1), ff(cx1), ff(cy1), ff(cx2), ff(cy2), ff(x2), ff(y2),
		escAttr(stroke), ff(strokeWidth))
	if markerID != "" {
		fmt.Fprintf(sb, ` marker-end="url(#%s)"`, markerID)
	}
	sb.WriteString("/>\n")
}

func renderOrthogonalArrow(sb *strings.Builder, x1, y1, x2, y2 float64, stroke string, strokeWidth float64, markerID string, indent string) {
	// Route: horizontal first, then vertical
	midX := (x1 + x2) / 2

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<path d="M%s,%s L%s,%s L%s,%s L%s,%s" fill="none" stroke="%s" stroke-width="%s"`,
		ff(x1), ff(y1), ff(midX), ff(y1), ff(midX), ff(y2), ff(x2), ff(y2),
		escAttr(stroke), ff(strokeWidth))
	if markerID != "" {
		fmt.Fprintf(sb, ` marker-end="url(#%s)"`, markerID)
	}
	sb.WriteString("/>\n")
}

// arrowMarkerID returns the marker definition ID for a given head style.
func arrowMarkerID(headStyle, color string) string {
	switch headStyle {
	case "none":
		return ""
	case "open":
		return "arrowhead-open"
	case "diamond":
		return "arrowhead-diamond"
	case "circle":
		return "arrowhead-circle"
	default: // "filled"
		return "arrowhead-filled"
	}
}

// WriteArrowDefs writes SVG <defs> for arrow markers.
func WriteArrowDefs(sb *strings.Builder, indent string) {
	sb.WriteString(indent + "<defs>\n")

	// Filled arrowhead
	sb.WriteString(indent + `  <marker id="arrowhead-filled" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">` + "\n")
	sb.WriteString(indent + `    <path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor"/>` + "\n")
	sb.WriteString(indent + "  </marker>\n")

	// Open arrowhead
	sb.WriteString(indent + `  <marker id="arrowhead-open" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">` + "\n")
	sb.WriteString(indent + `    <path d="M 0 0 L 10 5 L 0 10" fill="none" stroke="currentColor" stroke-width="1.5"/>` + "\n")
	sb.WriteString(indent + "  </marker>\n")

	// Diamond arrowhead
	sb.WriteString(indent + `  <marker id="arrowhead-diamond" viewBox="0 0 12 12" refX="6" refY="6" markerWidth="10" markerHeight="10" orient="auto-start-reverse">` + "\n")
	sb.WriteString(indent + `    <path d="M 0 6 L 6 0 L 12 6 L 6 12 z" fill="currentColor"/>` + "\n")
	sb.WriteString(indent + "  </marker>\n")

	// Circle arrowhead
	sb.WriteString(indent + `  <marker id="arrowhead-circle" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">` + "\n")
	sb.WriteString(indent + `    <circle cx="5" cy="5" r="4" fill="currentColor"/>` + "\n")
	sb.WriteString(indent + "  </marker>\n")

	sb.WriteString(indent + "</defs>\n")
}

// hasArrows checks if the scene contains any arrow elements.
func hasArrows(elements []*scene.Element) bool {
	for _, el := range elements {
		if el.Type == "arrow" {
			return true
		}
		if hasArrows(el.Children) {
			return true
		}
	}
	return false
}
