package ast

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// TypeInfo represents a Go type (struct or interface) for diagram generation.
type TypeInfo struct {
	Name       string
	Kind       string // "struct" or "interface"
	Fields     []FieldInfo
	Methods    []MethodInfo
	Implements []string // interface names this type implements
}

// FieldInfo represents a struct field.
type FieldInfo struct {
	Name string
	Type string
}

// MethodInfo represents a method.
type MethodInfo struct {
	Name       string
	Params     string
	ReturnType string
}

// GenerateClassDiagram parses Go source in the given directory and produces
// a Mermaid classDiagram string.
func GenerateClassDiagram(dir string) (string, error) {
	types, err := extractTypes(dir)
	if err != nil {
		return "", err
	}

	if len(types) == 0 {
		return "", nil
	}

	resolveImplements(types)

	return renderMermaid(types), nil
}

func extractTypes(dir string) ([]TypeInfo, error) {
	fset := token.NewFileSet()
	var types []TypeInfo

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}

		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				info := TypeInfo{Name: typeSpec.Name.Name}

				switch t := typeSpec.Type.(type) {
				case *ast.StructType:
					info.Kind = "struct"
					if t.Fields != nil {
						for _, field := range t.Fields.List {
							typeName := typeString(field.Type)
							for _, name := range field.Names {
								info.Fields = append(info.Fields, FieldInfo{
									Name: name.Name,
									Type: typeName,
								})
							}
						}
					}
				case *ast.InterfaceType:
					info.Kind = "interface"
					if t.Methods != nil {
						for _, method := range t.Methods.List {
							if len(method.Names) > 0 {
								info.Methods = append(info.Methods, MethodInfo{
									Name: method.Names[0].Name,
								})
							}
						}
					}
				default:
					continue
				}

				types = append(types, info)
			}
		}

		// Extract methods with receivers
		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil {
				continue
			}

			recvName := receiverName(funcDecl.Recv)
			if recvName == "" {
				continue
			}

			for i := range types {
				if types[i].Name == recvName {
					types[i].Methods = append(types[i].Methods, MethodInfo{
						Name: funcDecl.Name.Name,
					})
				}
			}
		}
	}

	return types, nil
}

// resolveImplements checks which structs implement which interfaces.
func resolveImplements(types []TypeInfo) {
	var interfaces []TypeInfo
	var structs []*TypeInfo

	for i := range types {
		if types[i].Kind == "interface" {
			interfaces = append(interfaces, types[i])
		} else if types[i].Kind == "struct" {
			structs = append(structs, &types[i])
		}
	}

	for _, s := range structs {
		structMethods := make(map[string]bool)
		for _, m := range s.Methods {
			structMethods[m.Name] = true
		}

		for _, iface := range interfaces {
			if len(iface.Methods) == 0 {
				continue
			}
			allMatch := true
			for _, m := range iface.Methods {
				if !structMethods[m.Name] {
					allMatch = false
					break
				}
			}
			if allMatch {
				s.Implements = append(s.Implements, iface.Name)
			}
		}
	}
}

func renderMermaid(types []TypeInfo) string {
	var b strings.Builder
	b.WriteString("classDiagram\n")

	for _, t := range types {
		if t.Kind == "interface" {
			fmt.Fprintf(&b, "    class %s {\n", t.Name)
			fmt.Fprintf(&b, "        <<interface>>\n")
			for _, m := range t.Methods {
				fmt.Fprintf(&b, "        +%s()\n", m.Name)
			}
			b.WriteString("    }\n")
		} else {
			fmt.Fprintf(&b, "    class %s {\n", t.Name)
			for _, f := range t.Fields {
				fmt.Fprintf(&b, "        +%s %s\n", f.Type, f.Name)
			}
			for _, m := range t.Methods {
				fmt.Fprintf(&b, "        +%s()\n", m.Name)
			}
			b.WriteString("    }\n")
		}
	}

	// Render relationships
	for _, t := range types {
		for _, impl := range t.Implements {
			fmt.Fprintf(&b, "    %s ..|> %s\n", t.Name, impl)
		}
	}

	return b.String()
}

func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return typeString(recv.List[0].Type)
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeString(t.X)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	default:
		return "any"
	}
}
