# Implementation Plan

Implement a unified virtual overlay and schema-driven remote configuration engine across Bisync Hub, combining Rclone Hasher checksum acceleration, client-side Crypt remote encryption, and a universal dynamic provider form engine.

This unified implementation merges the capabilities from `feat/hasher-backend-integration` and `feat/dynamic-provider-form` into a single cohesive architecture. By integrating Rclone's `/config/providers` metadata schema, `backend/command` interface, and overlay storage primitives (`hasher` and `crypt`), Bisync Hub will enable high-performance checksum caching for non-hash remotes, transparent zero-knowledge encryption for cloud remotes, and automated schema-driven UI forms for all standard storage providers without hardcoded field definitions.

## Types

### Go Backend Structs (`internal/models/`)

- `internal/models/hasher.go`:
  ```go
  // HasherWrapRequest defines the payload for wrapping an existing remote with Hasher.
  type HasherWrapRequest struct {
      BaseRemote string   `json:"base_remote"`
      HasherName string   `json:"hasher_name"`
      Hashes     []string `json:"hashes,omitempty"`
      MaxAge     string   `json:"max_age,omitempty"`
  }

  // HasherCommandRequest defines the payload for executing Hasher backend commands.
  type HasherCommandRequest struct {
      Remote  string                 `json:"remote"`
      Command string                 `json:"command"`
      Args    []string               `json:"args,omitempty"`
      Opt     map[string]interface{} `json:"opt,omitempty"`
  }

  // HasherCommandResponse defines the response from a Hasher backend command.
  type HasherCommandResponse struct {
      Result interface{} `json:"result,omitempty"`
  }
  ```

- `internal/models/crypt.go`:
  ```go
  // CryptWrapRequest defines the payload for wrapping an existing remote with Crypt encryption.
  type CryptWrapRequest struct {
      BaseRemote              string `json:"base_remote"`
      CryptName               string `json:"crypt_name"`
      Subpath                 string `json:"subpath,omitempty"`
      Password                string `json:"password"`
      Password2               string `json:"password2,omitempty"`
      FilenameEncryption      string `json:"filename_encryption,omitempty"`      // "standard", "obfuscate", "off"
      DirectoryNameEncryption bool   `json:"directory_name_encryption,omitempty"` // default true
  }

  // CryptWrapResponse defines the response after creating a Crypt remote wrapper.
  type CryptWrapResponse struct {
      RemoteName string `json:"remote_name"`
      BaseRemote string `json:"base_remote"`
      Success    bool   `json:"success"`
      Message    string `json:"message"`
  }
  ```

- `internal/models/provider.go`:
  ```go
  // ProviderOption represents a configurable field on an Rclone storage provider.
  type ProviderOption struct {
      Name       string            `json:"Name"`
      Help       string            `json:"Help"`
      Provider   string            `json:"Provider,omitempty"`
      Hide       int               `json:"Hide,omitempty"`
      Type       string            `json:"Type"`
      Default    interface{}       `json:"Default,omitempty"`
      DefaultStr string            `json:"DefaultStr,omitempty"`
      Required   bool              `json:"Required"`
      IsPassword bool              `json:"IsPassword"`
      Sensitive  bool              `json:"Sensitive"`
      Advanced   bool              `json:"Advanced"`
      Exclusive  bool              `json:"Exclusive"`
      Examples   []ProviderExample `json:"Examples,omitempty"`
  }

  // IsHiddenInConfig returns true if the option should be hidden from configuration wizards.
  func (o ProviderOption) IsHiddenInConfig() bool {
      return (o.Hide & 2) != 0
  }

  // CreateRemoteBridgeRequest defines the payload for creating a remote with an optional profile bridge and crypt wrap.
  type CreateRemoteBridgeRequest struct {
      Name                    string                 `json:"name"`
      Type                    string                 `json:"type"`
      Parameters              map[string]interface{} `json:"parameters"`
      Opt                     map[string]interface{} `json:"opt,omitempty"`
      CreateProfile           bool                   `json:"create_profile,omitempty"`
      LocalPath               string                 `json:"local_path,omitempty"`
      RemoteSubpath           string                 `json:"remote_subpath,omitempty"`
      ProfileName             string                 `json:"profile_name,omitempty"`
      Schedule                string                 `json:"schedule,omitempty"`
      EncryptRemote           bool                   `json:"encrypt_remote,omitempty"`
      CryptPassword           string                 `json:"crypt_password,omitempty"`
      CryptPassword2          string                 `json:"crypt_password2,omitempty"`
      CryptFilenameEncryption string                 `json:"crypt_filename_encryption,omitempty"`
  }
  ```

