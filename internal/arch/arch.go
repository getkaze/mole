package arch

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	ghclient "github.com/getkaze/mole/internal/github"
	"github.com/getkaze/mole/internal/llm"
)

// Validate checks Go source files for architecture layer violations.
// It parses imports from each file and checks them against the rules
// defined in the repo config.
func Validate(repoPath string, rules *ghclient.ArchitectureRule) []llm.InlineComment {
	if rules == nil || len(rules.Layers) == 0 {
		return nil
	}

	// Build layer lookup: package path pattern → Layer
	layers := buildLayerIndex(rules.Layers)

	var comments []llm.InlineComment

	filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, _ := filepath.Rel(repoPath, path)
		violations := checkFile(path, rel, layers)
		comments = append(comments, violations...)
		return nil
	})

	return comments
}

type layerInfo struct {
	name      string
	canImport map[string]bool
}

func buildLayerIndex(layers []ghclient.Layer) map[string]layerInfo {
	index := make(map[string]layerInfo)
	for _, l := range layers {
		allowed := make(map[string]bool)
		for _, imp := range l.CanImport {
			allowed[imp] = true
		}
		index[l.Path] = layerInfo{
			name:      l.Name,
			canImport: allowed,
		}
	}
	return index
}

// findLayer returns the layer a file belongs to, based on glob matching.
func findLayer(relPath string, layers map[string]layerInfo) (layerInfo, bool) {
	for pattern, info := range layers {
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return info, true
		}
		// Try matching the directory path
		dir := filepath.Dir(relPath)
		dirPattern := strings.TrimSuffix(pattern, "/**")
		dirPattern = strings.TrimSuffix(dirPattern, "/*")
		if strings.HasPrefix(dir, dirPattern) {
			return info, true
		}
	}
	return layerInfo{}, false
}

// findLayerByName returns the layer name for an import path segment.
func findLayerByName(importPath string, layers map[string]layerInfo) string {
	for _, info := range layers {
		if strings.Contains(importPath, info.name) {
			return info.name
		}
	}
	return ""
}

func checkFile(absPath, relPath string, layers map[string]layerInfo) []llm.InlineComment {
	srcLayer, ok := findLayer(relPath, layers)
	if !ok {
		return nil // file not in any defined layer
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, parser.ImportsOnly)
	if err != nil {
		return nil
	}

	var violations []llm.InlineComment
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		targetLayer := findLayerByName(importPath, layers)
		if targetLayer == "" || targetLayer == srcLayer.name {
			continue // not a known layer or same layer
		}
		if !srcLayer.canImport[targetLayer] {
			pos := fset.Position(imp.Pos())
			violations = append(violations, llm.InlineComment{
				File:        relPath,
				Line:        pos.Line,
				Category:    "Architecture",
				Subcategory: "Layer Violation",
				Severity:    "attention",
				Message: fmt.Sprintf(
					"Layer %q should not import from layer %q. Allowed imports: %s.",
					srcLayer.name, targetLayer, formatAllowed(srcLayer.canImport),
				),
			})
		}
	}

	return violations
}

func formatAllowed(allowed map[string]bool) string {
	if len(allowed) == 0 {
		return "none"
	}
	var names []string
	for k := range allowed {
		names = append(names, k)
	}
	return strings.Join(names, ", ")
}
