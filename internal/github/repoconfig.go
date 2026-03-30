package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v72/github"
	"gopkg.in/yaml.v3"
)

type RepoConfig struct {
	Language          string            `yaml:"language"`            // en, pt-BR
	Personality       string            `yaml:"personality"`         // mole, formal, minimal
	MinSeverity       string            `yaml:"min_severity"`        // suggestion, attention, critical
	MaxInlineComments int               `yaml:"max_inline_comments"` // 0 = unlimited
	Ignore            []string          `yaml:"ignore"`              // glob patterns for files to skip
	Architecture      *ArchitectureRule `yaml:"architecture"`
	Instructions      string            `yaml:"instructions"`        // custom instructions for LLM
}

type ArchitectureRule struct {
	Style  string  `yaml:"style"` // clean, hexagonal, layered, none
	Layers []Layer `yaml:"layers"`
}

type Layer struct {
	Name      string   `yaml:"name"`
	Path      string   `yaml:"path"`
	CanImport []string `yaml:"can_import"`
}

func LoadRepoConfig(ctx context.Context, client *gh.Client, owner, repo, ref string) (*RepoConfig, error) {
	content, err := fetchFileContent(ctx, client, owner, repo, ".mole/config.yaml", ref)
	if err != nil {
		// Not found is fine — return empty config (caller applies defaults)
		return &RepoConfig{}, nil
	}

	cfg := &RepoConfig{}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, fmt.Errorf("parsing .mole/config.yaml: %w", err)
	}

	return cfg, nil
}

// ApplyDefaults fills empty fields with the provided server-level defaults.
func (rc *RepoConfig) ApplyDefaults(language, personality string) {
	if rc.Language == "" {
		rc.Language = language
	}
	if rc.Personality == "" {
		rc.Personality = personality
	}
}
