#!/usr/bin/env bash
# ==============================================================================
# Ephemeral Rclone Bisync Sandbox Environment
# Runs a fully isolated test dashboard and rclone daemon using /tmp.
# Leaves ZERO files on the local filesystem or home directory upon exit.
# ==============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SANDBOX_DIR="$(mktemp -d /tmp/rclone-bisync-sandbox-XXXXXX)"

DASHBOARD_PORT="${DASHBOARD_PORT:-8085}"
MCP_PORT="${MCP_PORT:-8086}"
RC_PORT="${RC_PORT:-5575}"
RC_USER="sandbox"
RC_PASS="SandboxPass99!"

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

# 1. Compile single binary
BIN_PATH="$REPO_ROOT/bin/rclone-bisync"
echo "🔨 Compiling latest rclone-bisync binary..."
mkdir -p "$REPO_ROOT/bin"
(cd "$REPO_ROOT" && go build -o "$BIN_PATH" main.go)

# 2. Provision isolated directory structure
mkdir -p "$SANDBOX_DIR/local/subfolder"
mkdir -p "$SANDBOX_DIR/remote_storage/backup_docs"
mkdir -p "$SANDBOX_DIR/review_sessions"
mkdir -p "$SANDBOX_DIR/review_backups"
mkdir -p "$SANDBOX_DIR/cache"
touch "$SANDBOX_DIR/bisync.log"
export RCLONE_CACHE_DIR="$SANDBOX_DIR/cache"

# 3. Seed mock test data
echo "Local notes for testing parallel instance" > "$SANDBOX_DIR/local/sample_notes.txt"
printf "id,name,value\n1,alpha,100\n2,beta,200\n" > "$SANDBOX_DIR/local/data.csv"
echo "Nested file inside subfolder" > "$SANDBOX_DIR/local/subfolder/nested.txt"
touch "$SANDBOX_DIR/local/RCLONE_TEST"

printf "# Remote markdown file\nStored on sandbox remote storage.\n" > "$SANDBOX_DIR/remote_storage/remote_file.md"
printf '{"archived": true, "version": "1.0"}\n' > "$SANDBOX_DIR/remote_storage/backup_docs/archive_info.json"
touch "$SANDBOX_DIR/remote_storage/RCLONE_TEST"

# 4. Generate isolated rclone config with alias remote
cat <<EOF > "$SANDBOX_DIR/rclone.conf"
[testremote]
type = alias
remote = $SANDBOX_DIR/remote_storage
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

# 7. Generate profiles store
cat <<EOF > "$SANDBOX_DIR/profiles.json"
[
  {
    "id": "prof_sandbox",
    "name": "Sandbox Test Remote",
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
    "version": 1,
    "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "state": {
      "status": "healthy"
    }
  }
]
EOF

# 8. Start Rclone RC Daemon in background supervisor loop (supports live password change reload)
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
for i in {1..20}; do
    if curl -s -u "$RC_USER:$RC_PASS" -X POST "http://127.0.0.1:$RC_PORT/core/version" >/dev/null 2>&1; then
        break
    fi
    sleep 0.2
done

# 9. Start Dashboard
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
echo ""
echo "Press [Ctrl+C] to stop and completely destroy the sandbox."
echo "=================================================================="

wait "$DASH_PID"
