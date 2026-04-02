package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KarpelesLab/aipencil/layout"
	"github.com/KarpelesLab/aipencil/pattern"
	"github.com/KarpelesLab/aipencil/render"
	"github.com/KarpelesLab/aipencil/scene"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RenderInput struct {
	Scene  json.RawMessage `json:"scene" jsonschema:"The scene description JSON object"`
	Format string          `json:"format,omitempty" jsonschema:"Output format: svg or png (default: svg)"`
	Scale  float64         `json:"scale,omitempty" jsonschema:"Scale factor for PNG output (default: 1.0)"`
}

type ValidateInput struct {
	Scene json.RawMessage `json:"scene" jsonschema:"The scene description JSON object to validate"`
}

type ListPatternsInput struct{}

// Run starts the MCP server on stdio.
func Run() error {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "aipencil",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			Instructions: "aipencil renders structured JSON scene descriptions to SVG or PNG graphics. Use 'render' to generate images, 'validate' to check input, and 'list_patterns' to see available reusable patterns.",
		},
	)

	registry := pattern.NewRegistry()

	mcp.AddTool(server, &mcp.Tool{
		Name:        "render",
		Description: "Render a scene description to SVG or PNG. The scene is a JSON object describing shapes, text, groups with layouts, arrows between elements, and pattern instances.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RenderInput) (*mcp.CallToolResult, any, error) {
		return handleRender(input, registry)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate",
		Description: "Validate a scene description JSON without rendering. Returns validation errors or 'valid'.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ValidateInput) (*mcp.CallToolResult, any, error) {
		return handleValidate(input, registry)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_patterns",
		Description: "List all available built-in patterns with their parameters. Patterns can be instantiated in scenes using {\"type\": \"use\", \"pattern\": \"name\", \"params\": {...}}.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListPatternsInput) (*mcp.CallToolResult, any, error) {
		return handleListPatterns(registry)
	})

	return server.Run(context.Background(), &mcp.StdioTransport{})
}

func handleRender(input RenderInput, registry *pattern.Registry) (*mcp.CallToolResult, any, error) {
	s, err := scene.ParseBytes(input.Scene)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Parse error: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	errs := scene.Validate(s)
	if len(errs) > 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Validation errors:\n" + strings.Join(errs, "\n")}},
			IsError: true,
		}, nil, nil
	}

	if err := registry.Expand(s); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Pattern error: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	layout.Layout(s)

	format := input.Format
	if format == "" {
		format = "svg"
	}

	switch format {
	case "svg":
		svg := render.RenderSVG(s)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: svg}},
		}, nil, nil

	case "png":
		scale := input.Scale
		if scale <= 0 {
			scale = 1.0
		}
		pngData, err := render.RenderPNG(s, scale)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("PNG render error: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.ImageContent{
				Data:     pngData,
				MIMEType: "image/png",
			}},
		}, nil, nil

	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Unknown format: %s", format)}},
			IsError: true,
		}, nil, nil
	}
}

func handleValidate(input ValidateInput, registry *pattern.Registry) (*mcp.CallToolResult, any, error) {
	s, err := scene.ParseBytes(input.Scene)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Parse error: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	errs := scene.Validate(s)
	if len(errs) > 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Validation errors:\n" + strings.Join(errs, "\n")}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "valid"}},
	}, nil, nil
}

func handleListPatterns(registry *pattern.Registry) (*mcp.CallToolResult, any, error) {
	patterns := registry.List()

	var sb strings.Builder
	sb.WriteString("Available patterns:\n\n")

	for name, def := range patterns {
		fmt.Fprintf(&sb, "## %s (%gx%g)\n", name, def.Width, def.Height)
		if len(def.Params) > 0 {
			sb.WriteString("Parameters:\n")
			for pname, pdef := range def.Params {
				fmt.Fprintf(&sb, "  - %s (%s)", pname, pdef.Type)
				if pdef.Default != nil {
					fmt.Fprintf(&sb, " default: %v", pdef.Default)
				}
				if len(pdef.Enum) > 0 {
					fmt.Fprintf(&sb, " options: [%s]", strings.Join(pdef.Enum, ", "))
				}
				sb.WriteByte('\n')
			}
		}
		sb.WriteByte('\n')
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}
