package render

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"

	"github.com/KarpelesLab/aipencil/scene"
)

var clipIDCounter atomic.Int64

// RenderSVG produces an SVG string from a laid-out Scene.
func RenderSVG(s *scene.Scene) string {
	var sb strings.Builder

	w := s.Width
	h := s.Height
	if w == nil || h == nil {
		// Compute bounding box from elements
		bw, bh := computeBounds(s)
		pad := 20.0
		if s.Padding != nil {
			pad = *s.Padding
		}
		if w == nil {
			v := bw + pad*2
			w = &v
		}
		if h == nil {
			v := bh + pad*2
			h = &v
		}
	}

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s">`, ff(*w), ff(*h), ff(*w), ff(*h))
	sb.WriteByte('\n')

	// Arrow marker definitions
	if hasArrows(s.Elements) {
		WriteArrowDefs(&sb, "  ")
	}

	// Background
	if s.Background != "" && s.Background != "none" {
		fmt.Fprintf(&sb, `  <rect width="%s" height="%s" fill="%s"/>`, ff(*w), ff(*h), escAttr(s.Background))
		sb.WriteByte('\n')
	}

	// Render non-arrow/non-bubble elements
	for _, el := range s.Elements {
		if el.Type != "arrow" && el.Type != "bubble" {
			renderElement(&sb, el, s, "  ")
		}
	}

	// Render arrows on top of elements
	renderArrows(&sb, s.Elements, s, "  ")

	// Render bubbles on top of everything
	renderBubbles(&sb, s.Elements, s.Elements, s, "  ")

	sb.WriteString("</svg>\n")
	return sb.String()
}

func renderElement(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	switch el.Type {
	case "rect":
		renderRect(sb, el, s, indent)
	case "circle":
		renderCircle(sb, el, s, indent)
	case "ellipse":
		renderEllipse(sb, el, s, indent)
	case "line":
		renderLine(sb, el, s, indent)
	case "path":
		renderPath(sb, el, s, indent)
	case "polygon":
		renderPolygon(sb, el, s, indent)
	case "polyline":
		renderPolyline(sb, el, s, indent)
	case "text":
		renderText(sb, el, s, indent)
	case "image":
		renderImage(sb, el, s, indent)
	case "group":
		renderGroup(sb, el, s, indent)
	case "panel":
		renderPanel(sb, el, s, indent)
	}
}

