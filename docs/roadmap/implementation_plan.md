# Implementation Plan: 100% Feature Parity Migration (Legacy Alpine.js to Vue 3 SPA Cockpit)

## Overview

Achieve 100% functional, operational, and visual feature parity with the legacy Alpine.js cockpit by implementing all missing components, modals, tools, and telemetry views in the Vue 3 + PrimeVue v4 + Tailwind CSS single-page application while preserving the self-contained zero-runtime single-binary invariant.

The previous dashboard consisted of an unbundled Alpine.js + Go template architecture containing rich operational capabilities across multiple domains: interactive file manipulation and previewing in the Dual Explorer, hash-based deduplication workspaces on cloud remotes, multi-stream correlated daemon diagnostics, a run-aggregated timeline ledger with repetitive notice folding, an in-app slide-over reference manual with live self-diagnostics, and dedicated modals for system health, global fleet settings, and MCP inspection.

This plan methodically ports every legacy capability into modular, cohesive Vue 3 Single-File Components (SFCs), Pinia stores, and typed TypeScript contracts matching the existing Go REST endpoints in `internal/server/routes.go`.

---

## Types

### 1. File Operations & Preview Contracts (`src/types/explorer.ts`)
- `FilePreviewResponse`: name, path, remote, size, mime_type, mod_time, content, data_url, is_binary, is_pdf, truncated, hash, error.
- `CompareRequest`: remote1, path1, remote2, path2, profile_id.
- `CompareClusterItem`: remote, name, path, size, primary_hash, mod_time, error.
- `CompareResponse`: algorithm, matches, clusters, error.
- `MoveItemRequest`: src_remote, src_path, dst_remote, dst_path, is_dir, profile_id.
- `MkdirRequest`: remote, path, profile_id.
- `DeleteItemRequest`: remote, path, is_dir, profile_id, confirmation_token.

### 2. Deduplication Workspace Contracts (`src/types/dedupe.ts`)
- `DedupeMode`: interactive, oldest, newest, rename, largest, smallest, list.
- `DedupeCollisionCandidate`: id, path, name, size, hash, mod_time, remote.
- `DedupeGroup`: hash, size, count, items.
- `DedupeRunRequest`: remote, path, mode, by_hash, dedupe_both, profile_id, confirmation_token.
- `DedupeRunResponse`: status, total_collisions, groups, confirmation_token, deleted_count, log_lines, error.

### 3. Run-Aggregated Ledger & Correlated Logs (`src/types/logs.ts`)
- `RunAggregatedSession`: run_id, profile_id, profile_name, status, start_time, duration_sec, files_transferred, audit_status, entries.
- `LogCluster`: id, isCluster, count, entry, expanded, clusterEntries.
- `DaemonLogFilter`: level, run_id, job_id, search, limit.

### 4. Help, Safety Policies & Diagnostics (`src/types/help.ts`)
- `SafetyPolicy`: id, title, tag, contexts, description, action.
- `CliFlagDoc`: flag, name, category, backends, contexts, description, caveat, example, docUrl.
- `SelfDiagnosticResult`: status, timestamp, checks.

### 5. Fleet Settings & MCP Contracts (`src/types/settings.ts`)
- `GlobalFleetSettings`: max_delete_percent, default_schedule, concurrency, bandwidth_limit, enable_canary, polling_interval_ms, notification_webhook.
- `McpStatusResponse`: transport, pid, endpoint, read_only, tool_count, tools.

---

## Files

### New Files to Create
- `src/types/explorer.ts`, `src/types/dedupe.ts`, `src/types/help.ts`, `src/types/settings.ts`
- `src/components/ModalFilePreview.vue`
- `src/components/ModalCompare.vue`
- `src/components/ModalExplorerMkdir.vue`
- `src/components/ModalExplorerMove.vue`
- `src/components/ModalConfirm.vue`
- `src/components/ExplorerContextMenu.vue`
- `src/views/ViewTelemetryLedger.vue`
- `src/components/ModalDaemonLogs.vue`
- `src/components/ModalDedupe.vue`
- `src/components/ModalRemoteWizard.vue`
- `src/components/HelpDrawer.vue`
- `src/components/ModalSystemHealth.vue`
- `src/components/ModalGlobalSettings.vue`
- `src/components/ModalMcp.vue`
- `src/data/help-policies.ts`
- `src/data/help-flags.ts`
- `src/stores/dedupe.ts`
- `src/stores/settings.ts`

### Files to Modify
- `src/views/ViewExplorer.vue`
- `src/views/ViewTelemetry.vue`
- `src/views/ViewRemotes.vue`
- `src/components/LogDock.vue`
- `src/components/AppHeader.vue`
- `src/App.vue`
- `src/stores/explorer.ts`

---

## Testing
- Automated Typecheck & Build: `cd dashboard/frontend && npm run type-check && npm run build`
- LOC guidelines: `./scripts/check.sh loc`
- Full test suite: `./scripts/check.sh all`

---

## Implementation Order
- Phase 1: Dual Explorer File Operations, Previews & Context Menus
- Phase 2: Unified Run-Aggregated Ledger & Correlated Daemon Logs
- Phase 3: Remote Deduplication Workspace & Remote Creation Wizard
- Phase 4: Slide-over Help & Diagnostics Drawer
- Phase 5: System Health, Fleet Settings, and MCP Inspector Modals
