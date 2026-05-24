#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${1:-$ROOT_DIR/data}"
APP_UID="${APP_UID:-1000}"
APP_GID="${APP_GID:-1000}"

mkdir -p "$DATA_DIR"

if chown "$APP_UID:$APP_GID" "$DATA_DIR" 2>/dev/null; then
  :
elif command -v sudo >/dev/null 2>&1; then
  sudo chown -R "$APP_UID:$APP_GID" "$DATA_DIR"
else
  echo "Could not chown $DATA_DIR to $APP_UID:$APP_GID." >&2
  echo "Run: sudo chown -R $APP_UID:$APP_GID $DATA_DIR" >&2
  exit 1
fi

chmod u+rwx "$DATA_DIR"

echo "Fixed permissions for $DATA_DIR"
echo "The invoice-app container runs as $APP_UID:$APP_GID and can now write SQLite files there."