func renderRect(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	x, y := el.EffectiveX(), el.EffectiveY()
	w := el.EffectiveWidth()
	h := el.EffectiveHeight()

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<rect x="%s" y="%s" width="%s" height="%s"`, ff(x), ff(y), ff(w), ff(h))

	style := resolveStyle(el, s)

	// Corner radius from style or element
	rx := 0.0
	if el.RX != nil {
		rx = *el.RX
	} else if style != nil && style.RX != nil {
		rx = *style.RX
	}
	if rx > 0 {
		fmt.Fprintf(sb, ` rx="%s"`, ff(rx))
	}

	writeStyleAttrs(sb, style)
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderCircle(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	r := 25.0
	if el.R != nil {
		r = *el.R
	}
	cx := el.EffectiveX() + r
	cy := el.EffectiveY() + r

	// If explicit x/y was given, treat as center coordinates
	if el.X != nil {
		cx = *el.X
	}
	if el.Y != nil {
		cy = *el.Y
	}

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<circle cx="%s" cy="%s" r="%s"`, ff(cx), ff(cy), ff(r))
	writeStyleAttrs(sb, resolveStyle(el, s))
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderEllipse(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	rx, ry := 30.0, 20.0
	if el.RX != nil {
		rx = *el.RX
	}
	if el.RY != nil {
		ry = *el.RY
	}
	cx := el.EffectiveX() + rx
	cy := el.EffectiveY() + ry
	if el.X != nil {
		cx = *el.X
	}
	if el.Y != nil {
		cy = *el.Y
	}

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s"`, ff(cx), ff(cy), ff(rx), ff(ry))
	writeStyleAttrs(sb, resolveStyle(el, s))
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderLine(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	x1, y1 := el.EffectiveX(), el.EffectiveY()
	x2, y2 := 0.0, 0.0
	if el.X2 != nil {
		x2 = *el.X2
	}
	if el.Y2 != nil {
		y2 = *el.Y2
	}

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<line x1="%s" y1="%s" x2="%s" y2="%s"`, ff(x1), ff(y1), ff(x2), ff(y2))
	style := resolveStyle(el, s)
	if style == nil {
		style = &scene.Style{Stroke: "#000", StrokeWidth: scene.Ptr(1)}
	} else if style.Stroke == "" {
		style.Stroke = "#000"
	}
	writeStyleAttrs(sb, style)
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderPath(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<path d="%s"`, escAttr(el.D))
	writeStyleAttrs(sb, resolveStyle(el, s))
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderPolygon(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<polygon points="%s"`, pointsStr(el.Points))
	writeStyleAttrs(sb, resolveStyle(el, s))
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderPolyline(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<polyline points="%s"`, pointsStr(el.Points))
	style := resolveStyle(el, s)
	if style == nil {
		style = &scene.Style{Fill: "none", Stroke: "#000", StrokeWidth: scene.Ptr(1)}
	}
	writeStyleAttrs(sb, style)
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderText(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	style := resolveStyle(el, s)

	fontSize := 14.0
	if el.FontSize != nil {
		fontSize = *el.FontSize
	} else if style != nil && style.FontSize != nil {
		fontSize = *style.FontSize
	}

	// Text anchor from align
	anchor := "start"
	if el.Align == "center" {
		anchor = "middle"
	} else if el.Align == "right" {
		anchor = "end"
	}
	if style != nil && style.TextAnchor != "" {
		anchor = style.TextAnchor
	}

	// Convert layout bounding-box position to SVG text position.
	// Layout gives us the top-left of the text bounding box;
	// SVG text x/y is the anchor/baseline point.
	x, y := el.EffectiveX(), el.EffectiveY()

	// Adjust x for text-anchor
	switch anchor {
	case "middle":
		x += el.ComputedWidth / 2
	case "end":
		x += el.ComputedWidth
	}

	// Adjust y: dominant-baseline="central" means y is the vertical center
	y += el.ComputedHeight / 2

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<text x="%s" y="%s" font-size="%s"`, ff(x), ff(y), ff(fontSize))
	fmt.Fprintf(sb, ` text-anchor="%s"`, anchor)
	sb.WriteString(` dominant-baseline="central"`)

	fontWeight := el.FontWeight
	if fontWeight == "" && style != nil {
		fontWeight = style.FontWeight
	}
	if fontWeight != "" {
		fmt.Fprintf(sb, ` font-weight="%s"`, escAttr(fontWeight))
	}

	fontFamily := ""
	if style != nil {
		fontFamily = style.FontFamily
	}
	if fontFamily != "" {
		fmt.Fprintf(sb, ` font-family="%s"`, escAttr(fontFamily))
	}

	writeStyleAttrsText(sb, style)
	writeTransform(sb, el.Transform)

	// Handle multi-line text
	lines := strings.Split(el.Text, "\n")
	if len(lines) == 1 {
		sb.WriteByte('>')
		sb.WriteString(escText(el.Text))
		sb.WriteString("</text>\n")
	} else {
		// Shift y up so the text block is vertically centered.
		// Total block height = (n-1) * lineSpacing. First line is at y,
		// last line at y + (n-1)*lineSpacing. Center of block is at
		// y + (n-1)*lineSpacing/2. We want that center at the original y,
		// so offset = -(n-1)*lineSpacing/2.
		lineSpacing := fontSize * 1.4
		offsetY := -float64(len(lines)-1) * lineSpacing / 2
		fmt.Fprintf(sb, ` dy="%s"`, ff(offsetY))

		sb.WriteString(">\n")
		for i, line := range lines {
			dy := "0"
			if i > 0 {
				dy = ff(lineSpacing)
			}
			fmt.Fprintf(sb, `%s  <tspan x="%s" dy="%s">%s</tspan>`+"\n", indent, ff(x), dy, escText(line))
		}
		sb.WriteString(indent)
		sb.WriteString("</text>\n")
	}
}

func renderImage(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	x, y := el.EffectiveX(), el.EffectiveY()
	w := el.EffectiveWidth()
	h := el.EffectiveHeight()

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<image href="%s" x="%s" y="%s" width="%s" height="%s"`, escAttr(el.Href), ff(x), ff(y), ff(w), ff(h))

	style := resolveStyle(el, s)
	if style != nil && style.ImageRendering != "" {
		fmt.Fprintf(sb, ` image-rendering="%s"`, escAttr(style.ImageRendering))
	}
	writeTransform(sb, el.Transform)
	sb.WriteString("/>\n")
}

