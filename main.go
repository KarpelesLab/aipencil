package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/KarpelesLab/aipencil/layout"
	mcpserver "github.com/KarpelesLab/aipencil/mcp"
	"github.com/KarpelesLab/aipencil/pattern"
	"github.com/KarpelesLab/aipencil/render"
	"github.com/KarpelesLab/aipencil/scene"
)

func main() {
	var (
		output       = flag.String("o", "", "Output file (default: stdout)")
		format       = flag.String("format", "", "Output format: svg, png (default: inferred from -o extension, or svg)")
		scale        = flag.Float64("scale", 1.0, "PNG scale factor (e.g. 2.0 for 2x resolution)")
		validate     = flag.Bool("validate", false, "Validate input without rendering")
		listPatterns = flag.Bool("list-patterns", false, "List available built-in patterns")
		mcpMode      = flag.Bool("mcp", false, "Run as MCP server (stdio transport)")
	)
	flag.Parse()

	if *mcpMode {
		if err := mcpserver.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	registry := pattern.NewRegistry()

	if *listPatterns {
		patterns := registry.List()
		if len(patterns) == 0 {
			fmt.Println("No built-in patterns available.")
			return
		}
		for name, def := range patterns {
			fmt.Printf("  %s (%gx%g)\n", name, def.Width, def.Height)
			for pname, pdef := range def.Params {
				defVal := ""
				if pdef.Default != nil {
					defVal = fmt.Sprintf(" (default: %v)", pdef.Default)
				}
				enumStr := ""
				if len(pdef.Enum) > 0 {
					enumStr = fmt.Sprintf(" [%s]", strings.Join(pdef.Enum, ", "))
				}
				fmt.Printf("    - %s: %s%s%s\n", pname, pdef.Type, defVal, enumStr)
			}
		}
		return
	}

	// Read input from file argument or stdin
	var input io.Reader
	if args := flag.Args(); len(args) > 0 {
		f, err := os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		input = f
	} else {
		input = os.Stdin
	}

	s, err := scene.Parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Validate
	errs := scene.Validate(s)
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "validation errors:\n  %s\n", strings.Join(errs, "\n  "))
		os.Exit(1)
	}
	if *validate {
		fmt.Println("valid")
		return
	}

	// Expand patterns
	if err := registry.Expand(s); err != nil {
		fmt.Fprintf(os.Stderr, "pattern error: %v\n", err)
		os.Exit(1)
	}

	// Layout
	layout.Layout(s)

	// Infer format from output extension if not specified
	outFormat := *format
	if outFormat == "" {
		if strings.HasSuffix(*output, ".png") {
			outFormat = "png"
		} else {
			outFormat = "svg"
		}
	}

	// Render
	switch outFormat {
	case "svg":
		svg := render.RenderSVG(s)
		if *output != "" {
			if err := os.WriteFile(*output, []byte(svg), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Print(svg)
		}
	case "png":
		pngData, err := render.RenderPNG(s, *scale)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error rendering PNG: %v\n", err)
			os.Exit(1)
		}
		if *output != "" {
			if err := os.WriteFile(*output, pngData, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
				os.Exit(1)
			}
		} else {
			os.Stdout.Write(pngData)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", outFormat)
		os.Exit(1)
	}
}
