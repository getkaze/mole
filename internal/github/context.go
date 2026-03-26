package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v72/github"
)

const maxContextBytes = 200_000 // ~50K tokens

type ContextResult struct {
	Content   string
	Truncated bool
}

func LoadContext(ctx context.Context, client *gh.Client, owner, repo, ref string) (*ContextResult, error) {
	_, dirContent, resp, err := client.Repositories.GetContents(ctx, owner, repo, ".kite", &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return &ContextResult{}, nil
		}
		return nil, fmt.Errorf("reading .kite directory: %w", err)
	}

	var b strings.Builder
	totalBytes := 0
	truncated := false

	for _, entry := range dirContent {
		if truncated {
			break
		}
		if entry.GetType() == "file" && strings.HasSuffix(entry.GetName(), ".md") {
			content, err := fetchFileContent(ctx, client, owner, repo, entry.GetPath(), ref)
			if err != nil {
				continue
			}
			if totalBytes+len(content) > maxContextBytes {
				truncated = true
				continue
			}
			fmt.Fprintf(&b, "## %s\n\n%s\n\n", entry.GetPath(), content)
			totalBytes += len(content)
		} else if entry.GetType() == "dir" {
			err := walkDir(ctx, client, owner, repo, entry.GetPath(), ref, &b, &totalBytes, &truncated)
			if err != nil {
				continue
			}
		}
	}

	return &ContextResult{
		Content:   b.String(),
		Truncated: truncated,
	}, nil
}

func walkDir(ctx context.Context, client *gh.Client, owner, repo, path, ref string, b *strings.Builder, totalBytes *int, truncated *bool) error {
	_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return err
	}

	for _, entry := range dirContent {
		if *truncated {
			return nil
		}
		if entry.GetType() == "file" && strings.HasSuffix(entry.GetName(), ".md") {
			content, err := fetchFileContent(ctx, client, owner, repo, entry.GetPath(), ref)
			if err != nil {
				continue
			}
			if *totalBytes+len(content) > maxContextBytes {
				*truncated = true
				return nil
			}
			fmt.Fprintf(b, "## %s\n\n%s\n\n", entry.GetPath(), content)
			*totalBytes += len(content)
		} else if entry.GetType() == "dir" {
			walkDir(ctx, client, owner, repo, entry.GetPath(), ref, b, totalBytes, truncated)
		}
	}
	return nil
}

func fetchFileContent(ctx context.Context, client *gh.Client, owner, repo, path, ref string) (string, error) {
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", err
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return "", err
	}
	return content, nil
}
