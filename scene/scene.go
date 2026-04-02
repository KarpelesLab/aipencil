package scene

// Scene is the top-level document describing an image.
type Scene struct {
	Version      string            `json:"version,omitempty"`
	Width        *float64          `json:"width,omitempty"`
	Height       *float64          `json:"height,omitempty"`
	Background   string            `json:"background,omitempty"`
	Padding      *float64          `json:"padding,omitempty"`
	PixelPerfect bool              `json:"pixelPerfect,omitempty"`
	Defs         map[string]*Def   `json:"defs,omitempty"`
	Styles       map[string]*Style `json:"styles,omitempty"`
	Elements     []*Element        `json:"elements"`
}

// Element is any node in the scene graph.
type Element struct {
	// Common fields
	ID        string     `json:"id,omitempty"`
	Type      string     `json:"type"`
	X         *float64   `json:"x,omitempty"`
	Y         *float64   `json:"y,omitempty"`
	Style     *Style     `json:"style,omitempty"`
	Class     string     `json:"class,omitempty"`
	Transform *Transform `json:"transform,omitempty"`
	Children  []*Element `json:"children,omitempty"`
	Tooltip   string     `json:"tooltip,omitempty"`

	// Shape fields
	Width  *float64 `json:"width,omitempty"`
	Height *float64 `json:"height,omitempty"`
	R      *float64 `json:"r,omitempty"`
	RX     *float64 `json:"rx,omitempty"`
	RY     *float64 `json:"ry,omitempty"`

	// Line fields
	X2 *float64 `json:"x2,omitempty"`
	Y2 *float64 `json:"y2,omitempty"`

	// Path field
	D string `json:"d,omitempty"`

	// Polygon/polyline points
	Points [][2]float64 `json:"points,omitempty"`

	// Text fields
	Text       string   `json:"text,omitempty"`
	FontSize   *float64 `json:"fontSize,omitempty"`
	FontWeight string   `json:"fontWeight,omitempty"`
	Align      string   `json:"align,omitempty"`
	MaxWidth   *float64 `json:"maxWidth,omitempty"`

	// Image fields
	Href string `json:"href,omitempty"`

	// Arrow/bubble target fields
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Target    string `json:"target,omitempty"` // bubble tail target element ID
	Label     string `json:"label,omitempty"`
	HeadStyle string `json:"headStyle,omitempty"`
	Curve     string `json:"curve,omitempty"`

	// Bubble fields
	BubbleStyle string `json:"bubbleStyle,omitempty"` // speech, thought, shout

	// Group layout
	Layout *Layout `json:"layout,omitempty"`

	// Pattern instantiation
	Pattern string         `json:"pattern,omitempty"`
	Params  map[string]any `json:"params,omitempty"`

	// Conditional (for pattern elements)
	If string `json:"if,omitempty"`

	// Viewport fields
	Clip    *bool    `json:"clip,omitempty"`    // clip content to viewport bounds (default true)
	ViewBox *ViewBox `json:"viewBox,omitempty"` // explicit content region; auto-computed if nil
	Padding *float64 `json:"padding,omitempty"` // viewport inner padding (default 10)

	// Computed by layout engine (not serialized)
	ComputedX      float64 `json:"-"`
	ComputedY      float64 `json:"-"`
	ComputedWidth  float64 `json:"-"`
	ComputedHeight float64 `json:"-"`
	Positioned     bool    `json:"-"` // true after layout has positioned this element
}

// Style holds visual properties applicable to any element.
type Style struct {
	Fill            string   `json:"fill,omitempty"`
	Stroke          string   `json:"stroke,omitempty"`
	StrokeWidth     *float64 `json:"strokeWidth,omitempty"`
	Opacity         *float64 `json:"opacity,omitempty"`
	FontSize        *float64 `json:"fontSize,omitempty"`
	FontWeight      string   `json:"fontWeight,omitempty"`
	FontFamily      string   `json:"fontFamily,omitempty"`
	TextAnchor      string   `json:"textAnchor,omitempty"`
	RX              *float64 `json:"rx,omitempty"`
	RY              *float64 `json:"ry,omitempty"`
	StrokeDasharray string   `json:"strokeDasharray,omitempty"`
	StrokeLinecap   string   `json:"strokeLinecap,omitempty"`
	StrokeLinejoin  string   `json:"strokeLinejoin,omitempty"`
	Filter          string   `json:"filter,omitempty"`
	ImageRendering  string   `json:"imageRendering,omitempty"`
}

