# Shared bootstrap for every script in this directory.
#
# Resolves the repo root (one level up from this script), `cd`s into it,
# and exports the docker compose invocation as $COMPOSE so callers can
# just write `$COMPOSE up -d` etc.
#
# Sourced — never executed directly.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

ENV_FILE="$ROOT/.env.production"
COMPOSE_FILE="$ROOT/docker-compose.prod.yml"

if [[ ! -f "$COMPOSE_FILE" ]]; then
    echo "✗ docker-compose.prod.yml not found at repo root ($ROOT)" >&2
    exit 1
fi

# $COMPOSE is the prefix the other scripts prepend to every command.
# Quoting via an array keeps spaces / paths safe.
COMPOSE=(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE")

require_env_file() {
    if [[ ! -f "$ENV_FILE" ]]; then
        echo "✗ .env.production not found at $ENV_FILE" >&2
        echo "  Run ./scripts/prod-init-env.sh first." >&2
        exit 1
    fi
}
