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

// StructDoc contains extracted documentation for a Go struct.
type StructDoc struct {
	Name        string
	Description string
	Fields      []FieldDoc
}

// FieldDoc contains metadata and documentation for a struct field.
type FieldDoc struct {
	Name        string
	JSONKey     string
	Type        string
	Description string
}

// MCPToolDoc contains documentation for an MCP tool.
type MCPToolDoc struct {
	Name         string
	Description  string
	Confirmation bool
	Parameters   []MCPParamDoc
}

// MCPParamDoc contains metadata for an MCP tool argument.
type MCPParamDoc struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

func parseStructsFromFile(filePath string, targetStructs []string) ([]StructDoc, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	targets := make(map[string]bool)
	for _, t := range targetStructs {
		targets[t] = true
	}

	var results []StructDoc

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !targets[typeSpec.Name.Name] {
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

			sDoc := StructDoc{
				Name:        typeSpec.Name.Name,
				Description: doc,
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				fieldName := field.Names[0].Name
				jsonKey := extractTagValue(field.Tag, "json")
				if jsonKey == "-" {
					continue
				}
				if jsonKey == "" {
					jsonKey = fieldName
				}
				jsonKey = strings.Split(jsonKey, ",")[0]

				fDoc := ""
				if field.Doc != nil {
					fDoc = strings.TrimSpace(field.Doc.Text())
				}
				if fDoc == "" && field.Comment != nil {
					fDoc = strings.TrimSpace(field.Comment.Text())
				}
				if fDoc == "" {
					// Fallback to checking line comment in Comments
					fDoc = findTrailingComment(node.Comments, fset, fset.Position(field.End()).Line)
				}

				fieldType := exprToString(field.Type)

				sDoc.Fields = append(sDoc.Fields, FieldDoc{
					Name:        fieldName,
					JSONKey:     jsonKey,
					Type:        fieldType,
					Description: fDoc,
				})
			}

			results = append(results, sDoc)
		}
		return true
	})

	return results, nil
}

func findTrailingComment(comments []*ast.CommentGroup, fset *token.FileSet, targetLine int) string {
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

var (
	toolNameRe  = regexp.MustCompile(`NewTool\(\s*"([^"]+)"`)
	descRe      = regexp.MustCompile(`WithDescription\(\s*"([^"]+)"`)
	paramRe     = regexp.MustCompile(`mcp\.With(String|Boolean|Number)\(\s*"([^"]+)"`)
	paramDescRe = regexp.MustCompile(`mcp\.Description\(\s*"([^"]+)"`)
)

func parseMCPTools(mcpDir string) ([]MCPToolDoc, error) {
	entries, err := os.ReadDir(mcpDir)
	if err != nil {
		return nil, err
	}

	var tools []MCPToolDoc
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "tools_") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(mcpDir, entry.Name())
		cleanPath := filepath.Clean(filePath)
		// #nosec G304 - doc generator reading repo mcp tools
		content, err := os.ReadFile(cleanPath)
		if err != nil {
			continue
		}

		blocks := strings.Split(string(content), "s.mcpServer.AddTool(")
		for i := 1; i < len(blocks); i++ {
			block := blocks[i]
			nameMatch := toolNameRe.FindStringSubmatch(block)
			if len(nameMatch) < 2 {
				continue
			}
			toolName := nameMatch[1]
			if seen[toolName] {
				continue
			}
			seen[toolName] = true

			desc := ""
			descMatch := descRe.FindStringSubmatch(block)
			if len(descMatch) > 1 {
				desc = descMatch[1]
			}

			hasConfirmation := strings.Contains(block, "confirmation_token")

			var params []MCPParamDoc
			lines := strings.Split(block, "\n")
			for _, line := range lines {
				pMatch := paramRe.FindStringSubmatch(line)
				if len(pMatch) >= 3 {
					pType := strings.ToLower(pMatch[1])
					pName := pMatch[2]
					pDesc := ""
					if pdMatch := paramDescRe.FindStringSubmatch(line); len(pdMatch) > 1 {
						pDesc = pdMatch[1]
					}
					params = append(params, MCPParamDoc{
						Name:        pName,
						Type:        pType,
						Required:    strings.Contains(line, "Required()"),
						Description: pDesc,
					})
				}
			}

			tools = append(tools, MCPToolDoc{
				Name:         toolName,
				Description:  desc,
				Confirmation: hasConfirmation,
				Parameters:   params,
			})
		}
	}

	return tools, nil
}