func renderGroup(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	sb.WriteString(indent)
	sb.WriteString("<g")

	if el.ID != "" {
		fmt.Fprintf(sb, ` id="%s"`, escAttr(el.ID))
	}

	// Group-level transform combines explicit transform + position offset
	hasTranslate := false
	x, y := el.EffectiveX(), el.EffectiveY()
	if x != 0 || y != 0 {
		hasTranslate = true
	}

	if el.Transform != nil || hasTranslate {
		sb.WriteString(` transform="`)
		if hasTranslate {
			fmt.Fprintf(sb, "translate(%s,%s)", ff(x), ff(y))
		}
		if el.Transform != nil {
			if hasTranslate {
				sb.WriteByte(' ')
			}
			writeTransformInline(sb, el.Transform)
		}
		sb.WriteByte('"')
	}

	writeStyleAttrs(sb, resolveStyle(el, s))
	sb.WriteString(">\n")

	for _, child := range el.Children {
		renderElement(sb, child, s, indent+"  ")
	}

	sb.WriteString(indent)
	sb.WriteString("</g>\n")
}

func renderPanel(sb *strings.Builder, el *scene.Element, s *scene.Scene, indent string) {
	x, y := el.EffectiveX(), el.EffectiveY()
	w := el.EffectiveWidth()
	h := el.EffectiveHeight()
	clipID := fmt.Sprintf("panel-clip-%d", clipIDCounter.Add(1))

	style := resolveStyle(el, s)
	borderStroke := "#000"
	borderWidth := 2.0
	if style != nil {
		if style.Stroke != "" {
			borderStroke = style.Stroke
		}
		if style.StrokeWidth != nil {
			borderWidth = *style.StrokeWidth
		}
	}

	sb.WriteString(indent)
	sb.WriteString("<g")
	if el.ID != "" {
		fmt.Fprintf(sb, ` id="%s"`, escAttr(el.ID))
	}
	sb.WriteString(">\n")

	// ClipPath in local coordinates (the translate on the content group handles positioning)
	fmt.Fprintf(sb, "%s  <defs><clipPath id=\"%s\"><rect x=\"0\" y=\"0\" width=\"%s\" height=\"%s\"/></clipPath></defs>\n",
		indent, clipID, ff(w), ff(h))

	// Clipped content group — translate so children are in panel-local coordinates
	fmt.Fprintf(sb, "%s  <g clip-path=\"url(#%s)\" transform=\"translate(%s,%s)\">\n", indent, clipID, ff(x), ff(y))
	for _, child := range el.Children {
		renderElement(sb, child, s, indent+"    ")
	}
	fmt.Fprintf(sb, "%s  </g>\n", indent)

	// Border on top
	fmt.Fprintf(sb, "%s  <rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" fill=\"none\" stroke=\"%s\" stroke-width=\"%s\"/>\n",
		indent, ff(x), ff(y), ff(w), ff(h), escAttr(borderStroke), ff(borderWidth))

	sb.WriteString(indent)
	sb.WriteString("</g>\n")
}

// resolveStyle merges class + inline styles for an element.
func resolveStyle(el *scene.Element, s *scene.Scene) *scene.Style {
	var classStyle *scene.Style
	if el.Class != "" && s.Styles != nil {
		classStyle = s.Styles[el.Class]
	}
	return scene.ResolveStyle(classStyle, el.Style)
}

// writeStyleAttrs writes fill, stroke, etc. as SVG attributes.
func writeStyleAttrs(sb *strings.Builder, style *scene.Style) {
	if style == nil {
		return
	}
	if style.Fill != "" {
		fmt.Fprintf(sb, ` fill="%s"`, escAttr(style.Fill))
	}
	if style.Opacity != nil {
		fmt.Fprintf(sb, ` opacity="%s"`, ff(*style.Opacity))
	}
	if style.Stroke != "" {
		fmt.Fprintf(sb, ` stroke="%s"`, escAttr(style.Stroke))
	}
	if style.StrokeWidth != nil {
		fmt.Fprintf(sb, ` stroke-width="%s"`, ff(*style.StrokeWidth))
	}
	if style.StrokeDasharray != "" {
		fmt.Fprintf(sb, ` stroke-dasharray="%s"`, escAttr(style.StrokeDasharray))
	}
	if style.StrokeLinecap != "" {
		fmt.Fprintf(sb, ` stroke-linecap="%s"`, escAttr(style.StrokeLinecap))
	}
	if style.StrokeLinejoin != "" {
		fmt.Fprintf(sb, ` stroke-linejoin="%s"`, escAttr(style.StrokeLinejoin))
	}
	if style.Filter != "" {
		fmt.Fprintf(sb, ` filter="%s"`, escAttr(style.Filter))
	}
}

