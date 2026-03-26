package main

import "testing"

func TestParsePRRef(t *testing.T) {
	tests := []struct {
		ref       string
		wantOwner string
		wantRepo  string
		wantPR    int
		wantErr   bool
	}{
		{"owner/repo#123", "owner", "repo", 123, false},
		{"my-org/my-repo#1", "my-org", "my-repo", 1, false},
		{"invalid", "", "", 0, true},
		{"owner/repo#abc", "", "", 0, true},
		{"noslash#123", "", "", 0, true},
	}
	for _, tt := range tests {
		owner, repo, pr, err := parsePRRef(tt.ref)
		if (err != nil) != tt.wantErr {
			t.Errorf("parsePRRef(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if owner != tt.wantOwner || repo != tt.wantRepo || pr != tt.wantPR {
			t.Errorf("parsePRRef(%q) = (%q, %q, %d), want (%q, %q, %d)",
				tt.ref, owner, repo, pr, tt.wantOwner, tt.wantRepo, tt.wantPR)
		}
	}
}

func TestSetupLogging(t *testing.T) {
	// Just verify it doesn't panic for all valid levels
	for _, level := range []string{"debug", "info", "warn", "error", "unknown"} {
		setupLogging(level)
	}
}
