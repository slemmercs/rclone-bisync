package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// StructDoc contains extracted documentation and schema for a Go struct.
type StructDoc struct {
	Name        string
	Description string
	Fields      []FieldDoc
}

// FieldDoc contains metadata, types, and comments for a struct field.
type FieldDoc struct {
	Name        string
	JSONKey     string
	Type        string
	Description string
	IsOptional  bool
	IsPointer   bool
}

// parseStructsFromFile parses specified structs from a single Go source file.
func parseStructsFromFile(filePath string, targetStructs []string) ([]StructDoc, error) {
	targets := make(map[string]bool, len(targetStructs))
	for _, t := range targetStructs {
		targets[t] = true
	}
	return parseStructsInternal(filePath, targets)
}

// parseAllStructsFromDir parses all exported structs from Go files in a directory.
func parseAllStructsFromDir(dirPath string) ([]StructDoc, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var allStructs []StructDoc
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filePath := filepath.Join(dirPath, entry.Name())
		structs, err := parseStructsInternal(filePath, nil)
		if err != nil {
			return nil, err
		}
		allStructs = append(allStructs, structs...)
	}
	return allStructs, nil
}

func parseStructsInternal(filePath string, targets map[string]bool) ([]StructDoc, error) {
	fset := token.NewFileSet()
	// #nosec G304 - doc generator reading repo models
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var results []StructDoc
	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structName := typeSpec.Name.Name
			if targets != nil && !targets[structName] {
				continue
			}
			if targets == nil && !ast.IsExported(structName) {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			doc := strings.TrimSpace(genDecl.Doc.Text())
			if doc == "" && typeSpec.Doc != nil {
				doc = strings.TrimSpace(typeSpec.Doc.Text())
			}

			sDoc := StructDoc{Name: structName, Description: doc}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				fieldName := field.Names[0].Name
				rawTag := extractTagValue(field.Tag, "json")
				if rawTag == "-" {
					continue
				}

				jsonKey, isOmitEmpty := parseJSONTag(rawTag, fieldName)
				_, isPointer := field.Type.(*ast.StarExpr)
				fDoc := extractFieldDoc(field, node.Comments, fset)
				fieldType := exprToString(field.Type)

				sDoc.Fields = append(sDoc.Fields, FieldDoc{
					Name:        fieldName,
					JSONKey:     jsonKey,
					Type:        fieldType,
					Description: fDoc,
					IsOptional:  isOmitEmpty || isPointer,
					IsPointer:   isPointer,
				})
			}
			results = append(results, sDoc)
		}
		return true
	})

	return results, nil
}

func parseJSONTag(tag string, fallback string) (string, bool) {
	if tag == "" {
		return fallback, false
	}
	parts := strings.Split(tag, ",")
	key := parts[0]
	if key == "" {
		key = fallback
	}
	isOmitEmpty := false
	for _, p := range parts[1:] {
		if p == "omitempty" {
			isOmitEmpty = true
			break
		}
	}
	return key, isOmitEmpty
}

func extractFieldDoc(field *ast.Field, comments []*ast.CommentGroup, fset *token.FileSet) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}
	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}
	targetLine := fset.Position(field.End()).Line
	for _, cg := range comments {
		if fset.Position(cg.Pos()).Line == targetLine {
			return strings.TrimSpace(cg.Text())
		}
	}
	return ""
}

func extractTagValue(tag *ast.BasicLit, key string) string {
	if tag == nil {
		return ""
	}
	val := strings.Trim(tag.Value, "`")
	pattern := regexp.MustCompile(key + `:"([^"]+)"`)
	matches := pattern.FindStringSubmatch(val)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return "any"
	}
}
