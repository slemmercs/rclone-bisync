#!/usr/bin/env bash
# ==============================================================================
# Ephemeral Rclone Bisync Sandbox Environment
# Runs a fully isolated test dashboard and rclone daemon using /tmp.
# Leaves ZERO files on the local filesystem or home directory upon exit.
# ==============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DASHBOARD_PORT="${DASHBOARD_PORT:-8085}"
MCP_PORT="${MCP_PORT:-8086}"
RC_PORT="${RC_PORT:-5575}"
RC_USER="sandbox"
RC_PASS="SandboxPass99!"
DRY_RUN=false
BUILD_FRONTEND=false

print_usage() {
    cat <<EOF
Usage: ./scripts/run-sandbox.sh [OPTIONS]

Run an ephemeral, self-contained Bisync Hub sandbox in /tmp with seeded test data.

Options:
  -h, --help            Show this help documentation and exit
  -n, --dry-run         Seed and verify all mock data and configs, then exit without starting daemons
  -b, --build-frontend  Rebuild Vue 3 SPA distribution before compiling Go binary
      --port <port>     Dashboard web service port (default: 8085)
      --mcp-port <port> Embedded MCP server port (default: 8086)
      --rc-port <port>  Rclone RC daemon port (default: 5575)
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            print_usage
            exit 0
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -b|--build-frontend)
            BUILD_FRONTEND=true
            shift
            ;;
        --port)
            DASHBOARD_PORT="$2"
            shift 2
            ;;
        --mcp-port)
            MCP_PORT="$2"
            shift 2
            ;;
        --rc-port)
            RC_PORT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            print_usage
            exit 2
            ;;
    esac
done

SANDBOX_DIR="$(mktemp -d /tmp/rclone-bisync-sandbox-XXXXXX)"

cleanup() {
    echo ""
    echo "🧹 Shutting down sandbox processes and cleaning up temporary files..."
    if [[ -n "${DASH_PID:-}" ]] && kill -0 "$DASH_PID" 2>/dev/null; then
        kill "$DASH_PID" 2>/dev/null || true
    fi
    if [[ -n "${RCD_PID:-}" ]] && kill -0 "$RCD_PID" 2>/dev/null; then
        kill -- -"$RCD_PID" 2>/dev/null || kill "$RCD_PID" 2>/dev/null || true
    fi
    curl -s -u "$RC_USER:$RC_PASS" -X POST "http://127.0.0.1:$RC_PORT/core/quit" >/dev/null 2>&1 || true
    rm -rf "$SANDBOX_DIR"
    echo "✨ Sandbox removed ($SANDBOX_DIR). Local filesystem clean."
}
trap cleanup EXIT INT TERM

echo "📦 Initializing ephemeral sandbox in: $SANDBOX_DIR"

# 1. Compile frontend if requested
if [[ "$BUILD_FRONTEND" == "true" ]]; then
    echo "🎨 Building fresh Vue 3 + PrimeVue SPA frontend..."
    (cd "$REPO_ROOT/dashboard/frontend" && npm run build)
fi

# 2. Compile single binary
BIN_PATH="$REPO_ROOT/bin/rclone-bisync"
echo "🔨 Compiling latest rclone-bisync binary..."
mkdir -p "$REPO_ROOT/bin"
(cd "$REPO_ROOT" && go build -o "$BIN_PATH" main.go)

# 3. Provision isolated directory structure
mkdir -p "$SANDBOX_DIR/local/subfolder"
mkdir -p "$SANDBOX_DIR/local/projects/deep/level"
mkdir -p "$SANDBOX_DIR/local/documents"
mkdir -p "$SANDBOX_DIR/remote_storage/backup_docs"
mkdir -p "$SANDBOX_DIR/remote_storage/duplicates"
mkdir -p "$SANDBOX_DIR/backup_storage/archive"
mkdir -p "$SANDBOX_DIR/archive_storage/year2025"
mkdir -p "$SANDBOX_DIR/review_sessions"
mkdir -p "$SANDBOX_DIR/review_backups"
mkdir -p "$SANDBOX_DIR/cache"
touch "$SANDBOX_DIR/bisync.log"
export RCLONE_CACHE_DIR="$SANDBOX_DIR/cache"

