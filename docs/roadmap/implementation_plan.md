# Implementation Plan: Automated Rclone Config Passphrase Persistence & Daemon Safety

## Overview
When `rclone.conf` is encrypted, the background `rclone-daemon.service` must have access to `RCLONE_CONFIG_PASS` to decrypt the configuration non-interactively upon system boot or service restart. 

This feature implements:
1. **Automatic Secret Sync**: When `rclone.conf` is encrypted via the UI, the encryption passphrase is automatically and atomically persisted to `rc-secrets.env` (`0600` permissions) under `RCLONE_CONFIG_PASS`. When encryption is removed via the UI, `RCLONE_CONFIG_PASS` is automatically purged from `rc-secrets.env`.
2. **Startup & Polling Detection**: On startup and polling cycles, Bisync Hub inspects `rclone.conf` encryption status. If `rclone.conf` is encrypted but `RCLONE_CONFIG_PASS` is missing from both the daemon secrets file and process environment, it sets a warning status.
3. **UI Warning Banner & Assisted Persistence**: When an external encryption mismatch is detected, a prominent warning banner is displayed across the Bisync Hub cockpit alerting the operator that the background daemon will fail on reboot, with an action button to enter and validate the passphrase to save it directly into `rc-secrets.env`.

---

## Proposed Changes

### Affected Components & Packages

- `internal/config/secrets.go` & `internal/config/config.go`:
  - Add atomic `.env` file management utilities: `UpdateSecret(path, key, value string) error` and `RemoveSecret(path, key string) error` with strict POSIX `0600` permissions.
  - Track `SecretsFile` path on the `Config` struct during discovery in `LoadConfig`.

- `internal/models/encryption.go`:
  - Extend `EncryptionStatusResponse` with `HasDaemonPassphrase bool` and `Warning string`.
  - Add `SaveDaemonPassphraseRequest` struct.

- `internal/services/config_encryption_service.go`:
  - Update `GetStatus()` to inspect `os.Getenv("RCLONE_CONFIG_PASS")` and `rc-secrets.env`.
  - In `SetPassword()`: Automatically persist `RCLONE_CONFIG_PASS` to `rc-secrets.env` upon successful encryption change.
  - In `RemovePassword()`: Automatically purge `RCLONE_CONFIG_PASS` from `rc-secrets.env`.
  - Add `SaveDaemonPassphrase(ctx, passphrase)`: Validate passphrase against `rclone` (e.g. running `rclone listremotes`), then update `rc-secrets.env` and in-memory environment.

- `internal/server/handlers_encryption.go` & `internal/server/server.go`:
  - Add `POST /api/config/encryption/save-passphrase` endpoint.
  - Wire route and security context.

- `dashboard/frontend/index.html` & `dashboard/frontend/partials/modals/modal_global_settings.html`:
  - Add top-level warning banner for missing daemon passphrase (`rclone.conf` encrypted but missing in `rc-secrets.env`).
  - Add inline indicator and action in Global Fleet Settings (Security tab) showing whether daemon persistence is active.

- `dashboard/frontend/js/settings.js` & `dashboard/frontend/app.js`:
  - Fetch encryption status during dashboard initialization and polling.
  - Handle submitting daemon passphrase to save to `rc-secrets.env`.

---

## Safety & Error Handling

- **Atomic Writes**: `UpdateSecret` and `RemoveSecret` write to a temporary file (`.tmp.<pid>`) in the same directory and execute `os.Rename` with `0600` file permissions to prevent race conditions or partial reads by systemd.
- **Passphrase Validation**: Before persisting an externally supplied passphrase, verify it successfully decrypts `rclone.conf` using `rclone listremotes` via the ephemeral helper script.
- **Path Confinement**: `rc-secrets.env` path is strictly resolved from configured `BaseDir` or user home directory, never accepted from untrusted user input.
- **Secret Redaction**: Passphrases are excluded from general telemetry, audit detail logs, and error responses.

---

## Verification Plan

- Unit test secret updating, escaping, appending, and removal: `go test -v -count=1 ./internal/config/... -run TestUpdateRemoveSecret`
- Unit test service status reporting and automatic secret sync: `go test -v -count=1 ./internal/services/... -run TestConfigEncryptionService`
- Unit test API handlers for encryption status and save-passphrase: `go test -v -count=1 ./internal/server/... -run TestEncryption`
- LOC & Template check: `./scripts/check-loc.sh 400` and `./scripts/check.sh templates`
- Full test suite: `./scripts/check.sh all`

- Full check suite: `./scripts/check.sh all`
- E2E smoke tests: `go test -v -count=1 ./tests/e2e/...`

