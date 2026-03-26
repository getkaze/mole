package i18n

const (
	LangEN = "en"
	LangPT = "pt-BR"
)

type Messages struct {
	ReviewHeader   string
	Summary        string
	IssuesFound    string
	Suggestions    string
	NoIssues       string
	SplitNote      string // format: "Reviewed in %d groups due to PR size."
	MustFix        string
	ShouldFix      string
	Consider       string
	SeverityLabel  string
	CategoryLabel  string
	FileLabel      string
	DiagramHeader  string
}

var translations = map[string]Messages{
	LangEN: {
		ReviewHeader:  "Kite Review",
		Summary:       "Summary",
		IssuesFound:   "Issues Found",
		Suggestions:   "Suggestions",
		NoIssues:      "No issues found. Looking good! :white_check_mark:",
		SplitNote:     "Reviewed in %d groups due to PR size.",
		MustFix:       "Must Fix",
		ShouldFix:     "Should Fix",
		Consider:      "Consider",
		SeverityLabel: "Severity",
		CategoryLabel: "Category",
		FileLabel:     "File",
		DiagramHeader: "Diagram",
	},
	LangPT: {
		ReviewHeader:  "Kite Review",
		Summary:       "Resumo",
		IssuesFound:   "Problemas Encontrados",
		Suggestions:   "Sugestoes",
		NoIssues:      "Nenhum problema encontrado. Tudo certo! :white_check_mark:",
		SplitNote:     "Revisado em %d grupos devido ao tamanho do PR.",
		MustFix:       "Corrigir",
		ShouldFix:     "Deveria Corrigir",
		Consider:      "Considerar",
		SeverityLabel: "Severidade",
		CategoryLabel: "Categoria",
		FileLabel:     "Arquivo",
		DiagramHeader: "Diagrama",
	},
}

func Get(lang string) Messages {
	if m, ok := translations[lang]; ok {
		return m
	}
	return translations[LangEN]
}
