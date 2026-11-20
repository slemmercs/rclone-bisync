package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
