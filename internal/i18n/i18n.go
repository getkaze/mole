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