### TypeScript Frontend Contracts (`dashboard/frontend/src/types/`)

- `dashboard/frontend/src/types/hasher.ts`:
  ```ts
  export interface HasherWrapRequest {
    base_remote: string
    hasher_name: string
    hashes?: string[]
    max_age?: string
  }

  export interface HasherCommandRequest {
    remote: string
    command: 'dump' | 'drop' | 'import' | 'stickyimport'
    args?: string[]
    opt?: Record<string, any>
  }

  export interface HasherCommandResponse {
    result?: any
  }
  ```

- `dashboard/frontend/src/types/crypt.ts`:
  ```ts
  export interface CryptWrapRequest {
    base_remote: string
    crypt_name: string
    subpath?: string
    password: string
    password2?: string
    filename_encryption?: 'standard' | 'obfuscate' | 'off'
    directory_name_encryption?: boolean
  }

  export interface CryptWrapResponse {
    remote_name: string
    base_remote: string
    success: boolean
    message: string
  }
  ```

- `dashboard/frontend/src/types/dedupe.ts`:
  Update `ProviderOption` to include `Provider?: string` and `Hide?: number`.
  Update `CreateRemotePayload` to include `opt?: Record<string, any>`, `encrypt_remote?: boolean`, `crypt_password?: string`, `crypt_password2?: string`, and `crypt_filename_encryption?: string`.

## Files

### New Files
- `internal/models/crypt.go`: Models for Crypt remote wrapping requests and responses.
- `internal/models/crypt_test.go`: Unit tests for Crypt model JSON serialization and validation.
- `internal/services/hasher_service.go`: Business logic for Hasher remote wrapping and `POST /backend/command` dispatch.
- `internal/services/hasher_service_test.go`: Unit tests for `HasherService` with mocked Rclone RC server.
- `internal/services/crypt_service.go`: Business logic for Crypt remote wrapping with automated password obscuring.
- `internal/services/crypt_service_test.go`: Unit tests for `CryptService`.
- `internal/server/handlers_virtual_remotes.go`: Route handlers for `/api/remotes/hasher/wrap`, `/api/remotes/hasher/command`, and `/api/remotes/crypt/wrap`.
- `internal/server/handlers_virtual_remotes_test.go`: HTTP handler unit tests.
- `dashboard/frontend/src/types/hasher.ts`: Frontend TypeScript interfaces for Hasher requests/responses.
- `dashboard/frontend/src/types/crypt.ts`: Frontend TypeScript interfaces for Crypt requests/responses.
- `dashboard/frontend/src/components/DynamicProviderForm.vue`: Reusable schema-driven form component rendering provider options, S3 flavor selector, and advanced options toggle.
- `dashboard/frontend/src/components/ModalHasherManager.vue`: Cockpit modal for Hasher cache operations (`dump`, `drop`, `import`).
- `dashboard/frontend/src/components/ModalCryptWrap.vue`: Cockpit modal for wrapping existing remotes with Crypt encryption.

### Existing Files to Modify
- `internal/core/client/remotes.go`: Add `Obscure(ctx, clear)` to call Rclone RC `POST /core/obscure`.
- `internal/core/client/client_test.go`: Add unit test for `Obscure`.
- `internal/services/remote_wizard_service.go`: Update `CategorizeProvider` to categorize `"hasher"` and `"crypt"` as `models.CategoryVirtual`. Support optional Crypt auto-wrapping in `CreateRemote`.
- `internal/services/remote_wizard_service_test.go`: Add test cases for `"hasher"` and `"crypt"` categorization.
- `internal/app/container.go`: Register `HasherService` and `CryptService` into the dependency injection container.
- `internal/server/routes.go`: Register virtual remote HTTP endpoints.
- `internal/mcp/tools_remote.go`: Add MCP tools `bisync_hasher_command` and `bisync_create_crypt_remote`.
- `internal/mcp/tools_remote_test.go`: Add unit tests for Hasher and Crypt MCP tools.
- `dashboard/frontend/src/components/ModalRemoteWizard.vue`: Replace static provider input templates with `<DynamicProviderForm>`. Add "Encrypt remote with Crypt overlay" toggle and credentials in Step 2.
- `dashboard/frontend/src/views/ViewRemotes.vue`: Add "Wrap in Hasher" and "Encrypt (Crypt)" action buttons to remote cards.
- `dashboard/frontend/src/stores/remotes.ts`: Add `wrapRemoteInHasher`, `executeHasherCommand`, and `wrapRemoteInCrypt` actions.
- `deploy/install.sh`: Ensure `$REAL_HOME/.cache/rclone` is created and configured with correct user permissions.
- `deploy/systemd/rclone-daemon.service`: Verify `--cache-dir %h/.cache/rclone` argument.
- `bisync.env.example`: Add documentation on `RCLONE_CACHE_DIR` persistence.

