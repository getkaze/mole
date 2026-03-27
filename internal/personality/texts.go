package personality

type texts struct {
	reviewHeader string

	// Mole personality
	summaryClean  string
	summaryIssues string
	moleCritical  string
	moleAttention string
	moleSuggestion string

	// Formal personality
	formalClean           string
	formalIssues          string
	formalCriticalPrefix  string
	formalAttentionPrefix string
	formalSuggestionPrefix string

	// Minimal personality
	minimalSummary string
	minimalClean   string

	// Severity labels
	labelCritical   string
	labelAttention  string
	labelSuggestion string
}

var allTexts = map[string]texts{
	"en": {
		reviewHeader: "Mole Review",

		summaryClean:  "🟢 **Looking good!** Mole dug deep and found nothing buried. Clean code, ready to merge! 🎉",
		summaryIssues: "🐭 **Mole dug deep into this PR!** Found %d issues to review. Score: **%d/100**",
		moleCritical:  "🔴 **Found something buried deep!** ",
		moleAttention: "🟡 **Mole spotted something!** ",
		moleSuggestion: "🟢 **Just a thought!** ",

		formalClean:           "No issues identified. The code meets quality standards.",
		formalIssues:          "Review complete. %d issues identified. Quality score: %d/100.",
		formalCriticalPrefix:  "Critical: ",
		formalAttentionPrefix: "Attention: ",
		formalSuggestionPrefix: "Suggestion: ",

		minimalSummary: "Score: %d/100 | %d issues",
		minimalClean:   "Clean. No issues.",

		labelCritical:   "Critical",
		labelAttention:  "Attention",
		labelSuggestion: "Suggestion",
	},
	"pt-BR": {
		reviewHeader: "Mole Review",

		summaryClean:  "🟢 **Tudo certo!** A toupeira cavou fundo e nao encontrou nada enterrado. Codigo limpo, pode mergear! 🎉",
		summaryIssues: "🐭 **A toupeira cavou fundo nesse PR!** Encontrou %d problemas para revisar. Score: **%d/100**",
		moleCritical:  "🔴 **Eita, achei algo enterrado aqui!** ",
		moleAttention: "🟡 **A toupeira sentiu algo!** ",
		moleSuggestion: "🟢 **So uma ideia!** ",

		formalClean:           "Nenhum problema identificado. O codigo atende aos padroes de qualidade.",
		formalIssues:          "Revisao concluida. %d problemas identificados. Score de qualidade: %d/100.",
		formalCriticalPrefix:  "Critico: ",
		formalAttentionPrefix: "Atencao: ",
		formalSuggestionPrefix: "Sugestao: ",

		minimalSummary: "Score: %d/100 | %d problemas",
		minimalClean:   "Limpo. Sem problemas.",

		labelCritical:   "Critico",
		labelAttention:  "Atencao",
		labelSuggestion: "Sugestao",
	},
}

func (e *Engine) texts() texts {
	t, ok := allTexts[e.lang]
	if !ok {
		return allTexts["en"]
	}
	return t
}
