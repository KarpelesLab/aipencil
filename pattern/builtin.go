package pattern

import (
	"embed"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/KarpelesLab/aipencil/scene"
)

//go:embed builtins/default/*.json builtins/comic/*.json builtins/manga/*.json builtins/stickman/*.json builtins/cute/*.json
var builtinFS embed.FS

// Available art styles (directories under builtins/)
var AvailableStyles = []string{"default", "comic", "manga", "stickman", "cute"}

func (r *Registry) loadBuiltins() {
	r.loadStyleDir("default")
}

// SetStyle loads patterns from the specified style directory,
// overlaying on top of the default patterns. Patterns not defined
// in the requested style fall back to default.
func (r *Registry) SetStyle(style string) {
	if style == "" || style == "default" {
		return
	}
	r.loadStyleDir(style)
}

func (r *Registry) loadStyleDir(style string) {
	dir := filepath.Join("builtins", style)
	entries, err := builtinFS.ReadDir(dir)
	if err != nil {
		// Style directory may not exist yet — that's OK for optional styles
		if style != "default" {
			log.Printf("warning: art style %q not found, using default", style)
		}
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := builtinFS.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			log.Printf("warning: could not read pattern %s/%s: %v", style, entry.Name(), err)
			continue
		}

		var def scene.Def
		if err := json.Unmarshal(data, &def); err != nil {
			log.Printf("warning: could not parse pattern %s/%s: %v", style, entry.Name(), err)
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		name = strings.ReplaceAll(name, "_", "-")
		r.patterns[name] = &def
	}
}
