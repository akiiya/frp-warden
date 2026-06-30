#!/usr/bin/env bash
# 本地构建单二进制。
# 版本号唯一来源:Git tag(经 git describe 派生);可被环境变量 VERSION 覆盖。
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')}"
VERSION="${VERSION:-dev}"
LDFLAGS="-s -w -X github.com/fengheasia/frp-warden/internal/version.Version=${VERSION}"

# 内嵌前端:构建 web/dist 并同步到 internal/webui/dist 供 go embed。
(cd web && npm ci || npm install)
npm --prefix web run build
go run ./tools/sync-web-dist

mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags "${LDFLAGS}" -o dist/frp-warden ./cmd/frp-warden
echo "built dist/frp-warden (version=${VERSION})"
