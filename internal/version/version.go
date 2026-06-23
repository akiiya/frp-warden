// Package version 暴露 frp-warden 的构建元信息。
//
// 下列变量可在构建时通过 -ldflags 覆盖，例如：
//
//	go build -ldflags "-X github.com/fengheasia/frp-warden/internal/version.Version=v0.1.0"
package version

// 构建元信息。默认值用于本地/开发构建；CI 通过 -ldflags 覆盖，
// 使发布的二进制报告真实版本。
var (
	// Name 是项目/二进制名称。
	Name = "frp-warden"

	// Version 是构建的语义化版本（如 v0.1.0）。
	Version = "0.0.0-dev"

	// Commit 是构建所基于的 git 短哈希。
	Commit = "unknown"

	// BuildDate 是 UTC 构建时间戳（RFC3339）。
	BuildDate = "unknown"
)

// String 返回单行、便于阅读的版本摘要。
func String() string {
	return Name + " " + Version + " (commit " + Commit + ", built " + BuildDate + ")"
}
