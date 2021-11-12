package extends

import (
	"context"
	"errors"
	"fmt"
	flow "github.com/bytepowered/flow/pkg"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
	"net/http"
)

var (
	_ runv.Liveness = new(HttpServer)
	_ runv.Initable = new(HttpServer)
)

type HttpServerOptions struct {
	address string
	tlscert string
	tlskey  string
}

type HttpServer struct {
	state   *flow.StateWorker
	opts    HttpServerOptions
	httpkey string
	server  *http.Server
	Router  func() http.Handler
}

func NewHttpServer() *HttpServer {
	return &HttpServer{
		Router: func() http.Handler {
			return http.DefaultServeMux
		},
	}
}

func (h *HttpServer) OnInit() error {
	mkey := func(k string) string {
		return "http." + h.httpkey + "." + k
	}
	viper.SetDefault(mkey("address"), "0.0.0.0:8000")
	h.opts = HttpServerOptions{
		address: viper.GetString(mkey("address")),
		tlscert: viper.GetString(mkey("tlsCert")),
		tlskey:  viper.GetString(mkey("tlsKey")),
	}
	h.server = &http.Server{
		Addr:    h.opts.address,
		Handler: h.Router(),
	}
	h.state.AddWorkTask("http-server", func() error {
		var err error
		if h.opts.tlscert != "" && h.opts.tlskey != "" {
			err = h.server.ListenAndServeTLS(h.opts.tlscert, h.opts.tlskey)
		} else {
			err = h.server.ListenAndServe()
		}
		if err != nil {
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
	return h.state.Startup(ctx)
}

func (h *HttpServer) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		flow.Log().Errorf("http-server shutdown, error: %s", err)
	}
	return h.state.Shutdown(ctx)
}
