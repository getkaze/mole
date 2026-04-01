package personality

type texts struct {
	reviewHeader string

	// Mole personality
	summaryClean  string
	summaryIssues string
	moleCritical  string
	moleAttention string

	// Formal personality
	formalClean           string
	formalIssues          string
	formalCriticalPrefix  string
	formalAttentionPrefix string

	// Minimal personality
	minimalSummary string
	minimalClean   string

	// Severity labels
	labelCritical  string
	labelAttention string

	// Exploration messages
	exploreCloning   string
	exploreCloned    string
	exploreCloneFail string
}

var allTexts = map[string]texts{
	"en": {
		reviewHeader: "Mole Review",

		summaryClean:  "🟢 **Looking good!** Mole dug deep and found nothing buried. Clean code, ready to merge! 🎉",
		summaryIssues: "🐭 **Mole dug deep into this PR!** Found %d issues to review. Score: **%d/100**",
		moleCritical:  "🔴 **Found something buried deep!** ",
		moleAttention: "🟡 **Mole spotted something!** ",

		formalClean:          "No issues identified. The code meets quality standards.",
		formalIssues:         "Review complete. %d issues identified. Quality score: %d/100.",
		formalCriticalPrefix: "Critical: ",
		formalAttentionPrefix: "Attention: ",

		minimalSummary: "Score: %d/100 | %d issues",
		minimalClean:   "Clean. No issues.",

		labelCritical:  "Critical",
		labelAttention: "Attention",

		exploreCloning:   "🔍 Cloning repository for the first time. This may take a moment...",
		exploreCloned:    "🔍 Repository cloned. Exploring codebase for context...",
		exploreCloneFail: "🔍 Failed to prepare repository for contextual review. Reviewing with diff only.",
	},
	"pt-BR": {
		reviewHeader: "Mole Review",

		summaryClean:  "🟢 **Eh mole!** A toupeira cavou fundo e não encontrou nada enterrado. Código limpo, pode mergear! 🎉",
		summaryIssues: "🐭 **A toupeira cavou fundo nesse PR!** Encontrou %d problemas para revisar. Score: **%d/100**",
		moleCritical:  "🔴 **Eita, achei algo enterrado aqui!** ",
		moleAttention: "🟡 **A toupeira sentiu algo!** ",

		formalClean:           "Nenhum problema identificado. O código atende aos padrões de qualidade.",
		formalIssues:          "Revisão concluída. %d problemas identificados. Score de qualidade: %d/100.",
		formalCriticalPrefix:  "Crítico: ",
		formalAttentionPrefix: "Atenção: ",

		minimalSummary: "Score: %d/100 | %d problemas",
		minimalClean:   "Limpo. Sem problemas.",

		labelCritical:  "Crítico",
		labelAttention: "Atenção",

		exploreCloning:   "🔍 Clonando repositório pela primeira vez. Pode levar um momento...",
		exploreCloned:    "🔍 Repositório clonado. Explorando o código para contexto...",
		exploreCloneFail: "🔍 Falha ao preparar repositório para revisão contextual. Revisando apenas com a diff.",
	},
}

func (e *Engine) texts() texts {
	t, ok := allTexts[e.lang]
	if !ok {
		return allTexts["en"]
	}
	return t
}
