package render

import (
	"fmt"
	"math"
	"strings"

	"github.com/KarpelesLab/aipencil/font"
	"github.com/KarpelesLab/aipencil/scene"
)

// renderBubble renders a speech/thought/shout bubble with optional tail.
func renderBubble(sb *strings.Builder, el *scene.Element, allElements []*scene.Element, s *scene.Scene, indent string) {
	bx, by := el.EffectiveX(), el.EffectiveY()
	bw := el.ComputedWidth
	bh := el.ComputedHeight

	style := resolveStyle(el, s)
	fill := "#ffffff"
	stroke := "#000000"
	strokeWidth := 2.0
	if style != nil {
		if style.Fill != "" {
			fill = style.Fill
		}
		if style.Stroke != "" {
			stroke = style.Stroke
		}
		if style.StrokeWidth != nil {
			strokeWidth = *style.StrokeWidth
		}
	}

	// Compute tail target point
	var tailX, tailY float64
	hasTail := false
	if el.Target != "" {
		ids := collectIDs(allElements)
		if target, ok := ids[el.Target]; ok {
			tailX, tailY = elementCenter(target, allElements)
			hasTail = true
		}
	}

	switch el.BubbleStyle {
	case "thought":
		renderThoughtBubble(sb, bx, by, bw, bh, tailX, tailY, hasTail, fill, stroke, strokeWidth, indent)
	case "shout":
		renderShoutBubble(sb, bx, by, bw, bh, tailX, tailY, hasTail, fill, stroke, strokeWidth, indent)
	default: // "speech"
		renderSpeechBubble(sb, bx, by, bw, bh, tailX, tailY, hasTail, fill, stroke, strokeWidth, indent)
	}

	// Render text inside the bubble
	fontSize := 14.0
	if el.FontSize != nil {
		fontSize = *el.FontSize
	} else if style != nil && style.FontSize != nil {
		fontSize = *style.FontSize
	}
	fontWeight := el.FontWeight
	if fontWeight == "" && style != nil {
		fontWeight = style.FontWeight
	}

	padding := 16.0
	maxTextW := bw - padding*2

	// Split on explicit newlines first, then wrap each line
	var lines []string
	for _, seg := range strings.Split(el.Text, "\n") {
		lines = append(lines, font.WrapText(seg, maxTextW, fontSize, fontWeight)...)
	}

	_, totalH := font.MeasureLines(lines, fontSize, fontWeight, 1.4)
	textX := bx + bw/2
	textY := by + (bh-totalH)/2 + fontSize*0.6

	for i, line := range lines {
		sb.WriteString(indent)
		dy := ""
		if i > 0 {
			dy = fmt.Sprintf(` dy="%s"`, ff(fontSize*1.4))
		}
		if i == 0 {
			fmt.Fprintf(sb, `<text x="%s" y="%s" font-size="%s" text-anchor="middle" dominant-baseline="auto"`,
				ff(textX), ff(textY), ff(fontSize))
			if fontWeight != "" {
				fmt.Fprintf(sb, ` font-weight="%s"`, escAttr(fontWeight))
			}
			sb.WriteByte('>')
			if len(lines) == 1 {
				sb.WriteString(escText(line))
				sb.WriteString("</text>\n")
			} else {
				sb.WriteByte('\n')
				fmt.Fprintf(sb, `%s  <tspan x="%s"%s>%s</tspan>`+"\n", indent, ff(textX), dy, escText(line))
			}
		} else {
			fmt.Fprintf(sb, `  <tspan x="%s"%s>%s</tspan>`+"\n", ff(textX), dy, escText(line))
			if i == len(lines)-1 {
				sb.WriteString(indent)
				sb.WriteString("</text>\n")
			}
		}
	}
}

func renderSpeechBubble(sb *strings.Builder, bx, by, bw, bh, tailX, tailY float64, hasTail bool, fill, stroke string, strokeWidth float64, indent string) {
	rx := math.Min(12, bw/4)

	// Bubble body
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s" stroke="%s" stroke-width="%s"/>`,
		ff(bx), ff(by), ff(bw), ff(bh), ff(rx), escAttr(fill), escAttr(stroke), ff(strokeWidth))
	sb.WriteByte('\n')

	// Tail (triangle pointing at target)
	if hasTail {
		cx, cy := bx+bw/2, by+bh/2
		dx, dy := tailX-cx, tailY-cy
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1 {
			return
		}

		// Tail base on the edge of the bubble
		edgeX, edgeY := clipToEdge(cx, cy, bw/2, bh/2, tailX, tailY)

		// Perpendicular direction for tail width
		nx, ny := -dy/dist, dx/dist
		tailWidth := 8.0

		// Tail base points (slightly inside the bubble to overlap the border)
		b1x := edgeX + nx*tailWidth - dx/dist*2
		b1y := edgeY + ny*tailWidth - dy/dist*2
		b2x := edgeX - nx*tailWidth - dx/dist*2
		b2y := edgeY - ny*tailWidth - dy/dist*2

		// Tail tip (partway toward target, not all the way)
		tipDist := math.Min(30, dist*0.4)
		tipX := edgeX + dx/dist*tipDist
		tipY := edgeY + dy/dist*tipDist

		// Draw tail with fill to cover the bubble border underneath
		sb.WriteString(indent)
		fmt.Fprintf(sb, `<polygon points="%s,%s %s,%s %s,%s" fill="%s" stroke="none"/>`,
			ff(b1x), ff(b1y), ff(tipX), ff(tipY), ff(b2x), ff(b2y), escAttr(fill))
		sb.WriteByte('\n')

		// Draw tail outline (only the outer edges, not the base)
		sb.WriteString(indent)
		fmt.Fprintf(sb, `<polyline points="%s,%s %s,%s %s,%s" fill="none" stroke="%s" stroke-width="%s" stroke-linejoin="round"/>`,
			ff(b1x), ff(b1y), ff(tipX), ff(tipY), ff(b2x), ff(b2y), escAttr(stroke), ff(strokeWidth))
		sb.WriteByte('\n')
	}
}

