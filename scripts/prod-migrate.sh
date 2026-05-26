#!/usr/bin/env bash
# Run golang-migrate against the production database.
#
# Usage:
#   ./scripts/prod-migrate.sh                # default: `up` (apply all pending)
#   ./scripts/prod-migrate.sh up
#   ./scripts/prod-migrate.sh down 1
#   ./scripts/prod-migrate.sh version
#   ./scripts/prod-migrate.sh force <version>   # recovery only

source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
require_env_file

if [[ $# -eq 0 ]]; then
    set -- up
fi

exec "${COMPOSE[@]}" run --rm migrate "$@"
