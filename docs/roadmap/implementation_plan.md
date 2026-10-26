# Implementation Plan: Harden Headless Chrome E2E Smoke Tests for CI

## Proposed Changes

Harden `TestDashboard_HeadlessChromeSmoke` in `tests/e2e/dashboard_smoke_test.go` against transient timeouts and resource starvation in containerized / virtualized CI environments (such as GitHub Actions `ubuntu-latest`).

### Affected Components & Packages

- `tests/e2e/dashboard_smoke_test.go`:
  - Enhance `chromedp.DefaultExecAllocatorOptions` with container resilience flags (`--disable-dev-shm-usage`, `--disable-software-rasterizer`, `--proxy-server=direct://`, `--disable-background-*`).
  - Scale test context deadline dynamically when running under CI (`CI=true` / `GITHUB_ACTIONS=true`) to 60s while maintaining fast local default (35s).
  - Dump captured browser console logs on initial page navigation failures to provide actionable failure diagnostics instead of an opaque context deadline.

### Technical Strategy & Data Flow

- Explicitly pass flags to the chromedp browser process allocator:
  - `--disable-dev-shm-usage`: prevents crash/hang from 64MB default `/dev/shm` in docker/CI.
  - `--disable-software-rasterizer`: avoids stalling on software GL rasterizer fallback.
  - `--proxy-server=direct://` & `--proxy-bypass-list=*`: bypasses host ambient proxy environment variables for loopback addresses (`127.0.0.1`).
  - Background optimization flags (`disable-background-networking`, `disable-background-timer-throttling`, `disable-backgrounding-occluded-windows`, `disable-ipc-flooding-protection`, `disable-hang-monitor`, `disable-renderer-backgrounding`).
  - `--force-color-profile=srgb`: avoids color-profile lookup stalls.
- Diagnostic observability:
  - If initial `chromedp.Do(ctx, chromedp.Navigate(...))` returns an error, lock and print all collected `consoleErrors` in the failure message.

### Safety & Error Handling

- Changes are scoped strictly to test execution configuration in `tests/e2e/dashboard_smoke_test.go`.
- No application runtime code or API contracts are modified.
- Strict LOC budget check maintained (file remains well under 400 LOC limit).

## Verification Plan

- `go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke` (Local run verification)
- `CI=true go test -v -count=1 ./tests/e2e -run TestDashboard_HeadlessChromeSmoke` (Simulated CI environment verification)
- `./scripts/check.sh test` (Full unit test & statement coverage suite)
- `./scripts/check.sh all` (Pre-flight 8-stage quality gate)








