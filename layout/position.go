package layout

import (
	"strconv"
	"strings"

	"github.com/KarpelesLab/aipencil/scene"
)

// ResolvePositions resolves relative positioning for all elements.
// Must be called after the measure pass.
func ResolvePositions(elements []*scene.Element, idMap *IDMap) {
	for _, el := range elements {
		resolvePositionRecursive(el, idMap)
	}
}

func resolvePositionRecursive(el *scene.Element, idMap *IDMap) {
	// Recurse into children first (bottom-up)
	for _, child := range el.Children {
		resolvePositionRecursive(child, idMap)
	}
	for _, layer := range el.Layers {
		for _, child := range layer.Elements {
			resolvePositionRecursive(child, idMap)
		}
	}

	if el.Position == nil {
		return
	}

	pos := el.Position
	parentW, parentH := idMap.ParentSize(el)

	// Start with current position
	x := valOr(el.X, el.ComputedX)
	y := valOr(el.Y, el.ComputedY)
	w := el.ComputedWidth
	h := el.ComputedHeight

	// Percentage-based positioning
	if pos.X != "" {
		x = parsePositionValue(pos.X, parentW)
	}
	if pos.Y != "" {
		y = parsePositionValue(pos.Y, parentH)
	}

	// CenterOn: center this element on another element
	if pos.CenterOn != "" {
		if ref, ok := idMap.Elements[pos.CenterOn]; ok {
			refCX := ref.EffectiveX() + ref.ComputedWidth/2
			refCY := ref.EffectiveY() + ref.ComputedHeight/2
			x = refCX - w/2
			y = refCY - h/2
		}
	}

	// Relative positioning (below, above, rightOf, leftOf)
	if pos.Below != "" {
		if ref, ok := idMap.Elements[pos.Below]; ok {
			y = ref.EffectiveY() + ref.ComputedHeight + pos.Gap
			// Horizontal alignment
			x = alignRelativeX(pos.AlignX, el, ref, x)
		}
	}
	if pos.Above != "" {
		if ref, ok := idMap.Elements[pos.Above]; ok {
			y = ref.EffectiveY() - h - pos.Gap
			x = alignRelativeX(pos.AlignX, el, ref, x)
		}
	}
	if pos.RightOf != "" {
		if ref, ok := idMap.Elements[pos.RightOf]; ok {
			x = ref.EffectiveX() + ref.ComputedWidth + pos.Gap
			y = alignRelativeY(pos.AlignY, el, ref, y)
		}
	}
	if pos.LeftOf != "" {
		if ref, ok := idMap.Elements[pos.LeftOf]; ok {
			x = ref.EffectiveX() - w - pos.Gap
			y = alignRelativeY(pos.AlignY, el, ref, y)
		}
	}

	// Anchor adjustment: shift so the named anchor point lands at (x, y)
	if pos.Anchor != "" {
		ax, ay := anchorOffset(pos.Anchor, w, h)
		x -= ax
		y -= ay
	}

	// Final offsets
	x += pos.OffsetX
	y += pos.OffsetY

	el.ComputedX = x
	el.ComputedY = y
	el.Positioned = true
}

// parsePositionValue parses "50%" or "100" into a float64.
func parsePositionValue(s string, parentSize float64) float64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		pct, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil {
			return 0
		}
		return parentSize * pct / 100
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// anchorOffset returns how much to shift (x, y) so the named anchor point
// of an element of size (w, h) lands at the target position.
func anchorOffset(anchor string, w, h float64) (float64, float64) {
	switch anchor {
	case "top-left":
		return 0, 0
	case "top-center":
		return w / 2, 0
	case "top-right":
		return w, 0
	case "center-left":
		return 0, h / 2
	case "center":
		return w / 2, h / 2
	case "center-right":
		return w, h / 2
	case "bottom-left":
		return 0, h
	case "bottom-center":
		return w / 2, h
	case "bottom-right":
		return w, h
	default:
		return 0, 0
	}
}

func alignRelativeX(alignX string, el, ref *scene.Element, defaultX float64) float64 {
	switch alignX {
	case "left":
		return ref.EffectiveX()
	case "center":
		return ref.EffectiveX() + ref.ComputedWidth/2 - el.ComputedWidth/2
	case "right":
		return ref.EffectiveX() + ref.ComputedWidth - el.ComputedWidth
	default: // keep current or align center as default for below/above
		return ref.EffectiveX() + ref.ComputedWidth/2 - el.ComputedWidth/2
	}
}

func alignRelativeY(alignY string, el, ref *scene.Element, defaultY float64) float64 {
	switch alignY {
	case "top":
		return ref.EffectiveY()
	case "center":
		return ref.EffectiveY() + ref.ComputedHeight/2 - el.ComputedHeight/2
	case "bottom":
		return ref.EffectiveY() + ref.ComputedHeight - el.ComputedHeight
	default: // keep current or align center as default for rightOf/leftOf
		return ref.EffectiveY() + ref.ComputedHeight/2 - el.ComputedHeight/2
	}
}
