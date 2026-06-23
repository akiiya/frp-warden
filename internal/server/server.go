// Package server 将构成 frp-warden 的两个 HTTP 监听器组装在一起:
// 管理后台(admin + webui)与 frps plugin 接口。
//
// admin server 挂载:
//   - /api/* → 管理 API(internal/admin)
//   - /      → 前端静态资源 + SPA fallback(internal/webui)
//
// plugin server 挂载:
//   - /plugin/frp → frps plugin hooks(internal/plugin)
//
// 两个 server 独立监听,plugin 接口默认仅回环。
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/fengheasia/frp-warden/internal/admin"
	"github.com/fengheasia/frp-warden/internal/config"
	"github.com/fengheasia/frp-warden/internal/plugin"
	"github.com/fengheasia/frp-warden/internal/store"
	"github.com/fengheasia/frp-warden/internal/webui"
)

// Server 持有管理后台与 plugin 两个 HTTP 服务。
type Server struct {
	cfg    config.Config
	admin  *http.Server
	plugin *http.Server
}

// New 根据给定配置与 store 构造 Server。
//
// admin server 组合 admin API handler 与 webui handler:
//   - /api/* 优先匹配 admin API
//   - /       fallback 到 webui(静态资源 + SPA)
//
// plugin server 独立监听 /plugin/frp,不受 SPA fallback 影响。
func New(cfg config.Config, st *store.Store) *Server {
	// admin server:组合 /api/* 与 / (webui)。
	adminMux := http.NewServeMux()
	adminMux.Handle("/api/", admin.NewHandler(st, cfg))
	adminMux.Handle("/", webui.NewHandler(webui.DistFS))

	// plugin server:独立,只处理 /plugin/frp。
	pluginMux := http.NewServeMux()
	pluginMux.Handle("/plugin/frp", plugin.NewHandler(st))

	return &Server{
		cfg: cfg,
		admin: &http.Server{
			Addr:              cfg.Server.AdminAddr,
			Handler:           adminMux,
			ReadHeaderTimeout: 10 * time.Second,
		},
		plugin: &http.Server{
			Addr:              cfg.Server.PluginAddr,
			Handler:           pluginMux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Run 启动两个 HTTP 服务并阻塞,直到 ctx 被取消后优雅关停。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		log.Printf("管理后台监听于 %s", s.cfg.Server.AdminAddr)
		if err := s.admin.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		log.Printf("frps plugin 监听于 %s（默认仅回环）", s.cfg.Server.PluginAddr)
		if err := s.plugin.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		_ = s.shutdown()
		return err
	}
	return s.shutdown()
}

func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return errors.Join(s.admin.Shutdown(ctx), s.plugin.Shutdown(ctx))
}
