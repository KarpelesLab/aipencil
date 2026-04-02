package layout

import "github.com/KarpelesLab/aipencil/scene"

// IDMap maps element IDs to their elements and tracks parent relationships.
type IDMap struct {
	Elements  map[string]*scene.Element
	Parent    map[*scene.Element]*scene.Element // child → parent container
	SceneRoot *scene.Element                    // virtual root for scene-level parent refs
}

// BuildIDMap walks the entire element tree (including layers) and builds
// a global ID registry.
func BuildIDMap(elements []*scene.Element, s *scene.Scene) *IDMap {
	// Virtual root element representing the scene canvas
	root := &scene.Element{
		ID:   "_scene_root",
		Type: "group",
	}
	if s.Width != nil {
		root.Width = s.Width
		root.ComputedWidth = *s.Width
	}
	if s.Height != nil {
		root.Height = s.Height
		root.ComputedHeight = *s.Height
	}

	m := &IDMap{
		Elements:  make(map[string]*scene.Element),
		Parent:    make(map[*scene.Element]*scene.Element),
		SceneRoot: root,
	}
	for _, el := range elements {
		m.walk(el, root)
	}
	return m
}

func (m *IDMap) walk(el *scene.Element, parent *scene.Element) {
	if el.ID != "" {
		m.Elements[el.ID] = el
	}
	if parent != nil {
		m.Parent[el] = parent
	}
	for _, child := range el.Children {
		m.walk(child, el)
	}
	for _, layer := range el.Layers {
		for _, child := range layer.Elements {
			m.walk(child, el)
		}
	}
}

// ParentSize returns the computed width and height of an element's parent container.
func (m *IDMap) ParentSize(el *scene.Element) (float64, float64) {
	if p, ok := m.Parent[el]; ok {
		return p.EffectiveWidth(), p.EffectiveHeight()
	}
	return 0, 0
}
