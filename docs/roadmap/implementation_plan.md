# Implementation Plan: Storage Remote Delete Safeguard Modal & UI Cascade Warning

## Proposed Changes

Replace the crude native browser `window.confirm()` popup in `ViewRemotes.vue` with an accessible, themed Cockpit safeguard modal (`ModalRemoteDelete.vue`) adhering to the Agnoster Carbon / Electric Blue design system.

### 1. New Component: `ModalRemoteDelete.vue`
- Created under `dashboard/frontend/src/components/ModalRemoteDelete.vue`.
- PrimeVue `Dialog` with `modal` overlay and styling aligned with `ModalProfileDelete.vue`.
- Header:
  - Danger badge with `pi pi-trash` icon.
  - Title: "Delete Storage Remote" with subtitle "Irreversible configuration removal".
- Target Remote Details:
  - High-density monospace card showing remote name, provider type, and target endpoint.
- Cascade Warning Banner:
  - Visual amber warning card triggered whenever `descendants.length > 0`.
  - Lists each dependent child overlay with styled pills indicating name, type (`crypt`, `hasher`, `alias`), and parent link.
  - Explains the blast radius: downstream overlays will become broken or orphaned.
- Crypt Key Safeguard Banner:
  - Distinct warning when target remote is of type `crypt`, alerting the operator that cloud ciphertext remains unreadable unless decrypted prior to key deletion.
- Error Handling & Asynchronous Execution:
  - Integrated with `remotesStore.deleteRemote(name)`.
  - Disables actions and displays loading spinner during deletion.
  - Displays formatted error banner if backend rejection occurs.

### 2. View Integration: `ViewRemotes.vue`
- Retire `window.confirm()` in `confirmDelete(name: string)`.
- Replace with reactive modal state (`showDeleteModal`, `remoteToDelete`, `deleteDescendants`).
- Mount `<ModalRemoteDelete>` in the template.

---

### Affected Components & Packages

- `dashboard/frontend/src/components/ModalRemoteDelete.vue`: New safeguard modal component.
- `dashboard/frontend/src/views/ViewRemotes.vue`: Integration with remote cards and overlay tree deletion triggers.

---

### Safety & Error Handling

- **Zero Accidental Deletions**: Deletion requires explicit modal confirmation and cannot be triggered by keyboard mis-press.
- **Blast Radius Transparency**: Operators are explicitly shown all dependent downstream overlays before confirming deletion.
- **Canary & Jail Protection**: Backend continues to enforce blast-radius checks and profile-link safeguards.

---

## Verification Plan

- Component type checking: `npm --prefix dashboard/frontend run type-check`
- Production frontend bundling: `npm --prefix dashboard/frontend run build`
- Full project verification: `./scripts/check.sh all`
- UI Sandbox verification:
  - Trigger deletion on `testremote` (which has `testremote_crypt` and `testremote_crypt_hasher`).
  - Verify that the modal renders the cascade warning with dependent overlays in dark mode UI style without invoking native browser dialogs.
- Frontend production bundle build: `npm --prefix dashboard/frontend run build`
- Visual & functional validation in sandbox:
  - Verify `backup_remote` displays `backup_remote_crypt` and `backup_remote_hasher` as vertical tree children.
  - Verify `testremote` displays `testremote_hasher` and `testremote_crypt`.
  - Verify `ACTIVE` pill highlights the exact active profile target.
  - Verify `Test`, `Dedupe`, and `Delete` operate correctly on individual overlay rows.
  - Verify `./scripts/check.sh all` passes cleanly across all 7 stages.


