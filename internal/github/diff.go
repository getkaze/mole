package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v72/github"
)

type FileDiff struct {
	Filename string
	Status   string // "added", "modified", "removed", "renamed"
	Patch    string
	TooLarge bool
}

func FetchDiff(ctx context.Context, client *gh.Client, owner, repo string, prNumber int) ([]FileDiff, error) {
	var allFiles []*gh.CommitFile
	opts := &gh.ListOptions{PerPage: 100}

	for {
		files, resp, err := client.PullRequests.ListFiles(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("listing PR files: %w", err)
		}
		allFiles = append(allFiles, files...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	diffs := make([]FileDiff, 0, len(allFiles))
	for _, f := range allFiles {
		d := FileDiff{
			Filename: f.GetFilename(),
			Status:   f.GetStatus(),
			Patch:    f.GetPatch(),
		}
		if f.GetPatch() == "" && f.GetChanges() > 0 {
			d.TooLarge = true
		}
		diffs = append(diffs, d)
	}

	return diffs, nil
}
