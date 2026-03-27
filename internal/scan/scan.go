package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Result holds the structured output of a local repository scan.
type Result struct {
	RootPath   string
	Language   string   // go, node, python, rust, java, unknown
	Framework  string   // detected framework (e.g. gin, echo, net/http, express, django)
	BuildFiles []string // go.mod, package.json, Cargo.toml, etc.
	Structure  string   // tree-like representation of top-level dirs
	Samples    []Sample // representative code samples
	Configs    []string // config file contents (Dockerfile, CI, .env.example, etc.)
	TestInfo   string   // testing patterns detected
}

// Sample is a code snippet from a representative file.
type Sample struct {
	Path    string
	Content string
}

const (
	maxSampleBytes   = 8_000
	maxSamplesTotal  = 20
	maxConfigBytes   = 4_000
	maxConfigsTotal  = 10
	maxTreeDepth     = 3
)

// ignoredDirs are directories skipped during scanning.
var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".mole": true,
	"dist": true, "build": true, "__pycache__": true, ".next": true,
	".helm": true, "target": true, ".idea": true, ".vscode": true,
}

// Run performs a local scan of the repository at the given path.
func Run(root string) (*Result, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("path %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", root)
	}

	r := &Result{RootPath: root}

	r.BuildFiles = detectBuildFiles(root)
	r.Language, r.Framework = detectStack(root, r.BuildFiles)
	r.Structure = buildTree(root, "", 0)
	r.Samples = collectSamples(root, r.Language)
	r.Configs = collectConfigs(root)
	r.TestInfo = detectTests(root, r.Language)

	return r, nil
}

// Format returns a human-readable report suitable for LLM consumption.
func (r *Result) Format() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Repository Scan: %s\n\n", filepath.Base(r.RootPath))

	fmt.Fprintf(&b, "## Stack\n- Language: %s\n- Framework: %s\n", r.Language, r.Framework)
	if len(r.BuildFiles) > 0 {
		fmt.Fprintf(&b, "- Build files: %s\n", strings.Join(r.BuildFiles, ", "))
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## Directory Structure\n```\n%s```\n\n", r.Structure)

	if r.TestInfo != "" {
		fmt.Fprintf(&b, "## Testing\n%s\n\n", r.TestInfo)
	}

	if len(r.Configs) > 0 {
		b.WriteString("## Configuration Files\n\n")
		for _, c := range r.Configs {
			b.WriteString(c)
			b.WriteString("\n")
		}
	}

	if len(r.Samples) > 0 {
		b.WriteString("## Code Samples\n\n")
		for _, s := range r.Samples {
			ext := filepath.Ext(s.Path)
			lang := extToLang(ext)
			fmt.Fprintf(&b, "### %s\n```%s\n%s\n```\n\n", s.Path, lang, s.Content)
		}
	}

	return b.String()
}

func detectBuildFiles(root string) []string {
	candidates := []string{
		"go.mod", "go.sum", "package.json", "package-lock.json", "yarn.lock",
		"Cargo.toml", "pyproject.toml", "requirements.txt", "setup.py",
		"pom.xml", "build.gradle", "Gemfile", "mix.exs", "Makefile",
	}
	var found []string
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(root, c)); err == nil {
			found = append(found, c)
		}
	}
	return found
}

func detectStack(root string, buildFiles []string) (language, framework string) {
	for _, bf := range buildFiles {
		switch bf {
		case "go.mod":
			language = "go"
			framework = detectGoFramework(root)
			return
		case "package.json":
			language = "node"
			framework = detectNodeFramework(root)
			return
		case "Cargo.toml":
			return "rust", ""
		case "pyproject.toml", "requirements.txt", "setup.py":
			language = "python"
			framework = detectPythonFramework(root)
			return
		case "pom.xml", "build.gradle":
			return "java", ""
		case "Gemfile":
			return "ruby", ""
		case "mix.exs":
			return "elixir", ""
		}
	}
	return "unknown", ""
}

func detectGoFramework(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	content := string(data)
	switch {
	case strings.Contains(content, "github.com/gin-gonic/gin"):
		return "gin"
	case strings.Contains(content, "github.com/labstack/echo"):
		return "echo"
	case strings.Contains(content, "github.com/gofiber/fiber"):
		return "fiber"
	case strings.Contains(content, "github.com/gorilla/mux"):
		return "gorilla/mux"
	case strings.Contains(content, "github.com/go-chi/chi"):
		return "chi"
	default:
		return "net/http"
	}
}

func detectNodeFramework(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return ""
	}
	content := string(data)
	switch {
	case strings.Contains(content, "\"next\""):
		return "next"
	case strings.Contains(content, "\"express\""):
		return "express"
	case strings.Contains(content, "\"fastify\""):
		return "fastify"
	case strings.Contains(content, "\"nestjs\"") || strings.Contains(content, "\"@nestjs/core\""):
		return "nestjs"
	default:
		return ""
	}
}

