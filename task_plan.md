
# Task Plan: Phase 5 - Systemd & Installer Updates + Native CLI Subcommands

## Goal
Implement unified CLI command structure (`rclone-bisync sync`, `rclone-bisync dashboard`, `rclone-bisync canary`), update systemd user units to target `./rclone-bisync` binary (eliminating `.venv`), streamline `install-custom-rclone.sh`, and update `scripts/check.sh` quality gate to run `go vet`, static checks, secret leak verification, and `go test ./...`.

## Tasks
- [x] Task 1: Core CLI Subcommands & Engine Runner (`main.go`, `internal/engine/runner`, `internal/core/compiler`, `internal/core/classifier`)
  - **Scope**:
    - Port argument-to-RC compiler (`internal/core/compiler`).
    - Port error taxonomy classifier (`internal/core/classifier`).
    - Implement resilient job polling monitor (`internal/core/client/monitor.go`).
    - Implement native Go sync orchestrator (`internal/engine/runner/runner.go`).
    - Wire unified CLI commands (`sync`, `dashboard`, `canary`) in `main.go`.
- [x] Task 2: Systemd User Services Hardening
  - **Scope**:
    - Update `rclone-dashboard.service` to `ExecStart=%h/rclone-bisync/bin/rclone-bisync dashboard --host 127.0.0.1 --port 8080` with `MemoryMax=512M` and `CPUQuota=50%`.
    - Update `bisync-gdrive.service` to `ExecStart=%h/rclone-bisync/bin/rclone-bisync sync --origin=timer`.
    - Update `internal/services/sync_service.go` to execute `bin/rclone-bisync sync` in dynamic transient systemd units.
- [x] Task 3: Streamlined Single-Binary Installer (`install-custom-rclone.sh`)
  - **Scope**:
    - Removed Python virtual environment bootstrap and pip dependency installs.
    - Compiles or installs native single binary `bin/rclone-bisync`.
    - Maintains strict permission hygiene (`chmod 600 rc-secrets.env`).
- [x] Task 4: Upgraded Quality Gate (`scripts/check.sh`)
  - **Scope**:
    - Integrated `go vet ./...` static analysis.
    - Integrated `go test -count=1 ./...` across all Go packages.
    - Python compilation & Pytest suites preserved for backwards compatibility testing.
