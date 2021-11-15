package extends

import (
	"context"
	"errors"
	"fmt"
	flow "github.com/bytepowered/flow/pkg"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
	"net"
	"net/http"
)

var (
	_ runv.Liveness = new(HttpServer)
	_ runv.Initable = new(HttpServer)
)

type HttpServerOption func(s *HttpServer)

type HttpServer struct {
	state   *flow.StateWorker
	confkey string
	server  *http.Server
	routerf func() http.Handler
}

func NewHttpServer(opts ...HttpServerOption) *HttpServer {
	hs := &HttpServer{
		routerf: func() http.Handler {
			return http.DefaultServeMux
		},
		confkey: "server",
	}
	for _, opt := range opts {
		opt(hs)
	}
	return hs
}

func (h *HttpServer) OnInit() error {
	mkey := func(k string) string {
		return "http." + h.confkey + "." + k
	}
	viper.SetDefault(mkey("address"), "0.0.0.0:8000")
	opts := httpopts{
		address: viper.GetString(mkey("address")),
		tlscert: viper.GetString(mkey("tlsCert")),
		tlskey:  viper.GetString(mkey("tlsKey")),
	}
	h.server = &http.Server{
		Addr:    opts.address,
		Handler: h.routerf(),
	}
	h.state.AddStateTask("http-server", func(ctx context.Context) error {
		h.server.BaseContext = func(l net.Listener) context.Context {
			return context.WithValue(ctx, "conn.address", opts.address)
		}
		var err error
		if opts.tlscert != "" && opts.tlskey != "" {
			flow.Log().Infof("server listen serve[TLS]")
			err = h.server.ListenAndServeTLS(opts.tlscert, opts.tlskey)
		} else {
			flow.Log().Infof("server listen serve")
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

func (h *HttpServer) ServerHandler() http.Handler {
	return h.server.Handler
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

func WithHttpServerRouterFunc(rf func() http.Handler) HttpServerOption {
	return func(s *HttpServer) {
		s.routerf = rf
	}
}

func WithHttpServerConfigKey(key string) HttpServerOption {
	return func(s *HttpServer) {
		s.confkey = key
	}
}

type httpopts struct {
	address string
	tlscert string
	tlskey  string
}
