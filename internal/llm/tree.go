package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ignoredDirs are directories skipped during tree generation.
var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".mole": true,
	"dist": true, "build": true, "__pycache__": true, ".next": true,
	".helm": true, "target": true, ".idea": true, ".vscode": true,
}

// BuildTree generates a text representation of the directory structure at root,
// suitable for inclusion in an LLM prompt. Skips common non-essential directories.
func BuildTree(root string, maxDepth int) string {
	var b strings.Builder
	b.WriteString(filepath.Base(root) + "/\n")
	b.WriteString(buildTree(root, "", 0, maxDepth))
	return b.String()
}

func buildTree(dir, prefix string, depth, maxDepth int) string {
	if depth >= maxDepth {
		return ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var visible []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".github" {
			continue
		}
		if e.IsDir() && ignoredDirs[name] {
			continue
		}
		visible = append(visible, e)
	}

	var b strings.Builder
	for i, e := range visible {
		isLast := i == len(visible)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		name := e.Name()
		if e.IsDir() {
			fmt.Fprintf(&b, "%s%s%s/\n", prefix, connector, name)
			b.WriteString(buildTree(filepath.Join(dir, name), childPrefix, depth+1, maxDepth))
		} else if depth < maxDepth-1 {
			fmt.Fprintf(&b, "%s%s%s\n", prefix, connector, name)
		}
	}

	return b.String()
}