func detectPythonFramework(root string) string {
	for _, f := range []string{"pyproject.toml", "requirements.txt"} {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		content := string(data)
		switch {
		case strings.Contains(content, "django"):
			return "django"
		case strings.Contains(content, "fastapi"):
			return "fastapi"
		case strings.Contains(content, "flask"):
			return "flask"
		}
	}
	return ""
}

func buildTree(dir, prefix string, depth int) string {
	if depth > maxTreeDepth {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var b strings.Builder
	// Filter relevant entries
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
			b.WriteString(buildTree(filepath.Join(dir, name), childPrefix, depth+1))
		} else if depth < 2 { // only show files at top 2 levels
			fmt.Fprintf(&b, "%s%s%s\n", prefix, connector, name)
		}
	}

	return b.String()
}

func collectSamples(root, language string) []Sample {
	var exts []string
	switch language {
	case "go":
		exts = []string{".go"}
	case "node":
		exts = []string{".ts", ".js", ".tsx", ".jsx"}
	case "python":
		exts = []string{".py"}
	case "rust":
		exts = []string{".rs"}
	case "java":
		exts = []string{".java"}
	case "ruby":
		exts = []string{".rb"}
	case "elixir":
		exts = []string{".ex", ".exs"}
	default:
		exts = []string{".go", ".ts", ".js", ".py", ".rs", ".java"}
	}

	var samples []Sample
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if len(samples) >= maxSamplesTotal {
			return filepath.SkipAll
		}

		ext := filepath.Ext(path)
		if !matchExt(ext, exts) {
			return nil
		}
		// Skip test files for samples
		name := d.Name()
		if strings.Contains(name, "_test.") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		if len(content) > maxSampleBytes {
			content = content[:maxSampleBytes] + "\n// ... truncated"
		}

		rel, _ := filepath.Rel(root, path)
		samples = append(samples, Sample{Path: rel, Content: content})
		return nil
	})

	return samples
}

func collectConfigs(root string) []string {
	candidates := []string{
		"Dockerfile", "docker-compose.yml", "docker-compose.yaml",
		".github/workflows", ".env.example", ".env.sample",
		"Makefile", ".goreleaser.yml", ".goreleaser.yaml",
	}

	var configs []string
	for _, c := range candidates {
		if len(configs) >= maxConfigsTotal {
			break
		}
		path := filepath.Join(root, c)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			// Read files inside directory (e.g. .github/workflows/)
			entries, err := os.ReadDir(path)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() || len(configs) >= maxConfigsTotal {
					continue
				}
				data, err := os.ReadFile(filepath.Join(path, e.Name()))
				if err != nil {
					continue
				}
				content := string(data)
				if len(content) > maxConfigBytes {
					content = content[:maxConfigBytes] + "\n# ... truncated"
				}
				rel := filepath.Join(c, e.Name())
				configs = append(configs, fmt.Sprintf("### %s\n```yaml\n%s\n```\n", rel, content))
			}
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			content := string(data)
			if len(content) > maxConfigBytes {
				content = content[:maxConfigBytes] + "\n# ... truncated"
			}
			configs = append(configs, fmt.Sprintf("### %s\n```\n%s\n```\n", c, content))
		}
	}

	return configs
}

func detectTests(root, language string) string {
	var b strings.Builder
	var testFiles int
	var testDirs []string

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			if d.Name() == "test" || d.Name() == "tests" || d.Name() == "__tests__" {
				rel, _ := filepath.Rel(root, path)
				testDirs = append(testDirs, rel)
			}
			return nil
		}

		name := d.Name()
		if strings.Contains(name, "_test.") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
			testFiles++
		}
		return nil
	})

	if testFiles > 0 {
		fmt.Fprintf(&b, "- Test files found: %d\n", testFiles)
	}
	if len(testDirs) > 0 {
		fmt.Fprintf(&b, "- Test directories: %s\n", strings.Join(testDirs, ", "))
	}

	switch language {
	case "go":
		b.WriteString("- Framework: Go standard testing\n")
	case "node":
		if _, err := os.Stat(filepath.Join(root, "jest.config.js")); err == nil {
			b.WriteString("- Framework: Jest\n")
		} else if _, err := os.Stat(filepath.Join(root, "vitest.config.ts")); err == nil {
			b.WriteString("- Framework: Vitest\n")
		}
	case "python":
		if _, err := os.Stat(filepath.Join(root, "pytest.ini")); err == nil {
			b.WriteString("- Framework: pytest\n")
		} else if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
			b.WriteString("- Framework: pytest (assumed)\n")
		}
	}

	return b.String()
}

func matchExt(ext string, exts []string) bool {
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

func extToLang(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".ex", ".exs":
		return "elixir"
	default:
		return ""
	}
}
