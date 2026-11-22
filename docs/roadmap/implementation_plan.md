# Implementation Plan: Clustered Remote Storage Families with Vertical Overlay Pipeline Trees

## Proposed Changes

Transition the Storage Remotes Cockpit (`ViewRemotes.vue` & `RemoteCard.vue`) from a flat, fragmented 2-column card grid to a hierarchical **Storage Family Cluster** model. Base storage remotes serve as root cards with real storage allocation metrics, while virtual middleware overlays (`crypt`, `hasher`, `chunker`, `alias`) render inside their parent card as a clean, vertical indented pipeline tree.

### 1. Dependency Graph & Topology Builder (`dashboard/frontend/src/views/ViewRemotes.vue`)
- Parse Rclone remotes into a normalized hierarchy:
  - `rootRemotes`: Base storage remotes (local drives, S3, Google Drive, B2, OneDrive, SFTP, etc., or alias pointing to filesystem path).
  - `childrenByParent`: Map each parent remote to its direct downstream child overlays (`childrenByParent[parentName] -> Remote[]`).
  - `compositeOrOrphanRemotes`: Remotes with multi-upstreams (`union`, `combine`) or whose declared upstream remote does not exist.
  - Cycle Guard: Maintain a `visited` set during tree traversal to prevent infinite recursion if invalid configurations contain circular references (`A -> B -> A`).

### 2. Precise Active Profile Endpoint Matching
- In `RemoteCard.vue` and child overlay rows:
  - Compare the active sync profile's remote endpoint against the exact remote name.
  - Display the `ACTIVE` / `PROFILE TARGET` pill on the exact node receiving sync traffic (root or specific overlay), eliminating ambiguity.

### 3. Vertical Overlay Pipeline Tree (`dashboard/frontend/src/components/remotes/RemoteOverlayTree.vue`)
- Replace horizontal arrow wireframes with a compact, readable vertical indented tree (`├─`, `└─`).
- Every overlay node displays:
  - Type icon (🛡️ Crypt, ⚡ Hasher, 🔗 Alias) & name
  - Upstream reference subtitle: `wraps <upstream_remote>`
  - Status indicator (`CONNECTED` / `ERROR`)
  - Target badge if selected by active sync profile (`ACTIVE`)
  - Compact action controls:
    - `Test`: validate connectivity through this overlay layer.
    - `Dedupe`: execute deduplication scoped to this overlay.
    - `+ Wrap`: contextual wrap trigger (e.g. allow Hasher on top of Crypt; prevent Hasher on Hasher).
    - `Delete`: delete overlay with confirmation.

### 4. Root Card Modernization & Cascade Delete Safeguards (`RemoteCard.vue`)
- Relabel root remote target: Change ambiguous `Wraps:` to `Target:` when pointing to filesystem paths or cloud buckets.
- Add `+ Add Overlay` action dropdown on the base card allowing rapid addition of Crypt or Hasher overlays.
- Cascade Delete Warning: When deleting a base remote that has child overlays, warn the user that child overlays (`backup_remote_crypt`, `backup_remote_hasher`) depend on it.

### 5. Composite & Unlinked Remotes Section (`ViewRemotes.vue`)
- Multi-upstream backends (`union`, `combine`) and orphaned remotes render in a distinct "Composite & Unlinked Remotes" section below the primary storage families.

---

### Affected Components & Packages

- `dashboard/frontend/src/views/ViewRemotes.vue`: Build hierarchical graph (`rootRemotes`, `childrenByParent`, `compositeOrOrphans`), render family clusters and composite section.
- `dashboard/frontend/src/components/remotes/RemoteCard.vue`: Modernize root card, integrate overlay tree, add `Target:` label and cascade deletion warnings.
- `dashboard/frontend/src/components/remotes/RemoteOverlayTree.vue` (New): Dedicated cohesive component rendering vertical indented tree with full per-overlay action suites.

---

### Safety & Error Handling

- **Cycle Detection**: Enforce maximum depth and cycle tracking (`visited: Set<string>`) so broken configs cannot crash the browser.
- **Cascade Warning**: Explicit modal/confirm prompt enumerating dependent overlay names before permitting base remote deletion.
- **Zero Backend Changes Needed**: Leverages existing enriched telemetry (`underlying_remote`, `underlying_path`, `type`, `about`) provided by `TelemetryService`.

---

## Verification Plan

- Frontend type checking: `npm --prefix dashboard/frontend run type-check`
- Frontend production bundle build: `npm --prefix dashboard/frontend run build`
- Visual & functional validation in sandbox:
  - Verify `backup_remote` displays `backup_remote_crypt` and `backup_remote_hasher` as vertical tree children.
  - Verify `testremote` displays `testremote_hasher` and `testremote_crypt`.
  - Verify `ACTIVE` pill highlights the exact active profile target.
  - Verify `Test`, `Dedupe`, and `Delete` operate correctly on individual overlay rows.
  - Verify `./scripts/check.sh all` passes cleanly across all 7 stages.


