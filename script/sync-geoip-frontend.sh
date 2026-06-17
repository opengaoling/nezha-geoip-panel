#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
FRONTEND_REPO="${FRONTEND_REPO:-https://github.com/opengaoling/nezha-geoip-frontend.git}"
FRONTEND_REF="${FRONTEND_REF:-main}"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

git clone --depth 1 --branch "$FRONTEND_REF" "$FRONTEND_REPO" "$TMP_DIR/frontend"

rm -rf "$ROOT_DIR/cmd/dashboard/admin-dist" "$ROOT_DIR/cmd/dashboard/user-dist"
mkdir -p "$ROOT_DIR/cmd/dashboard"
cp -a "$TMP_DIR/frontend/admin-dist" "$ROOT_DIR/cmd/dashboard/admin-dist"
cp -a "$TMP_DIR/frontend/user-dist" "$ROOT_DIR/cmd/dashboard/user-dist"

