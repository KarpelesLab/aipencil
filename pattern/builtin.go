package pattern

import (
	"embed"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/KarpelesLab/aipencil/scene"
)

//go:embed builtins/*.json
var builtinFS embed.FS

func (r *Registry) loadBuiltins() {
	entries, err := builtinFS.ReadDir("builtins")
	if err != nil {
		log.Printf("warning: could not read built-in patterns: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := builtinFS.ReadFile(filepath.Join("builtins", entry.Name()))
		if err != nil {
			log.Printf("warning: could not read built-in pattern %s: %v", entry.Name(), err)
			continue
		}

		var def scene.Def
		if err := json.Unmarshal(data, &def); err != nil {
			log.Printf("warning: could not parse built-in pattern %s: %v", entry.Name(), err)
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		name = strings.ReplaceAll(name, "_", "-")
		r.patterns[name] = &def
	}
}
