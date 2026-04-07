package pattern

import (
	"encoding/json"
	"fmt"

	"github.com/KarpelesLab/aipencil/scene"
)

// Registry holds named pattern definitions.
type Registry struct {
	patterns map[string]*scene.Def
}

// NewRegistry creates a registry pre-loaded with built-in patterns.
func NewRegistry() *Registry {
	r := &Registry{patterns: make(map[string]*scene.Def)}
	r.loadBuiltins()
	return r
}

// Register adds a pattern definition.
func (r *Registry) Register(name string, def *scene.Def) {
	r.patterns[name] = def
}

// Get retrieves a pattern by name.
func (r *Registry) Get(name string) (*scene.Def, bool) {
	def, ok := r.patterns[name]
	return def, ok
}

// List returns all pattern names and their parameter definitions.
func (r *Registry) List() map[string]*scene.Def {
	return r.patterns
}

// Expand resolves all "use" elements in a scene, replacing them with
// expanded groups containing the pattern's elements.
func (r *Registry) Expand(s *scene.Scene) error {
	// Register scene-level defs (which override built-ins)
	for name, def := range s.Defs {
		r.Register(name, def)
	}

	var err error
	s.Elements, err = r.expandElements(s.Elements)
	return err
}

func (r *Registry) expandElements(elements []*scene.Element) ([]*scene.Element, error) {
	result := make([]*scene.Element, 0, len(elements))
	for _, el := range elements {
		if el.Type == "use" {
			if el.Track != "" {
				// Deferred: will be expanded after layout computes positions.
				// Pre-set dimensions from pattern def for layout sizing.
				if def, ok := r.Get(el.Pattern); ok {
					if el.Width == nil && def.Width > 0 {
						el.ComputedWidth = def.Width
					}
					if el.Height == nil && def.Height > 0 {
						el.ComputedHeight = def.Height
					}
				}
				result = append(result, el)
				continue
			}
			expanded, err := r.expandUse(el)
			if err != nil {
				return nil, err
			}
			result = append(result, expanded)
		} else {
			// Recurse into children
			if len(el.Children) > 0 {
				var err error
				el.Children, err = r.expandElements(el.Children)
				if err != nil {
					return nil, err
				}
			}
			// Recurse into layers
			for _, layer := range el.Layers {
				if len(layer.Elements) > 0 {
					var err error
					layer.Elements, err = r.expandElements(layer.Elements)
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

func (r *Registry) expandUse(el *scene.Element) (*scene.Element, error) {
	def, ok := r.Get(el.Pattern)
	if !ok {
		return nil, fmt.Errorf("unknown pattern: %q", el.Pattern)
	}

	// Merge params: defaults from def, then overrides from el.Params
	params := make(map[string]any)
	for name, pdef := range def.Params {
		if pdef.Default != nil {
			params[name] = pdef.Default
		}
	}
	for k, v := range el.Params {
		params[k] = v
	}

	// Deep-clone the pattern's elements via JSON round-trip
	elemJSON, err := json.Marshal(def.Elements)
	if err != nil {
		return nil, fmt.Errorf("pattern %q: marshal error: %w", el.Pattern, err)
	}

	// Substitute parameters
	elemJSON, err = SubstituteParams(elemJSON, params)
	if err != nil {
		return nil, fmt.Errorf("pattern %q: param substitution error: %w", el.Pattern, err)
	}

	var children []*scene.Element
	if err := json.Unmarshal(elemJSON, &children); err != nil {
		return nil, fmt.Errorf("pattern %q: unmarshal error: %w", el.Pattern, err)
	}

	// Filter by "if" conditions
	children = filterConditionals(children, params)

	// Recursively expand any nested "use" elements
	children, err = r.expandElements(children)
	if err != nil {
		return nil, err
	}

	// Create a group to wrap the expanded pattern
	group := &scene.Element{
		Type:        "group",
		ID:          el.ID,
		X:           el.X,
		Y:           el.Y,
		Width:       el.Width,
		Height:      el.Height,
		Style:       el.Style,
		Class:       el.Class,
		Position:    el.Position,
		Constraints: el.Constraints,
		Transform:   el.Transform,
		Children:    children,
	}

	// If the use element specifies a different size than the pattern's natural size,
	// add a scale transform
	if el.Width != nil && def.Width > 0 {
		scaleX := *el.Width / def.Width
		scaleY := scaleX
		if el.Height != nil && def.Height > 0 {
			scaleY = *el.Height / def.Height
		}
		if scaleX != 1 || scaleY != 1 {
			// Use uniform scale (average) for simplicity
			scale := (scaleX + scaleY) / 2
			group.Transform = &scene.Transform{Scale: &scale}
		}
	}

	return group, nil
}

// filterConditionals removes elements whose "if" condition evaluates to false.
func filterConditionals(elements []*scene.Element, params map[string]any) []*scene.Element {
	result := make([]*scene.Element, 0, len(elements))
	for _, el := range elements {
		if el.If != "" && !EvalCondition(el.If, params) {
			continue
		}
		if len(el.Children) > 0 {
			el.Children = filterConditionals(el.Children, params)
		}
		result = append(result, el)
	}
	return result
}
