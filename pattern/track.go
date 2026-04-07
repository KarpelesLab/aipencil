package pattern

import (
	"fmt"
	"math"

	"github.com/KarpelesLab/aipencil/scene"
)

// ResolveTrack expands deferred "use" elements that have a track field.
// It computes the facing direction based on the relative position of the
// tracked target and injects an "angle" param before expanding.
// Must be called after layout has computed positions.
func (r *Registry) ResolveTrack(s *scene.Scene) error {
	ids := collectIDs(s.Elements)
	var err error
	s.Elements, err = r.resolveTrackElements(s.Elements, ids)
	return err
}

func (r *Registry) resolveTrackElements(elements []*scene.Element, ids map[string]*scene.Element) ([]*scene.Element, error) {
	result := make([]*scene.Element, 0, len(elements))
	for _, el := range elements {
		if el.Type == "use" && el.Track != "" {
			target, ok := ids[el.Track]
			if !ok {
				return nil, fmt.Errorf("track target %q not found", el.Track)
			}

			// Check if the pattern declares an "angle" param
			def, ok := r.Get(el.Pattern)
			if !ok {
				return nil, fmt.Errorf("unknown pattern: %q", el.Pattern)
			}

			if _, hasAngle := def.Params["angle"]; hasAngle {
				// Compute direction from source to target
				srcX := el.EffectiveX() + el.ComputedWidth/2
				srcY := el.EffectiveY() + el.ComputedHeight/2
				tgtX := target.EffectiveX() + target.ComputedWidth/2
				tgtY := target.EffectiveY() + target.ComputedHeight/2

				angle := computeAngle(srcX, srcY, tgtX, tgtY)

				// Inject angle param (only if not explicitly set)
				if el.Params == nil {
					el.Params = make(map[string]any)
				}
				if _, explicit := el.Params["angle"]; !explicit {
					el.Params["angle"] = angle
				}
			}

			// Now expand the pattern with the computed angle
			expanded, err := r.expandUse(el)
			if err != nil {
				return nil, err
			}

			// Transfer computed layout values
			expanded.ComputedX = el.ComputedX
			expanded.ComputedY = el.ComputedY
			expanded.ComputedWidth = el.ComputedWidth
			expanded.ComputedHeight = el.ComputedHeight
			expanded.Positioned = el.Positioned

			result = append(result, expanded)
		} else {
			// Recurse into children
			if len(el.Children) > 0 {
				var err error
				el.Children, err = r.resolveTrackElements(el.Children, ids)
				if err != nil {
					return nil, err
				}
			}
			// Recurse into layers
			for _, layer := range el.Layers {
				if len(layer.Elements) > 0 {
					var err error
					layer.Elements, err = r.resolveTrackElements(layer.Elements, ids)
					if err != nil {
						return nil, err
					}
				}
			}
			result = append(result, el)
		}
	}
	return result, nil
}

// computeAngle determines the facing direction based on relative position.
// Returns one of: "front", "quarter-right", "right", "quarter-left", "left".
// "back" is never auto-computed (only available via explicit param).
func computeAngle(srcX, srcY, tgtX, tgtY float64) string {
	dx := tgtX - srcX
	dy := tgtY - srcY
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < 5 {
		return "front"
	}

	ratio := math.Abs(dx) / dist

	if ratio < 0.3 {
		return "front" // mostly vertical offset
	}

	if ratio < 0.7 {
		// diagonal
		if dx > 0 {
			return "quarter-right"
		}
		return "quarter-left"
	}

	// mostly horizontal
	if dx > 0 {
		return "right"
	}
	return "left"
}

// collectIDs builds a map of element ID → element for the entire scene tree.
func collectIDs(elements []*scene.Element) map[string]*scene.Element {
	ids := make(map[string]*scene.Element)
	var walk func([]*scene.Element)
	walk = func(els []*scene.Element) {
		for _, el := range els {
			if el.ID != "" {
				ids[el.ID] = el
			}
			walk(el.Children)
			for _, layer := range el.Layers {
				walk(layer.Elements)
			}
		}
	}
	walk(elements)
	return ids
}
