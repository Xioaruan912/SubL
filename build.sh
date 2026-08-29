#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

cd "$ROOT_DIR/webs"
corepack enable >/dev/null 2>&1 || true
pnpm install --frozen-lockfile
pnpm build

cd "$ROOT_DIR"
GOOS=linux GOARCH=amd64 go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink_linux_amd64 .
GOOS=linux GOARCH=arm64 go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink_linux_arm64 .
