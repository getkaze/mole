package security

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkaze/mole/internal/llm"
)

// Scan checks Go source files for common security anti-patterns.
func Scan(repoPath string) []llm.InlineComment {
	var comments []llm.InlineComment

	filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, _ := filepath.Rel(repoPath, path)
		issues := scanFile(path, rel)
		comments = append(comments, issues...)
		return nil
	})

	return comments
}

func scanFile(absPath, relPath string) []llm.InlineComment {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, parser.AllErrors)
	if err != nil {
		return nil
	}

	var issues []llm.InlineComment

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if issue := checkSQLConcat(fset, node, relPath); issue != nil {
				issues = append(issues, *issue)
			}
			if issue := checkExecCommand(fset, node, relPath); issue != nil {
				issues = append(issues, *issue)
			}
		case *ast.BasicLit:
			if issue := checkHardcodedSecret(fset, node, relPath, f); issue != nil {
				issues = append(issues, *issue)
			}
		}
		return true
	})

	return issues
}

// checkSQLConcat detects SQL queries built with string concatenation.
// Looks for patterns like: db.Query("SELECT ... " + variable)
func checkSQLConcat(fset *token.FileSet, call *ast.CallExpr, file string) *llm.InlineComment {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	methodName := sel.Sel.Name
	sqlMethods := map[string]bool{
		"Query": true, "QueryRow": true, "Exec": true,
		"QueryContext": true, "QueryRowContext": true, "ExecContext": true,
	}

	if !sqlMethods[methodName] {
		return nil
	}

	// Check if any argument is a binary expression (string concat)
	for _, arg := range call.Args {
		if isBinaryConcat(arg) {
			pos := fset.Position(call.Pos())
			return &llm.InlineComment{
				File:        file,
				Line:        pos.Line,
				Category:    "Security",
				Subcategory: "SQL Injection",
				Severity:    "critical",
				Message:     fmt.Sprintf("SQL query built with string concatenation in %s(). Use parameterized queries instead.", methodName),
			}
		}
	}

	return nil
}

// checkExecCommand detects os/exec calls that may use unsanitized input.
func checkExecCommand(fset *token.FileSet, call *ast.CallExpr, file string) *llm.InlineComment {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// Check for exec.Command with non-literal arguments
	pkg, pkgOk := sel.X.(*ast.Ident)
	if !pkgOk || pkg.Name != "exec" {
		return nil
	}
	if sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext" {
		return nil
	}

	// Check if any argument beyond the first is a non-literal (potential injection)
	for i, arg := range call.Args {
		if i == 0 && sel.Sel.Name == "CommandContext" {
			continue // skip context arg
		}
		if _, isLit := arg.(*ast.BasicLit); !isLit {
			// Non-literal argument — could be user input
			if _, isIdent := arg.(*ast.Ident); isIdent {
				pos := fset.Position(call.Pos())
				return &llm.InlineComment{
					File:        file,
					Line:        pos.Line,
					Category:    "Security",
					Subcategory: "SQL Injection", // reusing closest subcategory — command injection
					Severity:    "critical",
					Message:     "exec.Command called with variable arguments. Ensure inputs are sanitized to prevent command injection.",
				}
			}
		}
	}

	return nil
}

// checkHardcodedSecret detects string literals that look like hardcoded secrets.
func checkHardcodedSecret(fset *token.FileSet, lit *ast.BasicLit, file string, f *ast.File) *llm.InlineComment {
	if lit.Kind != token.STRING {
		return nil
	}

	val := strings.Trim(lit.Value, `"` + "`")

	// Check for common secret patterns
	secretPatterns := []string{
		"sk-",     // API keys
		"sk_live", // Stripe
		"ghp_",    // GitHub PAT
		"glpat-",  // GitLab PAT
		"AKIA",    // AWS access key
	}

	for _, pattern := range secretPatterns {
		if strings.HasPrefix(val, pattern) && len(val) > len(pattern)+8 {
			pos := fset.Position(lit.Pos())
			return &llm.InlineComment{
				File:        file,
				Line:        pos.Line,
				Category:    "Security",
				Subcategory: "Secrets Exposure",
				Severity:    "critical",
				Message:     "Possible hardcoded secret detected. Use environment variables or a secret manager instead.",
			}
		}
	}

	return nil
}

func isBinaryConcat(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return bin.Op == token.ADD
}