## Functions

### New Functions
- `RcloneClient.BackendCommand(ctx context.Context, fs, command string, args []string, opt map[string]interface{}) (interface{}, error)` in `internal/core/client/backend.go`: Posts command payload to Rclone RC `/backend/command`.
- `RcloneClient.Obscure(ctx context.Context, clear string) (string, error)` in `internal/core/client/remotes.go`: Calls `POST /core/obscure` to produce obscured passphrase strings.
- `HasherService.WrapRemoteInHasher(ctx context.Context, req models.HasherWrapRequest, actor string) (*models.CreateRemoteResult, error)` in `internal/services/hasher_service.go`: Validates base remote and provisions Hasher remote.
- `HasherService.ExecuteHasherCommand(ctx context.Context, req models.HasherCommandRequest) (*models.HasherCommandResponse, error)` in `internal/services/hasher_service.go`: Dispatches backend commands (`dump`, `drop`, `import`) to the Hasher remote.
- `CryptService.WrapRemoteInCrypt(ctx context.Context, req models.CryptWrapRequest, actor string) (*models.CryptWrapResponse, error)` in `internal/services/crypt_service.go`: Validates base remote and provisions Crypt remote with `opt: { obscure: true }`.
- `s.handleHasherWrap(w http.ResponseWriter, r *http.Request)` in `internal/server/handlers_virtual_remotes.go`: HTTP handler for `/api/remotes/hasher/wrap`.
- `s.handleHasherCommand(w http.ResponseWriter, r *http.Request)` in `internal/server/handlers_virtual_remotes.go`: HTTP handler for `/api/remotes/hasher/command`.
- `s.handleCryptWrap(w http.ResponseWriter, r *http.Request)` in `internal/server/handlers_virtual_remotes.go`: HTTP handler for `/api/remotes/crypt/wrap`.
- `s.handleHasherCommandTool(ctx context.Context, req mcp.CallToolRequest)` in `internal/mcp/tools_remote.go`: MCP handler for Hasher cache operations.
- `s.handleCreateCryptRemoteTool(ctx context.Context, req mcp.CallToolRequest)` in `internal/mcp/tools_remote.go`: MCP handler for Crypt remote creation.
- `matchesProviderFlavor(optProvider: string, currentFlavor: string): boolean` in `dashboard/frontend/src/components/DynamicProviderForm.vue`: Parses comma-separated flavor tokens with `!` negation handling.

### Modified Functions
- `CategorizeProvider(pType string) models.ProviderCategory` in `internal/services/remote_wizard_service.go`: Add `"hasher"` to the virtual backends switch case.
- `RemoteWizardService.CreateRemote(ctx context.Context, req models.CreateRemoteBridgeRequest, actor string)` in `internal/services/remote_wizard_service.go`: If `req.EncryptRemote` is true, after successfully provisioning the base remote, immediately invoke `CryptService.WrapRemoteInCrypt` to create `<cleanName>_crypt`, and redirect the sync profile creation to target `<cleanName>_crypt`.
- `handleDeploy()` in `dashboard/frontend/src/components/ModalRemoteWizard.vue`: Collect dynamic form parameters, pass `opt: { obscure: true }` when passwords are set, and handle `encrypt_remote` payload fields.

## Classes

- `HasherService` (`internal/services/hasher_service.go`): Encapsulates remote validation, Hasher parameter synthesis, BoltDB cache maintenance, and backend command execution.
- `CryptService` (`internal/services/crypt_service.go`): Encapsulates remote validation, Crypt parameter synthesis, password obscuring, and overlay remote lifecycle management.
- `remotesStore` (`dashboard/frontend/src/stores/remotes.ts`): State management for storage remotes, extended with Hasher cache state and Crypt wrapping actions.

## Dependencies

- Zero runtime external dependencies: single static binary invariant is maintained.
- Standard Go library + existing PrimeVue / Tailwind in frontend.
- Native Rclone RC API targets: `/config/providers`, `/config/create`, `/backend/command`, `/core/obscure`.

## Testing

