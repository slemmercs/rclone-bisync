#!/usr/bin/env bash
# ==============================================================================
# Bisync Hub Live Development Launcher (Vite HMR + Backend Proxy)
# ==============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

VITE_PORT="${VITE_PORT:-5173}"
BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8080}"
START_SANDBOX=false

print_usage() {
    cat <<EOF
Usage: ./scripts/dev.sh [OPTIONS]

Launch the Vite live frontend development server with Hot Module Replacement (HMR),
reverse-proxying API calls to the Bisync Hub Go backend.

Options:
  -h, --help               Show this help documentation and exit
  -s, --sandbox            Launch full ephemeral sandbox backend daemon automatically
  -b, --backend-url <url>  Target backend daemon URL (default: http://127.0.0.1:8080)
  -p, --port <port>        Vite dev server port (default: 5173)

Examples:
  ./scripts/dev.sh                     # Connect to running Go backend on :8080
  ./scripts/dev.sh --sandbox           # Launch isolated sandbox backend + Vite HMR
  ./scripts/dev.sh -b http://127.0.0.1:8085 -p 3000
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            print_usage
            exit 0
            ;;
        -s|--sandbox)
            START_SANDBOX=true
            shift
            ;;
        -b|--backend-url)
            BACKEND_URL="$2"
            shift 2
            ;;
        -p|--port)
            VITE_PORT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            print_usage
            exit 2
            ;;
    esac
done

if [[ "$START_SANDBOX" == "true" ]]; then
    exec "$REPO_ROOT/scripts/run-sandbox.sh" --dev --vite-port "$VITE_PORT"
fi

# Standard dev mode: verify node & npm
if ! command -v npm >/dev/null 2>&1; then
    echo "❌ Node.js and npm are required for live frontend development."
    exit 1
fi

if [[ ! -d "$REPO_ROOT/dashboard/frontend/node_modules" ]]; then
    echo "📦 Installing frontend dependencies..."
    (cd "$REPO_ROOT/dashboard/frontend" && npm install)
fi

echo "=================================================================="
echo "⚡ Bisync Hub Frontend Dev Server (HMR)"
echo "   Vite Dev Server:  http://127.0.0.1:$VITE_PORT"
echo "   Backend Target:   $BACKEND_URL"
echo ""
echo "   Hint: Run './scripts/dev.sh --sandbox' to spin up an isolated"
echo "         Go backend with seeded test data automatically!"
echo "=================================================================="

cd "$REPO_ROOT/dashboard/frontend"
export VITE_PORT="$VITE_PORT"
export VITE_BACKEND_URL="$BACKEND_URL"
exec npm run dev
