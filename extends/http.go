package extends

import (
	"context"
	"errors"
	"fmt"
	flow "github.com/bytepowered/flow/pkg"
	"github.com/bytepowered/runv"
	"net/http"
)

var (
	_ runv.Liveness = new(HttpServer)
	_ runv.Initable = new(HttpServer)
)

type HttpServerOptions struct {
	Address string `toml:"address"`
}

type HttpServer struct {
	*flow.StateWorker
	opts    HttpServerOptions
	httpkey string
	server  *http.Server
}

func (h *HttpServer) OnInit() error {
	h.server = &http.Server{
		Addr: h.opts.Address,
	}
	h.AddWorkTask("http-server", func() error {
		if err := h.server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return fmt.Errorf("server listener error: %w", err)
		}
		return nil
	})
	return nil
}

func (h *HttpServer) Server() *http.Server {
	return h.server
}

func (h *HttpServer) Startup(ctx context.Context) error {
	return h.StateWorker.Startup(ctx)
}

func (h *HttpServer) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		flow.Log().Errorf("http-server shutdown, error: %s", err)
	}
	return h.StateWorker.Shutdown(ctx)
}
