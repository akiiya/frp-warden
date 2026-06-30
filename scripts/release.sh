#!/usr/bin/env bash
# 发布前完整构建 + 多平台打包(本脚本不真正发版)。
# 发版动作 = 推送一个新的 vX.Y.Z tag,由 GitHub Actions 触发 release.yml 完成 Release 发布。
# 版本号唯一来源:Git tag(经 git describe 派生);CI 发布时通过环境变量 VERSION 注入 tag 去前缀值。
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')}"
VERSION="${VERSION:-dev}"
LDFLAGS="-s -w -X github.com/fengheasia/frp-warden/internal/version.Version=${VERSION}"

# 随产物附带的文件(按项目实际调整):
EXTRA_FILES=( README.md LICENSE )

go vet ./...
go test ./...

# 内嵌前端:构建 web/dist 并同步到 internal/webui/dist 供 go embed。
(cd web && npm ci || npm install)
npm --prefix web run build
go run ./tools/sync-web-dist

rm -rf dist && mkdir -p dist

package() {
  local goos="$1" goarch="$2" ext="$3" archive="$4" goarm="${5:-}"
  local arch_label="${goarch}${goarm:+v${goarm}}"
  local bin="frp-warden${ext}"
  local stage="dist/stage_${goos}_${arch_label}"
  rm -rf "${stage}" && mkdir -p "${stage}"
  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" GOARM="${goarm}" \
    go build -trimpath -ldflags "${LDFLAGS}" -o "${stage}/${bin}" ./cmd/frp-warden
  for f in "${EXTRA_FILES[@]}"; do
    mkdir -p "${stage}/$(dirname "${f}")"
    cp "${f}" "${stage}/${f}"
  done
  case "${archive}" in
    tar.gz) tar -C "${stage}" -czf "dist/frp-warden_${VERSION}_${goos}_${arch_label}.tar.gz" . ;;
    zip)    command -v zip >/dev/null && ( cd "${stage}" && zip -qr "../frp-warden_${VERSION}_${goos}_${arch_label}.zip" . ) ;;
  esac
  rm -rf "${stage}"
}

# 多平台:<PLATFORMS>
package linux   amd64 ""   tar.gz
package linux   arm64 ""   tar.gz
package linux   386   ""   tar.gz
package linux   arm   ""   tar.gz 7
package windows amd64 .exe zip
package windows 386   .exe zip

( cd dist && (sha256sum frp-warden_* 2>/dev/null || shasum -a 256 frp-warden_*) > SHA256SUMS )
echo "done ${VERSION}"; ls -1 dist
