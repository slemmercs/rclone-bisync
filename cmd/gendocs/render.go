package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func renderFrontMatter(buf *bytes.Buffer) {
	buf.WriteString(`---
layout: default
title: Technical Reference
nav_order: 8
description: "Auto-generated technical reference for settings, API models, and MCP tools."
has_toc: true
---

# Technical & Settings Reference
{: .no_toc }

*Auto-generated directly from source code AST (` + "`internal/models/`" + `, ` + "`internal/profiles/`" + `, ` + "`internal/mcp/`" + `).*
{: .fs-3 .text-grey-dk-000 }

<div class="hero-badges">
  <span class="badge-online">● Source AST Synchronized</span>
  <span class="badge-indigo">Go 1.24+</span>
  <span class="badge-indigo">Zero Drift CI Guaranteed</span>
</div>

---

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
`)
}

func renderGlobalConfig(buf *bytes.Buffer, docs []StructDoc) {
	buf.WriteString("\n## Global System Configuration\n\n")
	buf.WriteString("Global settings define defaults and engine behavior across all synchronization workloads.\n\n")

	for _, doc := range docs {
		buf.WriteString(fmt.Sprintf("### `%s`\n\n", doc.Name))
		if doc.Description != "" {
			buf.WriteString(fmt.Sprintf("> %s\n\n", doc.Description))
		}
		buf.WriteString("| Setting Key | Type | Description |\n")
		buf.WriteString("| :--- | :--- | :--- |\n")
		for _, f := range doc.Fields {
			desc := f.Description
			if desc == "" {
				desc = "-"
			}
			cleanDesc := strings.ReplaceAll(desc, "\n", " ")
			buf.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", f.JSONKey, f.Type, cleanDesc))
		}
		buf.WriteString("\n---\n\n")
	}
}

func renderProfilePolicies(buf *bytes.Buffer, docs []StructDoc) {
	buf.WriteString("\n## Profile Workload Policies\n\n")
	buf.WriteString("Profile models govern path mappings, schedules, flag overrides, and failure thresholds.\n\n")

	for _, doc := range docs {
		buf.WriteString(fmt.Sprintf("### `%s`\n\n", doc.Name))
		if doc.Description != "" {
			buf.WriteString(fmt.Sprintf("> %s\n\n", doc.Description))
		}
		buf.WriteString("| Property Key | Type | Description |\n")
		buf.WriteString("| :--- | :--- | :--- |\n")
		for _, f := range doc.Fields {
			desc := f.Description
			if desc == "" {
				desc = "-"
			}
			cleanDesc := strings.ReplaceAll(desc, "\n", " ")
			buf.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", f.JSONKey, f.Type, cleanDesc))
		}
		buf.WriteString("\n---\n\n")
	}
}

func renderMCPTools(buf *bytes.Buffer, tools []MCPToolDoc) {
	buf.WriteString("\n## Model Context Protocol (MCP) AI Tools\n\n")
	buf.WriteString("Bisync Hub exposes native operational tools for AI agents (Claude, Cursor, Cline):\n\n")

	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	buf.WriteString("| Tool Identifier | Blast-Radius Guard | Description |\n")
	buf.WriteString("| :--- | :---: | :--- |\n")
	for _, tool := range tools {
		guard := "Read-Only"
		if tool.Confirmation {
			guard = "**SHA-256 Ticket**"
		} else if strings.Contains(tool.Name, "explore") || strings.Contains(tool.Name, "copy") || strings.Contains(tool.Name, "move") {
			guard = "Jail-Confined"
		}
		buf.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", tool.Name, guard, tool.Description))
	}

	buf.WriteString("\n### Tool Parameter Specifications\n\n")
	for _, tool := range tools {
		if len(tool.Parameters) == 0 {
			continue
		}
		buf.WriteString(fmt.Sprintf("#### `%s`\n\n", tool.Name))
		buf.WriteString("| Parameter | Type | Required | Description |\n")
		buf.WriteString("| :--- | :--- | :---: | :--- |\n")
		for _, p := range tool.Parameters {
			req := "No"
			if p.Required {
				req = "**Yes**"
			}
			pDesc := p.Description
			if pDesc == "" {
				pDesc = "-"
			}
			buf.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s |\n", p.Name, p.Type, req, pDesc))
		}
		buf.WriteString("\n")
	}
}
