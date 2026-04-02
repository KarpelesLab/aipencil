package render

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/KarpelesLab/aipencil/scene"
)

// RenderPNG renders a scene to PNG bytes.
// scale controls the output resolution multiplier (1.0 = native, 2.0 = 2x).
func RenderPNG(s *scene.Scene, scale float64) ([]byte, error) {
	svg := RenderSVG(s)
	return SVGToPNG([]byte(svg), scale)
}

// SVGToPNG converts SVG bytes to PNG using the best available rasterizer.
func SVGToPNG(svg []byte, scale float64) ([]byte, error) {
	if scale <= 0 {
		scale = 1.0
	}

	// Try rsvg-convert first (best quality)
	if path, err := exec.LookPath("rsvg-convert"); err == nil {
		return rsvgConvert(path, svg, scale)
	}

	// Try inkscape
	if path, err := exec.LookPath("inkscape"); err == nil {
		return inkscapeConvert(path, svg, scale)
	}

	// Try ImageMagick
	if path, err := exec.LookPath("magick"); err == nil {
		return magickConvert(path, svg, scale)
	}

	return nil, fmt.Errorf("no SVG rasterizer found; install rsvg-convert (librsvg), inkscape, or imagemagick")
}

func rsvgConvert(path string, svg []byte, scale float64) ([]byte, error) {
	args := []string{
		"--format=png",
		"--dpi-x=" + strconv.Itoa(int(96*scale)),
		"--dpi-y=" + strconv.Itoa(int(96*scale)),
	}

	cmd := exec.Command(path, args...)
	cmd.Stdin = bytes.NewReader(svg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rsvg-convert: %w: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func inkscapeConvert(path string, svg []byte, scale float64) ([]byte, error) {
	dpi := strconv.Itoa(int(96 * scale))
	cmd := exec.Command(path, "--export-type=png", "--export-dpi="+dpi, "--pipe")
	cmd.Stdin = bytes.NewReader(svg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("inkscape: %w: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func magickConvert(path string, svg []byte, scale float64) ([]byte, error) {
	density := strconv.Itoa(int(96 * scale))
	cmd := exec.Command(path, "convert", "-density", density, "svg:-", "png:-")
	cmd.Stdin = bytes.NewReader(svg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("imagemagick: %w: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