// Transform holds geometric transformations.
type Transform struct {
	Translate [2]float64 `json:"translate,omitempty"`
	Rotate    *float64   `json:"rotate,omitempty"`
	Scale     *float64   `json:"scale,omitempty"`
}

// Layout controls how a group positions its children.
type Layout struct {
	Type       string  `json:"type"`
	Gap        float64 `json:"gap,omitempty"`
	Align      string  `json:"align,omitempty"`
	Columns    int     `json:"columns,omitempty"`
	CellWidth  float64 `json:"cellWidth,omitempty"`
	CellHeight float64 `json:"cellHeight,omitempty"`
}

// ViewBox defines the content coordinate region for a viewport.
type ViewBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Def is a reusable pattern/template definition.
type Def struct {
	Params   map[string]*ParamDef `json:"params,omitempty"`
	Width    float64              `json:"width,omitempty"`
	Height   float64              `json:"height,omitempty"`
	Elements []*Element           `json:"elements"`
}

// ParamDef describes a template parameter.
type ParamDef struct {
	Type    string   `json:"type"`
	Default any      `json:"default,omitempty"`
	Enum    []string `json:"enum,omitempty"`
}

// Helper to get the effective X position of an element.
func (e *Element) EffectiveX() float64 {
	if e.X != nil {
		return *e.X
	}
	return e.ComputedX
}

// Helper to get the effective Y position of an element.
func (e *Element) EffectiveY() float64 {
	if e.Y != nil {
		return *e.Y
	}
	return e.ComputedY
}

// Helper to get the effective width.
func (e *Element) EffectiveWidth() float64 {
	if e.Width != nil {
		return *e.Width
	}
	return e.ComputedWidth
}

// Helper to get the effective height.
func (e *Element) EffectiveHeight() float64 {
	if e.Height != nil {
		return *e.Height
	}
	return e.ComputedHeight
}

// Ptr returns a pointer to v. Useful for initializing *float64 fields.
func Ptr(v float64) *float64 {
	return &v
}

// ResolveStyle merges a class style with an inline style overlay.
// Class provides defaults; inline overrides non-zero fields.
func ResolveStyle(class *Style, inline *Style) *Style {
	if class == nil && inline == nil {
		return nil
	}
	if class == nil {
		return inline
	}
	if inline == nil {
		return class
	}
	merged := *class
	if inline.Fill != "" {
		merged.Fill = inline.Fill
	}
	if inline.Stroke != "" {
		merged.Stroke = inline.Stroke
	}
	if inline.StrokeWidth != nil {
		merged.StrokeWidth = inline.StrokeWidth
	}
	if inline.Opacity != nil {
		merged.Opacity = inline.Opacity
	}
	if inline.FontSize != nil {
		merged.FontSize = inline.FontSize
	}
	if inline.FontWeight != "" {
		merged.FontWeight = inline.FontWeight
	}
	if inline.FontFamily != "" {
		merged.FontFamily = inline.FontFamily
	}
	if inline.TextAnchor != "" {
		merged.TextAnchor = inline.TextAnchor
	}
	if inline.RX != nil {
		merged.RX = inline.RX
	}
	if inline.RY != nil {
		merged.RY = inline.RY
	}
	if inline.StrokeDasharray != "" {
		merged.StrokeDasharray = inline.StrokeDasharray
	}
	if inline.StrokeLinecap != "" {
		merged.StrokeLinecap = inline.StrokeLinecap
	}
	if inline.StrokeLinejoin != "" {
		merged.StrokeLinejoin = inline.StrokeLinejoin
	}
	if inline.Filter != "" {
		merged.Filter = inline.Filter
	}
	if inline.ImageRendering != "" {
		merged.ImageRendering = inline.ImageRendering
	}
	return &merged
}