- Unit Tests:
  - `internal/core/client/backend_test.go`: Test `BackendCommand` parameter formatting, successful execution, and error handling.
  - `internal/core/client/client_test.go`: Test `Obscure` method.
  - `internal/models/provider_test.go`: Test `ProviderOption` unmarshaling with `Provider`, `Hide`, and `IsHiddenInConfig()`.
  - `internal/models/crypt_test.go`: Test `CryptWrapRequest` and `CryptWrapResponse` JSON marshaling.
  - `internal/services/hasher_service_test.go`: Test `WrapRemoteInHasher` and `ExecuteHasherCommand` against mocked RC server.
  - `internal/services/crypt_service_test.go`: Test `WrapRemoteInCrypt` parameter generation and password obscuring.
  - `internal/server/handlers_virtual_remotes_test.go`: Test HTTP endpoints with valid and invalid payloads.
  - `internal/mcp/tools_remote_test.go`: Test MCP tool calls for Hasher and Crypt.
- Frontend Verification:
  - TypeScript type checking: `npm --prefix dashboard/frontend run type-check`.
  - Frontend production build: `npm --prefix dashboard/frontend run build`.
- Quality Pipeline:
  - Pre-commit check suite: `./scripts/check.sh all` (covering formatting, vetting, race-detector tests, LOC audits, and binary compilation).

## Implementation Order

1. **Branch Consolidation & Git Merge**:
   - Merge `feat/hasher-backend-integration` into the unified working branch (`feat/dynamic-provider-form`).
   - Resolve plan file conflicts by saving this unified plan to `implementation_plan.md` and `docs/roadmap/implementation_plan.md`.
   - Update `task_plan.md` with the unified micro-tasks.
   - Verify Go test suite passes: `go test -count=1 ./internal/...`.

2. **Core RC Client & Crypt Models**:
   - Add `Obscure` method in `internal/core/client/remotes.go`.
   - Create `internal/models/crypt.go` and `internal/models/crypt_test.go`.
   - Update `CreateRemoteBridgeRequest` in `internal/models/provider.go`.
   - Verify: `go test -count=1 ./internal/models/... ./internal/core/client/...`.

3. **Domain Services (`HasherService` and `CryptService`)**:
   - Implement `internal/services/hasher_service.go` and `internal/services/hasher_service_test.go`.
   - Implement `internal/services/crypt_service.go` and `internal/services/crypt_service_test.go`.
   - Update `CategorizeProvider` and `CreateRemote` in `internal/services/remote_wizard_service.go`.
   - Register services in `internal/app/container.go`.
   - Verify: `go test -count=1 ./internal/services/...`.

4. **HTTP Endpoints & MCP Tools**:
   - Create `internal/server/handlers_virtual_remotes.go` and tests in `internal/server/handlers_virtual_remotes_test.go`.
   - Register routes in `internal/server/routes.go`.
   - Add MCP tools `bisync_hasher_command` and `bisync_create_crypt_remote` in `internal/mcp/tools_remote.go`.
   - Verify: `go test -count=1 ./internal/server/... ./internal/mcp/...`.

5. **Deployment Cache Persistence**:
   - Update `deploy/install.sh`, `deploy/systemd/rclone-daemon.service`, and `bisync.env.example` to ensure persistent `$REAL_HOME/.cache/rclone`.
   - Verify: `bash -n deploy/install.sh`.

6. **Frontend TypeScript Contracts & Stores**:
   - Create `dashboard/frontend/src/types/hasher.ts` and `dashboard/frontend/src/types/crypt.ts`.
   - Update `dashboard/frontend/src/types/dedupe.ts`.
   - Add virtual overlay actions in `dashboard/frontend/src/stores/remotes.ts`.
   - Verify: `npm --prefix dashboard/frontend run type-check`.

7. **Dynamic Provider Form Component**:
   - Implement `dashboard/frontend/src/components/DynamicProviderForm.vue` with S3 flavor filtering, hide bitmasking, and preset dropdowns.
   - Verify: `npm --prefix dashboard/frontend run type-check`.

8. **Remote Wizard & Encryption Integration**:
   - Integrate `<DynamicProviderForm>` into `dashboard/frontend/src/components/ModalRemoteWizard.vue`.
   - Add "Encrypt remote with Crypt overlay" toggle and inputs to Step 2 of `ModalRemoteWizard.vue`.
   - Verify: `npm --prefix dashboard/frontend run type-check`.

9. **Remotes List Actions & Virtual Overlay Modals**:
   - Implement `dashboard/frontend/src/components/ModalHasherManager.vue`.
   - Implement `dashboard/frontend/src/components/ModalCryptWrap.vue`.
   - Add "Wrap in Hasher" and "Encrypt (Crypt)" quick actions to remote cards in `dashboard/frontend/src/views/ViewRemotes.vue`.
   - Verify: `npm --prefix dashboard/frontend run type-check && npm --prefix dashboard/frontend run build`.

10. **Full Pre-Flight Quality Check**:
    - Execute `./scripts/check.sh all` to validate all 8 build stages.
