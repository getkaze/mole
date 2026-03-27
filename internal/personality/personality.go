package personality

import "fmt"

// Mode controls the tone of review output.
type Mode string

const (
	ModeMole    Mode = "mole"
	ModeFormal  Mode = "formal"
	ModeMinimal Mode = "minimal"
)

// Engine generates personality-aware review text.
type Engine struct {
	mode Mode
	lang string // "en" or "pt-BR"
}

// New creates a personality engine. Falls back to ModeMole if mode is invalid.
func New(mode, lang string) *Engine {
	m := Mode(mode)
	switch m {
	case ModeMole, ModeFormal, ModeMinimal:
	default:
		m = ModeMole
	}
	if lang == "" {
		lang = "en"
	}
	return &Engine{mode: m, lang: lang}
}

// Summary returns the review header text.
func (e *Engine) Summary(score int, issueCount int) string {
	t := e.texts()
	switch e.mode {
	case ModeMole:
		if issueCount == 0 {
			return t.summaryClean
		}
		return fmt.Sprintf(t.summaryIssues, issueCount, score)
	case ModeFormal:
		if issueCount == 0 {
			return t.formalClean
		}
		return fmt.Sprintf(t.formalIssues, issueCount, score)
	case ModeMinimal:
		return fmt.Sprintf(t.minimalSummary, score, issueCount)
	default:
		return fmt.Sprintf("Score: %d | Issues: %d", score, issueCount)
	}
}

// CleanPR returns the message for a PR with zero issues.
func (e *Engine) CleanPR() string {
	t := e.texts()
	switch e.mode {
	case ModeMole:
		return t.summaryClean
	case ModeFormal:
		return t.formalClean
	case ModeMinimal:
		return t.minimalClean
	default:
		return t.formalClean
	}
}

// IssuePrefix returns the prefix for an issue based on severity.
func (e *Engine) IssuePrefix(severity string) string {
	t := e.texts()
	switch e.mode {
	case ModeMole:
		switch severity {
		case "critical":
			return t.moleCritical
		case "attention":
			return t.moleAttention
		case "suggestion":
			return t.moleSuggestion
		default:
			return t.moleSuggestion
		}
	case ModeFormal:
		switch severity {
		case "critical":
			return t.formalCriticalPrefix
		case "attention":
			return t.formalAttentionPrefix
		case "suggestion":
			return t.formalSuggestionPrefix
		default:
			return t.formalSuggestionPrefix
		}
	case ModeMinimal:
		return ""
	default:
		return ""
	}
}

// ScoreBadge returns a visual representation of the score.
func (e *Engine) ScoreBadge(score int) string {
	switch e.mode {
	case ModeMole:
		switch {
		case score >= 90:
			return fmt.Sprintf("🟢 **%d/100**", score)
		case score >= 70:
			return fmt.Sprintf("🟡 **%d/100**", score)
		default:
			return fmt.Sprintf("🔴 **%d/100**", score)
		}
	case ModeFormal:
		return fmt.Sprintf("**Quality Score: %d/100**", score)
	case ModeMinimal:
		return fmt.Sprintf("%d/100", score)
	default:
		return fmt.Sprintf("%d/100", score)
	}
}

// SeverityBadge returns the visual indicator for a severity level.
func (e *Engine) SeverityBadge(severity string) string {
	switch e.mode {
	case ModeMole, ModeFormal:
		switch severity {
		case "critical":
			return "🔴"
		case "attention":
			return "🟡"
		case "suggestion":
			return "🟢"
		default:
			return "🟢"
		}
	case ModeMinimal:
		switch severity {
		case "critical":
			return "[C]"
		case "attention":
			return "[A]"
		case "suggestion":
			return "[S]"
		default:
			return "[S]"
		}
	default:
		return ""
	}
}

// SeverityLabel returns the localized label for a severity level.
func (e *Engine) SeverityLabel(severity string) string {
	t := e.texts()
	switch severity {
	case "critical":
		return t.labelCritical
	case "attention":
		return t.labelAttention
	case "suggestion":
		return t.labelSuggestion
	default:
		return t.labelSuggestion
	}
}

// ReviewHeader returns the review header text.
func (e *Engine) ReviewHeader() string {
	return e.texts().reviewHeader
}
