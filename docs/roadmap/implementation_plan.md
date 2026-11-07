# Implementation Plan: Explorer Ergonomics, PDF Preview, Config Encryption Fix, and Header Layout Optimization

[Overview]
Resolve operational ergonomics, safety barriers, and layout issues identified in Milestone 1 testing:
1. Fix Rclone configuration password encryption failure by replacing quoted shell snippets with a clean, temporary POSIX helper executable that parses without fs.SpaceSepList quoting errors.
2. Filter file preview icons so unsupported binary formats (.docx, .xlsx, .pptx, archives, executables) do not display preview buttons or dump raw binary/zip text, and enable native inline PDF preview via an iframe streaming endpoint.
3. Eliminate header overcrowding by removing the bandwidth speed pill from the Tier 1 context bar and integrating Bandwidth Throttling controls into the Global Settings modal (Engine & Polling tab).
4. Remove the redundant standalone Bandwidth & Active Transfers telemetry card, unifying active transfer indicators into the existing synchronization progress monitor.
5. Enhance File Explorer with a dedicated "Upload" button (file input trigger) and a desktop-class right-click and hover "..." context menu hosting file operations (Preview, Download, Copy to opposing pane, Delete).

[Types]
Type system changes:
1. FilePreviewResponse (internal/models/preview.go):
   - Add IsPDF bool `json:"is_pdf,omitempty"` to explicitly flag PDF documents for native in-browser viewing.
2. FileUploadResponse (internal/models/explorer.go):
   - Add FileUploadResponse struct for multipart upload confirmations.

[Files]
File modifications:
1. internal/services/config_encryption_service.go:
   - Replace inlined shell string with temporary executable script in os.TempDir() to prevent fs.SpaceSepList parsing errors in rclone.
2. internal/services/explorer_preview.go:
   - Fix detectMimeType and isBinaryData so openxmlformats (.docx, .xlsx, .pptx) are identified as binary.
   - Detect PDF files and return IsPDF: true with MimeType: "application/pdf".
3. internal/server/handlers_preview.go:
   - Support inline=true query parameter on GET /api/explorer/download for PDF iframe embedding.
4. internal/server/handlers_explorer.go:
   - Add POST /api/explorer/upload handling multipart file uploads with path jail confinement.
5. dashboard/frontend/partials/modals/modal_file_preview.html:
   - Render iframe for PDF preview when is_pdf is true.
6. dashboard/frontend/partials/modals/modal_global_settings.html:
   - Add Bandwidth Throttling controls (presets, custom rate, status) under Engine & Polling tab.
7. dashboard/frontend/partials/components/component_header.html & dashboard/frontend/index.html:
   - Remove bandwidth widget from Tier 1 context bar.
8. dashboard/frontend/partials/view_telemetry.html:
   - Remove redundant standalone Bandwidth card.
9. dashboard/frontend/partials/view_explorer.html:
   - Add Upload button.
   - Restrict Preview button to isItemPreviewable(item).
   - Add hover Download button and "..." action button + right-click context menu.
10. dashboard/frontend/js/explorer_actions.js:
    - Implement isItemPreviewable, openContextMenu, closeContextMenu, and uploadFiles.

[Functions]
Function modifications:
1. createPasswordHelperScript() (string, error) in internal/services/config_encryption_service.go:
   - Writes a temporary POSIX shell script in os.TempDir() that inspects RCLONE_PASSWORD_CHANGE and echoes $NEW_PASS or $OLD_PASS.
2. isItemPreviewable(item) in dashboard/frontend/js/explorer_actions.js:
   - Returns true only for text, code, markdown, image, and PDF files.
3. handleUploadFile in internal/server/handlers_explorer.go:
   - Handles POST /api/explorer/upload with path jail confinement.

[Classes]
Frontend modules:
1. window.ExplorerActionsModule:
   - Extends explorer state with contextMenu: { open: false, x: 0, y: 0, paneKey: null, item: null }.
   - Adds triggerUpload(paneKey) and uploadFiles(paneKey, files).

[Dependencies]
No new external Go module or npm dependencies. Uses standard library and existing Rclone RC client.

[Testing]
Testing approach:
1. Unit tests:
   - internal/services/config_encryption_service_test.go: verify helper script generation and password set/change/remove.
   - internal/services/explorer_preview_test.go: verify .docx/.xlsx are recognized as binary and .pdf as is_pdf.
   - internal/server/handlers_explorer_test.go: verify upload and path jail confinement.
2. E2E tests (tests/e2e/dashboard_smoke_test.go):
   - Headless Chrome test verifying context menu, upload button, PDF preview iframe, and clean header layout.
3. ./scripts/check.sh all:
   - Run all 7 quality gates.

[Implementation Order]
1. Config Encryption Fix (helper script in config_encryption_service.go).
2. Preview Fix & PDF Streaming (explorer_preview.go, handlers_preview.go, modal_file_preview.html).
3. Header & Telemetry Layout Cleanup (remove header pill, move bandwidth to settings modal, remove duplicate telemetry card).
4. Explorer Upload & Context Menu (Upload button, isItemPreviewable check, hover Download + "..." context menu).
5. Full Verification, E2E Smoke Tests, and Binary Build.
