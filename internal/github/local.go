package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LocalGateway implements Gateway by reading PR data from a local fixtures
// directory. Reviews are printed to stdout instead of posted to GitHub.
type LocalGateway struct {
	dir string
}

// NewLocalGateway creates a Gateway that reads from the given directory.
func NewLocalGateway(dir string) *LocalGateway {
	return &LocalGateway{dir: dir}
}

// NewLocalGatewayFactory returns a GatewayFactory that always returns the
// same LocalGateway, ignoring the installID parameter.
func NewLocalGatewayFactory(dir string) GatewayFactory {
	gw := NewLocalGateway(dir)
	return func(_ int64) Gateway { return gw }
}

func (l *LocalGateway) GetPRInfo(_ context.Context, _ string, _ int) (*PRInfo, error) {
	data, err := os.ReadFile(filepath.Join(l.dir, "pr.json"))
	if err != nil {
		return nil, fmt.Errorf("reading pr.json: %w", err)
	}
	var info PRInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parsing pr.json: %w", err)
	}
	return &info, nil
}

func (l *LocalGateway) FetchDiff(_ context.Context, _ string, _ int) ([]FileDiff, error) {
	data, err := os.ReadFile(filepath.Join(l.dir, "diff.patch"))
	if err != nil {
		return nil, fmt.Errorf("reading diff.patch: %w", err)
	}
	return parsePatch(string(data)), nil
}

func (l *LocalGateway) LoadContext(_ context.Context, _, _ string) (*ContextResult, error) {
	data, err := os.ReadFile(filepath.Join(l.dir, "context.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return &ContextResult{}, nil
		}
		return nil, fmt.Errorf("reading context.md: %w", err)
	}
	return &ContextResult{Content: string(data)}, nil
}

func (l *LocalGateway) LoadRepoConfig(_ context.Context, _, _ string) (*RepoConfig, error) {
	data, err := os.ReadFile(filepath.Join(l.dir, "config.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return &RepoConfig{}, nil
		}
		return nil, fmt.Errorf("reading config.yaml: %w", err)
	}
	cfg := &RepoConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}
	return cfg, nil
}

func (l *LocalGateway) PostReview(_ context.Context, _ string, _ int, _ string, data *ReviewData) (*PostReviewResult, error) {
	var b strings.Builder

	b.WriteString(data.Body)

	if len(data.Comments) > 0 {
		fmt.Fprintf(&b, "\n\n---\n\n## Inline Comments (%d)\n\n", len(data.Comments))
		for i, c := range data.Comments {
			fmt.Fprintf(&b, "### [%d] `%s:%d`\n\n%s\n\n", i+1, c.File, c.Line, c.Body)
		}
	}

	output := b.String()

	// Print to stdout
	fmt.Println(output)

	// Save to output.md alongside the fixtures
	outPath := filepath.Join(l.dir, "output.md")
	if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
		slog.Warn("failed to save review output", "path", outPath, "error", err)
	} else {
		fmt.Printf("\nReview saved to %s\n", outPath)
	}

	return &PostReviewResult{}, nil
}

func (l *LocalGateway) AddReaction(_ context.Context, repo string, pr int, _ int64, reaction string) {
	slog.Debug("local: reaction skipped", "repo", repo, "pr", pr, "reaction", reaction)
}

func (l *LocalGateway) PostComment(_ context.Context, _ string, _ int, body string) (int64, error) {
	fmt.Printf("[comment] %s\n", body)
	return 1, nil
}

func (l *LocalGateway) EditComment(_ context.Context, _ string, _ int, _ int64, body string) error {
	fmt.Printf("[comment updated] %s\n", body)
	return nil
}

// parsePatch parses a unified diff (git diff output) into FileDiff entries.
func parsePatch(raw string) []FileDiff {
	var diffs []FileDiff
	lines := strings.Split(raw, "\n")

	var current *FileDiff
	var patch strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "diff --git ") {
			// Flush previous file
			if current != nil {
				current.Patch = strings.TrimRight(patch.String(), "\n")
				diffs = append(diffs, *current)
			}

			current = &FileDiff{}
			patch.Reset()

			// Extract filename from "diff --git a/path b/path"
			parts := strings.SplitN(line, " b/", 2)
			if len(parts) == 2 {
				current.Filename = parts[1]
			}
			current.Status = "modified"
			continue
		}

		if current == nil {
			continue
		}

		// Detect status from diff headers
		if strings.HasPrefix(line, "new file mode") {
			current.Status = "added"
			continue
		}
		if strings.HasPrefix(line, "deleted file mode") {
			current.Status = "removed"
			continue
		}
		if strings.HasPrefix(line, "rename from") || strings.HasPrefix(line, "rename to") {
			current.Status = "renamed"
			continue
		}

		// Skip diff metadata lines (index, ---, +++)
		if strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			continue
		}

		// Accumulate hunk content (starts with @@ or context/add/remove lines)
		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "+") ||
			strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
			patch.WriteString(line)
			patch.WriteByte('\n')
		}
	}

	// Flush last file
	if current != nil {
		current.Patch = strings.TrimRight(patch.String(), "\n")
		diffs = append(diffs, *current)
	}

	return diffs
}
