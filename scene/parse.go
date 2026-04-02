package scene

import (
	"encoding/json"
	"fmt"
	"io"
)

// Parse reads JSON from r and returns a Scene with defaults applied.
func Parse(r io.Reader) (*Scene, error) {
	var s Scene
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse scene: %w", err)
	}
	applyDefaults(&s)
	return &s, nil
}

// ParseBytes parses a JSON byte slice into a Scene.
func ParseBytes(data []byte) (*Scene, error) {
	var s Scene
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scene: %w", err)
	}
	applyDefaults(&s)
	return &s, nil
}

func applyDefaults(s *Scene) {
	if s.Version == "" {
		s.Version = "1"
	}
	if s.Background == "" {
		s.Background = "#ffffff"
	}
	if s.Padding == nil {
		s.Padding = Ptr(20)
	}

	for _, el := range s.Elements {
		applyElementDefaults(el)
	}
}

func applyElementDefaults(el *Element) {
	switch el.Type {
	case "rect":
		// Don't force default sizes — let layout auto-size from context.
		// Only set defaults for standalone rects (no parent stack/group).
	case "circle":
		if el.R == nil {
			el.R = Ptr(25)
		}
	case "ellipse":
		if el.RX == nil {
			el.RX = Ptr(30)
		}
		if el.RY == nil {
			el.RY = Ptr(20)
		}
	case "text":
		// Don't set default FontSize here — let the layout/render resolve
		// from style cascade with 14 as the ultimate fallback
	case "arrow":
		if el.HeadStyle == "" {
			el.HeadStyle = "filled"
		}
		if el.Curve == "" {
			el.Curve = "straight"
		}
	case "bubble":
		if el.BubbleStyle == "" {
			el.BubbleStyle = "speech"
		}
	case "panel":
		if el.Width == nil {
			el.Width = Ptr(300)
		}
		if el.Height == nil {
			el.Height = Ptr(200)
		}
	case "viewport":
		// viewport size can be set by layout; no hard defaults
	}

	if el.Layout != nil && el.Layout.Type == "" {
		el.Layout.Type = "free"
	}

	for _, child := range el.Children {
		applyElementDefaults(child)
	}
	for _, layer := range el.Layers {
		for _, child := range layer.Elements {
			applyElementDefaults(child)
		}
	}
}