// writeStyleAttrsText writes text-specific style attributes (excluding fill/stroke which are handled by writeStyleAttrs).
func writeStyleAttrsText(sb *strings.Builder, style *scene.Style) {
	if style == nil {
		return
	}
	// Only write fill/stroke for text (not font attrs, which are written directly)
	if style.Fill != "" {
		fmt.Fprintf(sb, ` fill="%s"`, escAttr(style.Fill))
	}
	if style.Stroke != "" {
		fmt.Fprintf(sb, ` stroke="%s"`, escAttr(style.Stroke))
	}
	if style.Opacity != nil {
		fmt.Fprintf(sb, ` opacity="%s"`, ff(*style.Opacity))
	}
}

func writeTransform(sb *strings.Builder, t *scene.Transform) {
	if t == nil {
		return
	}
	sb.WriteString(` transform="`)
	writeTransformInline(sb, t)
	sb.WriteByte('"')
}

func writeTransformInline(sb *strings.Builder, t *scene.Transform) {
	parts := []string{}
	if t.Translate[0] != 0 || t.Translate[1] != 0 {
		parts = append(parts, fmt.Sprintf("translate(%s,%s)", ff(t.Translate[0]), ff(t.Translate[1])))
	}
	if t.Rotate != nil && *t.Rotate != 0 {
		parts = append(parts, fmt.Sprintf("rotate(%s)", ff(*t.Rotate)))
	}
	if t.Scale != nil && *t.Scale != 0 && *t.Scale != 1 {
		parts = append(parts, fmt.Sprintf("scale(%s)", ff(*t.Scale)))
	}
	sb.WriteString(strings.Join(parts, " "))
}

// computeBounds calculates the bounding box of all elements.
func computeBounds(s *scene.Scene) (width, height float64) {
	for _, el := range s.Elements {
		w, h := elementBounds(el)
		if w > width {
			width = w
		}
		if h > height {
			height = h
		}
	}
	if width == 0 {
		width = 100
	}
	if height == 0 {
		height = 100
	}
	return
}

func elementBounds(el *scene.Element) (maxX, maxY float64) {
	x, y := el.EffectiveX(), el.EffectiveY()

	switch el.Type {
	case "rect", "image":
		maxX = x + el.EffectiveWidth()
		maxY = y + el.EffectiveHeight()
	case "circle":
		r := 25.0
		if el.R != nil {
			r = *el.R
		}
		maxX = x + r
		maxY = y + r
	case "ellipse":
		rx, ry := 30.0, 20.0
		if el.RX != nil {
			rx = *el.RX
		}
		if el.RY != nil {
			ry = *el.RY
		}
		maxX = x + rx
		maxY = y + ry
	case "line":
		maxX = x
		maxY = y
		if el.X2 != nil {
			maxX = math.Max(maxX, *el.X2)
		}
		if el.Y2 != nil {
			maxY = math.Max(maxY, *el.Y2)
		}
	case "text":
		// Rough estimate: 8px per character width, fontSize height
		fontSize := 14.0
		if el.FontSize != nil {
			fontSize = *el.FontSize
		}
		maxX = x + float64(len(el.Text))*fontSize*0.6
		maxY = y + fontSize
	case "group":
		for _, child := range el.Children {
			cx, cy := elementBounds(child)
			maxX = math.Max(maxX, x+cx)
			maxY = math.Max(maxY, y+cy)
		}
	case "polygon", "polyline":
		for _, p := range el.Points {
			maxX = math.Max(maxX, p[0])
			maxY = math.Max(maxY, p[1])
		}
	}

	return
}

// ff formats a float with minimal precision (strips trailing zeros).
func ff(v float64) string {
	if v == math.Floor(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func pointsStr(pts [][2]float64) string {
	parts := make([]string, len(pts))
	for i, p := range pts {
		parts[i] = fmt.Sprintf("%s,%s", ff(p[0]), ff(p[1]))
	}
	return strings.Join(parts, " ")
}

func escAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
