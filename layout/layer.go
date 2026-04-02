package layout

import (
	"sort"

	"github.com/KarpelesLab/aipencil/scene"
)

// SortLayers sorts layers by zIndex (stable, declaration order breaks ties).
func SortLayers(layers []*scene.Layer) {
	sort.SliceStable(layers, func(i, j int) bool {
		return layers[i].ZIndex < layers[j].ZIndex
	})
}

// AllLayerElements returns all elements from all layers in z-order.
// Used by the measure and layout passes to process all elements.
func AllLayerElements(layers []*scene.Layer) []*scene.Element {
	sorted := make([]*scene.Layer, len(layers))
	copy(sorted, layers)
	SortLayers(sorted)

	var all []*scene.Element
	for _, layer := range sorted {
		all = append(all, layer.Elements...)
	}
	return all
}

// MeasureLayers measures all elements across all layers of an element.
func MeasureLayers(el *scene.Element, s *scene.Scene) {
	for _, layer := range el.Layers {
		for _, child := range layer.Elements {
			measure(child, s)
		}
	}

	// Compute container size from all layer content
	var maxW, maxH float64
	for _, layer := range el.Layers {
		for _, child := range layer.Elements {
			x := valOr(child.X, child.ComputedX)
			y := valOr(child.Y, child.ComputedY)
			right := x + child.ComputedWidth
			bottom := y + child.ComputedHeight
			if right > maxW {
				maxW = right
			}
			if bottom > maxH {
				maxH = bottom
			}
		}
	}

	if el.Width == nil && maxW > el.ComputedWidth {
		el.ComputedWidth = maxW
	}
	if el.Height == nil && maxH > el.ComputedHeight {
		el.ComputedHeight = maxH
	}
}