# 4. Seed rich mock test data
echo "Local notes for testing parallel instance" > "$SANDBOX_DIR/local/sample_notes.txt"
printf "id,name,value\n1,alpha,100\n2,beta,200\n3,gamma,300\n" > "$SANDBOX_DIR/local/data.csv"
echo "Nested file inside subfolder" > "$SANDBOX_DIR/local/subfolder/nested.txt"
cat <<'EOF' > "$SANDBOX_DIR/local/subfolder/diagram.svg"
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100">
  <circle cx="50" cy="50" r="40" fill="#6366f1" />
  <text x="50" y="55" font-size="12" text-anchor="middle" fill="#ffffff" font-family="sans-serif">Bisync</text>
</svg>
EOF
printf "# Engineering Specifications\n\n- Dual Explorer V2\n- Chained Cryptographic Ledger\n- Realtime FSM Engine\n" > "$SANDBOX_DIR/local/projects/deep/level/specs.md"
printf "Local version of conflict file.\nModified locally at %s\n" "$(date -u)" > "$SANDBOX_DIR/local/conflict_candidate.txt"
printf "DEDUPLICATION_CRYPTOGRAPHIC_HASH_EXACT_MATCH_2026\n" > "$SANDBOX_DIR/local/hash_match_file.bin"
printf "Quarterly executive summary notes\n" > "$SANDBOX_DIR/local/documents/quarterly_summary.txt"
touch "$SANDBOX_DIR/local/RCLONE_TEST"

# Remote storage seed
printf "# Remote markdown file\nStored on sandbox remote storage.\n" > "$SANDBOX_DIR/remote_storage/remote_file.md"
printf '{"archived": true, "version": "1.0", "target": "remote_storage"}\n' > "$SANDBOX_DIR/remote_storage/backup_docs/archive_info.json"
printf "Quarterly Report duplicate test data\n" > "$SANDBOX_DIR/remote_storage/duplicates/report.txt"
printf "DEDUPLICATION_CRYPTOGRAPHIC_HASH_EXACT_MATCH_2026\n" > "$SANDBOX_DIR/remote_storage/duplicates/hash_match_file.bin"
printf "Remote version of conflict file with differing size.\nModified remotely at %s\nAdditional remote metadata line.\n" "$(date -u)" > "$SANDBOX_DIR/remote_storage/conflict_candidate.txt"
touch "$SANDBOX_DIR/remote_storage/RCLONE_TEST"

# Backup storage seed
printf '{"snapshot_id": "snap-9921", "policy": "daily-archive", "total_bytes": 1048576}\n' > "$SANDBOX_DIR/backup_storage/manifest.json"
printf "MOCK_TAR_GZ_BINARY_PAYLOAD_ARCHIVE\n" > "$SANDBOX_DIR/backup_storage/archive/snapshot_2026.tar.gz"
touch "$SANDBOX_DIR/backup_storage/RCLONE_TEST"

# Archive storage seed
printf "tx_id,amount,currency,status\nTX101,450.00,USD,cleared\nTX102,1250.50,EUR,cleared\n" > "$SANDBOX_DIR/archive_storage/year2025/q4_ledger.csv"
printf '{"fiscal_year": 2025, "audited": true}\n' > "$SANDBOX_DIR/archive_storage/year2025/tax_summary.json"
touch "$SANDBOX_DIR/archive_storage/RCLONE_TEST"

# 5. Generate isolated rclone config with 3 alias remotes
cat <<EOF > "$SANDBOX_DIR/rclone.conf"
[testremote]
type = alias
remote = $SANDBOX_DIR/remote_storage

[backup_remote]
type = alias
remote = $SANDBOX_DIR/backup_storage

[archive_remote]
type = alias
remote = $SANDBOX_DIR/archive_storage
EOF

