// Package version 暴露构建版本信息。
//
// 版本号唯一来源是 Git tag:
//   - 正式版本号 = 推送的 vX.Y.Z tag 去掉前缀 v,由 ldflags 注入。
//   - 日常 / CI 构建由 git describe 派生。
//
// 缺省值为 dev。启动日志通过 version.String() 打印,便于运维核对版本。
package version

// Version 是构建版本号;由 ldflags 注入,缺省为 dev。
var Version = "dev"

// Name 是二进制名称。
var Name = "frp-warden"

// String 返回单行、便于阅读的版本摘要。
func String() string {
	return Name + " " + Version
}
