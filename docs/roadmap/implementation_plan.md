# Implementation Plan: Alternative A - Single-Source Declarative Schema Engine for Provider Options

## Proposed Changes

Transition Target Provider Options from hardcoded, duplicated Go/JS/HTML implementations to **Alternative A: Single-Source Declarative Schema Engine**.
Instead of hand-coding UI inputs, backend flag compilers, and frontend CLI previews for each remote type (Google Drive, OneDrive, Amazon S3, Dropbox, etc.), all provider-specific options will be defined declaratively in a unified schema.

### Affected Components & Packages

- `internal/profiles/provider_options.json`: Single source of truth containing metadata for all provider options (flags, types, labels, default values, options).
- `internal/profiles/registry.go`: Go registry embedding `provider_options.json` with lookup utilities and validation.
- `internal/profiles/builder.go`: Generic iteration over schema definitions to synthesize CLI arguments, replacing the hardcoded `if/switch` chain.
- `internal/profiles/builder_test.go`: Comprehensive unit tests verifying schema integrity and parity with existing CLI generation.
- `dashboard/frontend/js/provider_options.js`: Modular frontend registry and Alpine.js helper methods for dynamic options handling.
- `dashboard/frontend/index.html`: Registration of the `provider_options.js` script tag.
- `dashboard/frontend/partials/modals/modal_provider_options.html`: Dynamic Alpine `<template x-for>` rendering for boolean flags and value inputs.
- `dashboard/frontend/js/flags_builder.js`: Generic synthesis of provider flags in live CLI preview.
- `tests/e2e/dashboard_smoke_test.go`: Verification of dynamically rendered provider options across provider types.

### Technical Strategy & Data Flow

- **Declarative Schema Structure**:
  - `provider`: Provider type identifier (e.g., `drive`, `onedrive`, `s3`, `dropbox`).
  - `key`: Option field identifier (e.g., `onedrive_delta`, `drive_skip_shortcuts`, `s3_storage_class`).
  - `flag`: Exact Rclone CLI argument flag (e.g., `--onedrive-delta`, `--drive-skip-shortcuts`, `--s3-storage-class`).
  - `type`: `bool`, `text`, or `select`.
  - `label`: Concise human-readable UI label.
  - `default`: Default value (e.g., `"true"`, `"false"`, `"STANDARD"`).
  - `placeholder`: Optional input placeholder text.
  - `options`: Allowed values for `select` type.
  - `desc`: Inline guidance and tooltip description.
- **Backend CLI Synthesis**:
  - `buildProviderCLIArgs(providerOpts)` iterates through `GetProviderOptions()`, formatting boolean flags (including `=false` when disabled if default was true) and value flags cleanly.
- **Frontend Live Preview & UI Rendering**:
  - Alpine helper `isProviderFieldChecked(field)` provides AST-safe boolean binding without dotted paths (complying with `cmd/check/html_scanner.go`).
  - Live CLI preview iterates over `PROVIDER_OPTIONS_REGISTRY` identically to Go backend.

### Safety & Error Handling

- **Alpine Attribute AST Safety**: Avoid dotted expressions in `:checked` bindings to eliminate Alpine boolean coercion bugs and comply with `./scripts/check.sh templates`.
- **Zero Subprocess / Safe Defaults**: Unset optional values generate zero CLI noise; defaults are strictly respected.
- **LOC Boundaries**: Modularized into small files well under the 400 LOC limit.

## Verification Plan

- `go test -v -count=1 ./internal/profiles/...`: Validate Go registry parsing and CLI flag builder parity.
- `./scripts/check.sh templates`: Verify template AST scanner passes without boolean attribute violations.
- `go test -v -count=1 ./tests/e2e/...`: Verify headless Chrome dynamic rendering across provider switches.
- `./scripts/check.sh all`: Full 7-stage quality gate check.