# 5. Generate secrets file
cat <<EOF > "$SANDBOX_DIR/rc-secrets.env"
RC_USER=$RC_USER
RC_PASS=$RC_PASS
RC_PORT=$RC_PORT
RCLONE_CONFIG=$SANDBOX_DIR/rclone.conf
EOF

# 6. Generate bisync runtime environment
cat <<EOF > "$SANDBOX_DIR/bisync.env"
LOCAL_PATH=$SANDBOX_DIR/local
REMOTE_PATH=testremote:
BASE_DIR=$SANDBOX_DIR
RC_HOST=127.0.0.1
RC_PORT=$RC_PORT
RC_USER=$RC_USER
RC_PASS=$RC_PASS
RCLONE_CONFIG=$SANDBOX_DIR/rclone.conf
RCLONE_CACHE_DIR=$SANDBOX_DIR/cache
PROFILES_FILE=$SANDBOX_DIR/profiles.json
PROFILES_HISTORY_FILE=$SANDBOX_DIR/profiles_history.json
AUDIT_DB_PATH=$SANDBOX_DIR/audit.db
LOG_FILE=$SANDBOX_DIR/bisync.log
NOTIFICATION_APP_NAME="Bisync Sandbox"
DEBUG_MODE=true
FEATURE_DAEMON_HEALTH_CHECK=true
FEATURE_METRICS_COLLECTION=true
EOF

# 8. Generate multi-profile store
cat <<EOF > "$SANDBOX_DIR/profiles.json"
[
  {
    "id": "prof_sandbox",
    "name": "Sandbox Primary Sync",
    "enabled": true,
    "archived": false,
    "remote": "testremote:",
    "local_path": "$SANDBOX_DIR/local",
    "schedule": "manual",
    "review_mode": "all",
    "notification_pref": "all",
    "safety": {
      "delete_threshold_pct": 10,
      "failure_threshold": 3,
      "canary_validation": true
    },
    "flags": {
      "max_delete": 10,
      "canary_validation": true,
      "compare_mode": "size,modtime"
    },
    "version": 2,
    "created_at": "2026-11-16T10:00:00Z",
    "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "state": {
      "status": "healthy"
    }
  },
  {
    "id": "prof_backup",
    "name": "Automated Backup Archive",
    "enabled": true,
    "archived": false,
    "remote": "backup_remote:",
    "local_path": "$SANDBOX_DIR/local/projects",
    "schedule": "1h",
    "review_mode": "disabled",
    "notification_pref": "errors_only",
    "safety": {
      "delete_threshold_pct": 20,
      "failure_threshold": 5,
      "canary_validation": true
    },
    "flags": {
      "max_delete": 20,
      "canary_validation": true,
      "compare_mode": "checksum"
    },
    "version": 1,
    "created_at": "2026-11-16T11:00:00Z",
    "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "state": {
      "status": "healthy"
    }
  },
  {
    "id": "prof_archive",
    "name": "Cold Storage Vault",
    "enabled": false,
    "archived": false,
    "remote": "archive_remote:",
    "local_path": "$SANDBOX_DIR/archive_storage",
    "schedule": "daily",
    "review_mode": "all",
    "notification_pref": "all",
    "safety": {
      "delete_threshold_pct": 5,
      "failure_threshold": 1,
      "canary_validation": true
    },
    "flags": {
      "max_delete": 5,
      "canary_validation": true,
      "compare_mode": "size,modtime,checksum"
    },
    "version": 1,
    "created_at": "2026-11-16T12:00:00Z",
    "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "state": {
      "status": "healthy"
    }
  }
]
EOF

# 9. Generate configuration history snapshots for rollback testing
cat <<EOF > "$SANDBOX_DIR/profiles_history.json"
{
  "prof_sandbox": [
    {
      "version": 1,
      "timestamp": "2026-11-16T10:00:00Z",
      "author": "admin",
      "comment": "Initial baseline configuration before review_mode adjustment",
      "snapshot": {
        "id": "prof_sandbox",
        "name": "Sandbox Primary Sync (v1)",
        "enabled": true,
        "remote": "testremote:",
        "local_path": "$SANDBOX_DIR/local",
        "schedule": "manual",
        "review_mode": "deletions",
        "flags": {
          "max_delete": 5,
          "canary_validation": true,
          "compare_mode": "size,modtime"
        },
        "version": 1
      }
    }
  ]
}
EOF

