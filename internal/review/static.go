package review

import (
	"log/slog"

	moleast "github.com/getkaze/mole/internal/ast"
	ghclient "github.com/getkaze/mole/internal/github"
	"github.com/getkaze/mole/internal/arch"
	"github.com/getkaze/mole/internal/llm"
	"github.com/getkaze/mole/internal/security"
)

// StaticAnalysisResult holds the combined output of static analysis.
type StaticAnalysisResult struct {
	Comments []llm.InlineComment
	Diagrams []string
}

// RunStaticAnalysis executes architecture validation, security scanning,
// and AST class diagram generation against a local repository.
func RunStaticAnalysis(repoPath string, repoCfg *ghclient.RepoConfig, deep bool) *StaticAnalysisResult {
	result := &StaticAnalysisResult{}

	// Architecture validation
	if repoCfg != nil && repoCfg.Architecture != nil {
		archComments := arch.Validate(repoPath, repoCfg.Architecture)
		result.Comments = append(result.Comments, archComments...)
		slog.Debug("architecture validation", "violations", len(archComments))
	}

	// Security scanning
	secComments := security.Scan(repoPath)
	result.Comments = append(result.Comments, secComments...)
	slog.Debug("security scan", "findings", len(secComments))

	// AST class diagrams (deep review only)
	if deep {
		diagram, err := moleast.GenerateClassDiagram(repoPath)
		if err != nil {
			slog.Warn("AST class diagram failed", "error", err)
		} else if diagram != "" {
			result.Diagrams = append(result.Diagrams, diagram)
		}
	}

	return result
}

// MergeStaticAnalysis merges static analysis results into the LLM review response.
func MergeStaticAnalysis(resp *llm.ReviewResponse, static *StaticAnalysisResult) {
	if static == nil {
		return
	}
	resp.Comments = append(resp.Comments, static.Comments...)
	resp.Diagrams = append(resp.Diagrams, static.Diagrams...)
}
