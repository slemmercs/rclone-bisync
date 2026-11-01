# Implementation Plan: Documentation, Roadmap, and GitHub Pages Showcase (`v1.0.0-beta.20`)

## Proposed Changes

Bring all repository documentation, roadmap planning records, and GitHub Pages assets up to date with the latest features delivered through `v1.0.0-beta.20` (notably the native Model Context Protocol server, unified logging ledger, and quality hardening), integrate 3 new dashboard screenshots and 7 refreshed screenshots, and document the upcoming Tier 1 curated options & guided `+ Add Flag` row builder.

### Affected Components & Packages

- `docs/roadmap/`:
  - `ROADMAP.md`: Update delivered milestone table and details up to `v1.0.0-beta.20`; record MCP server, unified logging, and technical debt sprints; add active priority for Curated Tier 1 Standard Options & Guided `+ Add Flag` Builder; deduplicate secondary priorities.
  - `backlog.md`: Move completed items (`PR #49–#56`) to Completed Epics; detail the upcoming Tier 1 / Tier 2 provider flags and interactive builder backlog item; renumber active items.
- `README.md`:
  - Add native MCP server to differentiators, capabilities, and quickstart commands; update quality check stage count to 7; update clone URL and showcase new screenshots (`fleet-overview`, `profile-settings`, and updated `audit-ledger`).
- `site/`:
  - `index.md`: Add Pillar 6 reference, feature fleet overview screenshot in embedded dashboard tour.
  - `usps/index.md`: Add Pillar 6 listing.
  - `usps/06-mcp-agent-server.md` (NEW): Full architectural specification and showcase of the native Model Context Protocol server.
  - `gallery.md`: Add Multi-Profile Fleet Overview & Workload Management section; update all 7 refreshed screenshots; replace obsolete `-v2` file references.
  - `usps/04-audit-ledger.md`: Update screenshot reference from `-v2` to `bisync-hub_audit-ledger.png`.
  - `roadmap.md`: Align with `ROADMAP.md` updates; move Hybrid Fleet, Deduplication, and MCP server to Shipped.
  - `architecture.md`: Add MCP Confirmation Ticket & Parameter Fingerprinting safety mechanisms; add copyleft license guard to `cmd/check`.
  - `quickstart.md`: Fix binary artifact download URLs to `bisync-hub-linux-amd64`; add MCP agent configuration section for Cline and Claude Desktop.
  - `comparison.md`: Update webhook alerts to built-in; add rows for Multi-Profile Fleet, Remote Hash Deduplication, and Native MCP Server.

### Safety & Error Handling

- Verify all Jekyll front matter, markdown links, and relative image links across `site/`.
- Ensure zero broken links or missing image assets.
- Adhere to the Slate-900 / Indigo-600 design system across all site additions.

## Verification Plan

- `git status` check on all modified screenshot and documentation paths.
- `./scripts/check.sh all` to run all 7 quality gate stages.
- Link and front-matter integrity check on `site/`.








