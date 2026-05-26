#!/usr/bin/env bash
# Bootstrap .env.production by copying .env.production.example.
# Safe to re-run — refuses to clobber an existing file.

source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

EXAMPLE="$ROOT/.env.production.example"

if [[ ! -f "$EXAMPLE" ]]; then
    echo "✗ .env.production.example not found at $EXAMPLE" >&2
    exit 1
fi

if [[ -f "$ENV_FILE" ]]; then
    echo "✓ .env.production already exists at $ENV_FILE — leaving it alone."
    echo "  (edit it then run ./scripts/prod-up.sh)"
    exit 0
fi

cp "$EXAMPLE" "$ENV_FILE"
echo "✓ Created .env.production from the example."
echo ""
echo "Next steps:"
echo "  1. Open .env.production in your editor."
echo "  2. Replace every CHANGE_ME value:"
echo "       DB_PASSWORD             — e.g.  openssl rand -base64 24"
echo "       JWT_SECRET              — e.g.  openssl rand -hex 32"
echo "       STREAM_SIGNING_SECRET   — e.g.  openssl rand -hex 32"
echo "     and set your real public values:"
echo "       PUBLIC_BASE_URL  ALLOWED_STREAM_DOMAINS  ALLOWED_CORS_ORIGINS"
echo "  3. ./scripts/prod-up.sh"
echo "  4. ./scripts/prod-migrate.sh"