func renderThoughtBubble(sb *strings.Builder, bx, by, bw, bh, tailX, tailY float64, hasTail bool, fill, stroke string, strokeWidth float64, indent string) {
	// Cloud-shaped body using overlapping ellipses
	cx, cy := bx+bw/2, by+bh/2

	// Main ellipse
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s" fill="%s" stroke="%s" stroke-width="%s"/>`,
		ff(cx), ff(cy), ff(bw/2), ff(bh/2), escAttr(fill), escAttr(stroke), ff(strokeWidth))
	sb.WriteByte('\n')

	// Cloud bumps
	bumps := [][2]float64{
		{cx - bw*0.35, cy - bh*0.3},
		{cx, cy - bh*0.42},
		{cx + bw*0.35, cy - bh*0.3},
		{cx + bw*0.42, cy},
		{cx + bw*0.35, cy + bh*0.3},
		{cx, cy + bh*0.42},
		{cx - bw*0.35, cy + bh*0.3},
		{cx - bw*0.42, cy},
	}
	bumpR := math.Min(bw, bh) * 0.18
	for _, b := range bumps {
		sb.WriteString(indent)
		fmt.Fprintf(sb, `<circle cx="%s" cy="%s" r="%s" fill="%s" stroke="%s" stroke-width="%s"/>`,
			ff(b[0]), ff(b[1]), ff(bumpR), escAttr(fill), escAttr(stroke), ff(strokeWidth))
		sb.WriteByte('\n')
	}
	// Fill center to cover inner bump borders
	sb.WriteString(indent)
	fmt.Fprintf(sb, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s" fill="%s" stroke="none"/>`,
		ff(cx), ff(cy), ff(bw/2-2), ff(bh/2-2), escAttr(fill))
	sb.WriteByte('\n')

	// Chain of circles leading to target
	if hasTail {
		edgeX, edgeY := clipToEdge(cx, cy, bw/2, bh/2, tailX, tailY)
		dx, dy := tailX-edgeX, tailY-edgeY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 5 {
			for i := range 3 {
				t := float64(i+1) / 4.0
				r := 6.0 - float64(i)*1.5
				px := edgeX + dx*t
				py := edgeY + dy*t
				sb.WriteString(indent)
				fmt.Fprintf(sb, `<circle cx="%s" cy="%s" r="%s" fill="%s" stroke="%s" stroke-width="%s"/>`,
					ff(px), ff(py), ff(r), escAttr(fill), escAttr(stroke), ff(strokeWidth*0.7))
				sb.WriteByte('\n')
			}
		}
	}
}

func renderShoutBubble(sb *strings.Builder, bx, by, bw, bh, tailX, tailY float64, hasTail bool, fill, stroke string, strokeWidth float64, indent string) {
	cx, cy := bx+bw/2, by+bh/2
	spikes := 12
	outerRX, outerRY := bw/2+8, bh/2+8
	innerRX, innerRY := bw/2-4, bh/2-4

	// Build spiked polygon
	var points []string
	for i := range spikes * 2 {
		angle := 2 * math.Pi * float64(i) / float64(spikes*2)
		var rx, ry float64
		if i%2 == 0 {
			rx, ry = outerRX, outerRY
		} else {
			rx, ry = innerRX, innerRY
		}
		px := cx + rx*math.Cos(angle)
		py := cy + ry*math.Sin(angle)
		points = append(points, fmt.Sprintf("%s,%s", ff(px), ff(py)))
	}

	sb.WriteString(indent)
	fmt.Fprintf(sb, `<polygon points="%s" fill="%s" stroke="%s" stroke-width="%s" stroke-linejoin="round"/>`,
		strings.Join(points, " "), escAttr(fill), escAttr(stroke), ff(strokeWidth))
	sb.WriteByte('\n')

	// Tail for shout bubble
	if hasTail {
		edgeX, edgeY := clipToEdge(cx, cy, bw/2, bh/2, tailX, tailY)
		dx, dy := tailX-edgeX, tailY-edgeY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 5 {
			nx, ny := -dy/dist, dx/dist
			tipDist := math.Min(25, dist*0.4)
			tipX := edgeX + dx/dist*tipDist
			tipY := edgeY + dy/dist*tipDist

			sb.WriteString(indent)
			fmt.Fprintf(sb, `<polygon points="%s,%s %s,%s %s,%s" fill="%s" stroke="%s" stroke-width="%s" stroke-linejoin="round"/>`,
				ff(edgeX+nx*6), ff(edgeY+ny*6),
				ff(tipX), ff(tipY),
				ff(edgeX-nx*6), ff(edgeY-ny*6),
				escAttr(fill), escAttr(stroke), ff(strokeWidth))
			sb.WriteByte('\n')
		}
	}
}

// hasBubbles checks if the scene contains any bubble elements.
func hasBubbles(elements []*scene.Element) bool {
	for _, el := range elements {
		if el.Type == "bubble" {
			return true
		}
		if hasBubbles(el.Children) {
			return true
		}
	}
	return false
}

// renderBubbles renders all bubble elements in the scene.
func renderBubbles(sb *strings.Builder, elements []*scene.Element, allElements []*scene.Element, s *scene.Scene, indent string) {
	for _, el := range elements {
		if el.Type == "bubble" {
			renderBubble(sb, el, allElements, s, indent)
		}
		if len(el.Children) > 0 {
			renderBubbles(sb, el.Children, allElements, s, indent)
		}
	}
}
