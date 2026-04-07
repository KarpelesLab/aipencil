package scene

import (
	"fmt"
	"strings"
)

var knownTypes = map[string]bool{
	"rect":     true,
	"circle":   true,
	"ellipse":  true,
	"line":     true,
	"path":     true,
	"polygon":  true,
	"polyline": true,
	"text":     true,
	"image":    true,
	"group":    true,
	"use":      true,
	"arrow":    true,
	"panel":    true,
	"bubble":   true,
	"viewport": true,
}

var knownStyles = map[string]bool{
	"":         true,
	"default":  true,
	"comic":    true,
	"manga":    true,
	"stickman": true,
	"cute":     true,
}

// Validate checks a parsed Scene for structural errors.
// Returns a list of error strings (empty = valid).
func Validate(s *Scene) []string {
	var errs []string

	if s.ArtStyle != "" && !knownStyles[s.ArtStyle] {
		errs = append(errs, fmt.Sprintf("unknown artStyle %q (known: default, comic, manga, stickman, cute)", s.ArtStyle))
	}

	ids := make(map[string]bool)
	for i, el := range s.Elements {
		errs = append(errs, validateElement(el, fmt.Sprintf("elements[%d]", i), ids)...)
	}
	return errs
}

func validateElement(el *Element, path string, ids map[string]bool) []string {
	var errs []string

	if el.Type == "" {
		errs = append(errs, fmt.Sprintf("%s: missing required field 'type'", path))
		return errs
	}

	if !knownTypes[el.Type] {
		errs = append(errs, fmt.Sprintf("%s: unknown type %q (known: %s)", path, el.Type, knownTypeList()))
	}

	if el.ID != "" {
		if ids[el.ID] {
			errs = append(errs, fmt.Sprintf("%s: duplicate id %q", path, el.ID))
		}
		ids[el.ID] = true
	}

	switch el.Type {
	case "path":
		if el.D == "" {
			errs = append(errs, fmt.Sprintf("%s: path element requires 'd' field", path))
		}
	case "polygon", "polyline":
		if len(el.Points) < 2 {
			errs = append(errs, fmt.Sprintf("%s: %s requires at least 2 points", path, el.Type))
		}
	case "arrow":
		if el.From == "" || el.To == "" {
			errs = append(errs, fmt.Sprintf("%s: arrow requires 'from' and 'to' fields", path))
		}
	case "use":
		if el.Pattern == "" {
			errs = append(errs, fmt.Sprintf("%s: use element requires 'pattern' field", path))
		}
	case "bubble":
		if el.Text == "" {
			errs = append(errs, fmt.Sprintf("%s: bubble requires 'text' field", path))
		}
	case "panel":
		if len(el.Children) == 0 {
			errs = append(errs, fmt.Sprintf("%s: panel requires children", path))
		}
	case "image":
		if el.Href == "" {
			errs = append(errs, fmt.Sprintf("%s: image element requires 'href' field", path))
		}
	}

	if el.Layout != nil {
		validLayouts := map[string]bool{
			"free": true, "row": true, "column": true,
			"grid": true, "stack": true, "graph": true,
			"constrained": true,
			"ranked": true,
		}
		if !validLayouts[el.Layout.Type] {
			errs = append(errs, fmt.Sprintf("%s: unknown layout type %q", path, el.Layout.Type))
		}
	}

	// Layers and children are mutually exclusive
	if len(el.Layers) > 0 && len(el.Children) > 0 {
		errs = append(errs, fmt.Sprintf("%s: cannot have both 'layers' and 'children'", path))
	}

	for i, child := range el.Children {
		errs = append(errs, validateElement(child, fmt.Sprintf("%s.children[%d]", path, i), ids)...)
	}

	for i, layer := range el.Layers {
		lpath := fmt.Sprintf("%s.layers[%d]", path, i)
		for j, child := range layer.Elements {
			errs = append(errs, validateElement(child, fmt.Sprintf("%s.elements[%d]", lpath, j), ids)...)
		}
	}

	return errs
}

func knownTypeList() string {
	types := make([]string, 0, len(knownTypes))
	for t := range knownTypes {
		types = append(types, t)
	}
	// Sort for deterministic output
	for i := 0; i < len(types); i++ {
		for j := i + 1; j < len(types); j++ {
			if types[i] > types[j] {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
	return strings.Join(types, ", ")
}
