# Implementation Plan: Scheduler Active Jobs Race Condition & Test Teardown

## Proposed Changes

Eliminate race condition in `Scheduler.runJob()` where `item.ErrCh` is signaled before `s.activeJobs[item.ProfileID]` is deleted under mutex lock. This caused rapid sequential `TriggerSync` calls to fail intermittently with `ErrSyncInProgress` ("synchronization is already in progress for this profile"), particularly under `-shuffle=on` or heavy concurrency. In addition, ensure all test schedulers properly invoke `defer scheduler.Stop()` to prevent background worker leaks across test runs.

### Affected Components & Packages

- `internal/profiles/scheduler.go`:
  - Reorder `runJob` to remove `item.ProfileID` from `s.activeJobs` and reset worker state *before* signaling `item.ErrCh <- err`.
- `internal/mcp/tools_write_test.go`:
  - Add missing `defer scheduler.Stop()` in `TestWriteTools_TriggerSyncConfirmation` and `TestWriteTools_TriggerSyncDiagnostics`.
- `internal/profiles/scheduler_test.go`:
  - Add missing `defer sched.Stop()` in tests starting a scheduler without teardown.

### Technical Strategy & Data Flow

- In `runJob`:
  ```go
  err := s.executeJob(ctx, item)

  s.mu.Lock()
  delete(s.activeJobs, item.ProfileID)
  if w, ok := s.workers[item.ProfileID]; ok {
      w.isSyncing.Store(false)
  }
  s.signalDispatcherLocked()
  s.mu.Unlock()

  if item.ErrCh != nil {
      item.ErrCh <- err
  }
  ```
- By moving the locked state de-registration prior to channel notification, any synchronous caller waiting on `TriggerSyncWithOptions` (`<-item.ErrCh`) is strictly guaranteed to observe the idle state upon return.

### Safety & Error Handling

- Mutex lock scope is unchanged; deadlock-free.
- Unbuffered/buffered channel behavior remains identical.
- Thread-safe and race-detector clean.

## Verification Plan

- `go test -race -shuffle=on -count=20 ./internal/mcp -run TestWriteTools_TriggerSyncConfirmation`
- `go test -race -shuffle=1795249090223818792 -count=5 ./internal/mcp`
- `go test -race -count=1 ./internal/profiles/... ./internal/mcp/...`
- `./scripts/check.sh all`





