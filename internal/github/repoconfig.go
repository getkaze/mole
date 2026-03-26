package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v72/github"
	"gopkg.in/yaml.v3"
)

type RepoConfig struct {
	Language string `yaml:"language"` // en, pt-BR
}

func LoadRepoConfig(ctx context.Context, client *gh.Client, owner, repo, ref string) (*RepoConfig, error) {
	content, err := fetchFileContent(ctx, client, owner, repo, ".kite/config.yaml", ref)
	if err != nil {
		// Not found is fine — return defaults
		return &RepoConfig{Language: "en"}, nil
	}

	cfg := &RepoConfig{Language: "en"}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, fmt.Errorf("parsing .kite/config.yaml: %w", err)
	}

	return cfg, nil
}
