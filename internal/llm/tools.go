package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	maxFileSize       = 100 * 1024 // 100KB per file read
	maxSearchFileSize = 512 * 1024 // 512KB per file for search (skip larger files)
	maxSearchMatches  = 50
	maxRegexLen       = 500 // max length for user-provided regex patterns
)

// ToolExecutor executes exploration tools sandboxed within a worktree directory.
type ToolExecutor struct {
	root string // absolute path to the worktree root
}

// NewToolExecutor creates a ToolExecutor bound to the given worktree root.
func NewToolExecutor(worktreeRoot string) *ToolExecutor {
	abs, _ := filepath.Abs(worktreeRoot)
	return &ToolExecutor{root: abs}
}

// Execute runs a tool by name with the given JSON input.
// Returns the result string and whether it represents an error.
func (te *ToolExecutor) Execute(name string, input json.RawMessage) (string, bool) {
	switch name {
	case "get_file":
		return te.getFile(input)
	case "search_code":
		return te.searchCode(input)
	case "list_dir":
		return te.listDir(input)
	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

// ToolDefinitions returns the Anthropic tool definitions for the 3 exploration tools.
func ToolDefinitions() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{
			OfTool: &anthropic.ToolParam{
				Name:        "get_file",
				Description: anthropic.String("Read the contents of a file. Returns the full file content up to 100KB."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "File path relative to the repository root",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		{
			OfTool: &anthropic.ToolParam{
				Name:        "search_code",
				Description: anthropic.String("Search for a regex pattern across files in the repository. Returns up to 50 matches with file path, line number, and matching line."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Regex pattern to search for",
						},
						"file_pattern": map[string]any{
							"type":        "string",
							"description": "Optional glob pattern to filter files (e.g. '*.go', '*.ts')",
						},
					},
					Required: []string{"query"},
				},
			},
		},
		{
			OfTool: &anthropic.ToolParam{
				Name:        "list_dir",
				Description: anthropic.String("List files and directories at a given path. Directories are marked with a trailing slash."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type: "object",
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Directory path relative to the repository root. Use '.' for the root.",
						},
					},
					Required: []string{"path"},
				},
			},
		},
	}
}

// safePath resolves a relative path within the worktree root and validates
// it doesn't escape via traversal or symlinks.
func (te *ToolExecutor) safePath(relPath string) (string, error) {
	cleaned := filepath.Clean(relPath)
	abs := filepath.Join(te.root, cleaned)

	resolved, err := filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", relPath)
	}

	// Ensure the resolved path is within the worktree root
	if !strings.HasPrefix(resolved, te.root+string(filepath.Separator)) && resolved != te.root {
		return "", fmt.Errorf("path outside repository: %s", relPath)
	}

	// Check for symlink escape
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(resolved)
		if err != nil {
			return "", fmt.Errorf("cannot resolve symlink: %s", relPath)
		}
		if !strings.HasPrefix(target, te.root+string(filepath.Separator)) {
			return "", fmt.Errorf("symlink points outside repository: %s", relPath)
		}
	}

	return resolved, nil
}

type getFileInput struct {
	Path string `json:"path"`
}

func (te *ToolExecutor) getFile(input json.RawMessage) (string, bool) {
	var in getFileInput
	if err := json.Unmarshal(input, &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}

	abs, err := te.safePath(in.Path)
	if err != nil {
		return err.Error(), true
	}

	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Sprintf("file not found: %s", in.Path), true
	}
	if info.IsDir() {
		return fmt.Sprintf("%s is a directory, not a file. Use list_dir instead.", in.Path), true
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Sprintf("error reading file: %v", err), true
	}

	if len(data) > maxFileSize {
		return string(data[:maxFileSize]) + fmt.Sprintf("\n\n[truncated — file is %d bytes, showing first %d]", len(data), maxFileSize), false
	}

	return string(data), false
}

type searchCodeInput struct {
	Query       string `json:"query"`
	FilePattern string `json:"file_pattern"`
}

func (te *ToolExecutor) searchCode(input json.RawMessage) (string, bool) {
	var in searchCodeInput
	if err := json.Unmarshal(input, &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}

	if len(in.Query) > maxRegexLen {
		return fmt.Sprintf("query too long: max %d characters", maxRegexLen), true
	}

	re, err := regexp.Compile(in.Query)
	if err != nil {
		return fmt.Sprintf("invalid regex: %v", err), true
	}

	var matches []string
	_ = filepath.WalkDir(te.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] || (strings.HasPrefix(d.Name(), ".") && d.Name() != ".github") {
				return filepath.SkipDir
			}
			return nil
		}
		if len(matches) >= maxSearchMatches {
			return filepath.SkipAll
		}

		rel, _ := filepath.Rel(te.root, path)

		// Apply file pattern filter
		if in.FilePattern != "" {
			matched, _ := filepath.Match(in.FilePattern, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		// Skip binary files by checking extension
		if isBinaryExt(filepath.Ext(path)) {
			return nil
		}

		// Skip files too large for search
		info, err := d.Info()
		if err != nil || info.Size() > maxSearchFileSize {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if len(matches) >= maxSearchMatches {
				break
			}
			if re.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
		}

		return nil
	})

	if len(matches) == 0 {
		return "no matches found", false
	}

	result := strings.Join(matches, "\n")
	if len(matches) == maxSearchMatches {
		result += fmt.Sprintf("\n\n[showing first %d matches — refine your query for more specific results]", maxSearchMatches)
	}
	return result, false
}

type listDirInput struct {
	Path string `json:"path"`
}

func (te *ToolExecutor) listDir(input json.RawMessage) (string, bool) {
	var in listDirInput
	if err := json.Unmarshal(input, &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}

	abs, err := te.safePath(in.Path)
	if err != nil {
		return err.Error(), true
	}

	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Sprintf("directory not found: %s", in.Path), true
	}
	if !info.IsDir() {
		return fmt.Sprintf("%s is a file, not a directory. Use get_file instead.", in.Path), true
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return fmt.Sprintf("error reading directory: %v", err), true
	}

	var lines []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".github" {
			continue
		}
		if e.IsDir() {
			if ignoredDirs[name] {
				continue
			}
			lines = append(lines, name+"/")
		} else {
			lines = append(lines, name)
		}
	}

	if len(lines) == 0 {
		return "directory is empty", false
	}
	return strings.Join(lines, "\n"), false
}

func isBinaryExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".exe", ".bin", ".so", ".dll", ".dylib", ".a", ".o",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".svg",
		".woff", ".woff2", ".ttf", ".eot",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z",
		".pdf", ".doc", ".docx",
		".mp3", ".mp4", ".avi", ".mov",
		".wasm", ".pyc", ".class":
		return true
	}
	return false
}