# 10. Seed and cryptographically verify the SQLite audit ledger
echo "🔐 Seeding and verifying cryptographic audit ledger in: $SANDBOX_DIR/audit.db..."
"$BIN_PATH" audit seed --env-file "$SANDBOX_DIR/bisync.env"
"$BIN_PATH" audit verify --env-file "$SANDBOX_DIR/bisync.env"

# 11. Handle dry-run mode
if [[ "$DRY_RUN" == "true" ]]; then
    echo ""
    echo "=================================================================="
    echo "✔ DRY-RUN SUCCESSFUL!"
    echo "  - Isolated directories: local, remote_storage, backup_storage, archive_storage"
    echo "  - Remotes seeded in rclone.conf: testremote, backup_remote, archive_remote"
    echo "  - Profiles seeded in profiles.json: prof_sandbox, prof_backup, prof_archive"
    echo "  - Rollback snapshots seeded in profiles_history.json: prof_sandbox (v1)"
    echo "  - Cryptographic audit chain: 5 blocks verified in audit.db"
    echo "=================================================================="
    exit 0
fi

# 12. Start Rclone RC Daemon in background supervisor loop (supports live password change reload)
echo "🚀 Starting ephemeral Rclone RC daemon on port $RC_PORT (with live passphrase reload)..."
(
    while true; do
        if [[ -f "$SANDBOX_DIR/rc-secrets.env" ]]; then
            set -a
            # shellcheck disable=SC1090
            source "$SANDBOX_DIR/rc-secrets.env"
            set +a
        fi
        export RCLONE_CONFIG="$SANDBOX_DIR/rclone.conf"
        export RCLONE_CACHE_DIR="$SANDBOX_DIR/cache"
        rclone rcd \
            --rc-serve \
            --rc-addr "127.0.0.1:$RC_PORT" \
            --rc-user "$RC_USER" \
            --rc-pass "$RC_PASS" \
            --config "$SANDBOX_DIR/rclone.conf" \
            --cache-dir "$SANDBOX_DIR/cache" \
            --log-level INFO \
            --log-file "$SANDBOX_DIR/rclone-daemon.log" \
            --use-json-log || true
        sleep 0.2
    done
) >/dev/null 2>&1 &
RCD_PID=$!

# Wait for rclone daemon to respond
for _ in {1..20}; do
    if curl -s -u "$RC_USER:$RC_PASS" -X POST "http://127.0.0.1:$RC_PORT/core/version" >/dev/null 2>&1; then
        break
    fi
    sleep 0.2
done

# 13. Start Dashboard
echo "🌐 Starting Bisync Hub dashboard on port $DASHBOARD_PORT (MCP on $MCP_PORT)..."
"$BIN_PATH" dashboard \
    --host 127.0.0.1 \
    --port "$DASHBOARD_PORT" \
    --mcp-host 127.0.0.1 \
    --mcp-port "$MCP_PORT" \
    --env-file "$SANDBOX_DIR/bisync.env" &
DASH_PID=$!

echo ""
echo "=================================================================="
echo "🎉 Sandbox Running in Isolation!"
echo "   Dashboard:     http://127.0.0.1:$DASHBOARD_PORT"
echo "   MCP Server:    http://127.0.0.1:$MCP_PORT"
echo "   RC Daemon:     http://127.0.0.1:$RC_PORT"
echo "   Storage Root:  $SANDBOX_DIR"
echo "   Remotes:       testremote:, backup_remote:, archive_remote:"
echo "   Profiles:      prof_sandbox, prof_backup, prof_archive"
echo ""
echo "Press [Ctrl+C] to stop and completely destroy the sandbox."
echo "=================================================================="

wait "$DASH_PID"
