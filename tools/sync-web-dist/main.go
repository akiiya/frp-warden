// sync-web-dist 将 web/dist 构建产物同步到 internal/webui/dist,供 Go embed 内嵌。
//
// 用法:
//
//	go run ./tools/sync-web-dist
//
// 行为:
//  1. 清空 internal/webui/dist 目录。
//  2. 复制 web/dist 的全部文件到 internal/webui/dist,保留目录结构。
//  3. 如果 web/dist 不存在,报错提示先执行前端构建。
//  4. 跨平台可用(Windows/Linux/macOS)。
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 从项目根目录运行,获取根目录路径。
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 获取工作目录失败: %v\n", err)
		os.Exit(1)
	}

	srcDir := filepath.Join(root, "web", "dist")
	dstDir := filepath.Join(root, "internal", "webui", "dist")

	// 检查 web/dist 是否存在。
	srcInfo, err := os.Stat(srcDir)
	if err != nil || !srcInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "错误: web/dist 不存在。请先执行:\n  cd web && npm install && npm run build\n")
		os.Exit(1)
	}

	// 清空目标目录。
	if err := os.RemoveAll(dstDir); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 清空 %s 失败: %v\n", dstDir, err)
		os.Exit(1)
	}

	// 复制文件。
	count := 0
	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		// 复制文件。
		if err := copyFile(path, dstPath); err != nil {
			return fmt.Errorf("复制 %s 失败: %w", rel, err)
		}
		count++
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 复制文件失败: %v\n", err)
		os.Exit(1)
	}

	// 统一路径分隔符为正斜杠(日志可读性)。
	srcDisplay := strings.ReplaceAll(srcDir, "\\", "/")
	dstDisplay := strings.ReplaceAll(dstDir, "\\", "/")
	fmt.Printf("已同步 %d 个文件: %s → %s\n", count, srcDisplay, dstDisplay)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
