package font

import (
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/math/fixed"
)

var (
	regularFont *opentype.Font
	boldFont    *opentype.Font
	monoFont    *opentype.Font
	initOnce    sync.Once
	initErr     error

	faceCache   = make(map[faceKey]font.Face)
	faceCacheMu sync.Mutex
)

type faceKey struct {
	style string // "regular", "bold", "mono"
	size  float64
}

func initFonts() {
	regularFont, initErr = opentype.Parse(goregular.TTF)
	if initErr != nil {
		return
	}
	boldFont, initErr = opentype.Parse(gobold.TTF)
	if initErr != nil {
		return
	}
	monoFont, initErr = opentype.Parse(gomono.TTF)
}

func getFace(style string, size float64) (font.Face, error) {
	initOnce.Do(initFonts)
	if initErr != nil {
		return nil, initErr
	}

	key := faceKey{style, size}
	faceCacheMu.Lock()
	defer faceCacheMu.Unlock()

	if f, ok := faceCache[key]; ok {
		return f, nil
	}

	var ft *opentype.Font
	switch style {
	case "bold":
		ft = boldFont
	case "mono", "monospace":
		ft = monoFont
	default:
		ft = regularFont
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	faceCache[key] = face
	return face, nil
}

// MeasureText returns the pixel width and height of a single line of text.
func MeasureText(text string, fontSize float64, fontWeight string) (width, height float64) {
	style := "regular"
	if fontWeight == "bold" {
		style = "bold"
	}

	face, err := getFace(style, fontSize)
	if err != nil {
		// Fallback: rough estimate
		return float64(len(text)) * fontSize * 0.6, fontSize * 1.2
	}

	advance := font.MeasureString(face, text)
	width = fixedToFloat(advance)
	height = fontSize * 1.2 // Use font size as line height base

	return
}

// MeasureLines measures multi-line text and returns total width and height.
func MeasureLines(lines []string, fontSize float64, fontWeight string, lineSpacing float64) (width, height float64) {
	if lineSpacing == 0 {
		lineSpacing = 1.4
	}
	for _, line := range lines {
		w, _ := MeasureText(line, fontSize, fontWeight)
		if w > width {
			width = w
		}
	}
	height = float64(len(lines)) * fontSize * lineSpacing
	return
}

// WrapText wraps text to fit within maxWidth, returning lines.
func WrapText(text string, maxWidth float64, fontSize float64, fontWeight string) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	words := splitWords(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		candidate := currentLine
		if candidate != "" {
			candidate += " "
		}
		candidate += word

		w, _ := MeasureText(candidate, fontSize, fontWeight)
		if w > maxWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = candidate
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

func splitWords(s string) []string {
	var words []string
	current := ""
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

func fixedToFloat(v fixed.Int26_6) float64 {
	return float64(v) / 64.0
}
