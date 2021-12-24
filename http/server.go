package http

import (
	"context"
	"errors"
	"fmt"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/ext"
	"github.com/spf13/viper"
	"net"
	"net/http"
)

var (
	_ runv.Liveness = new(Server)
	_ runv.Initable = new(Server)
)

type ServerOption func(s *Server)

type Options struct {
	address string
	tlscert string
	tlskey  string
}

type Server struct {
	*ext.StateWorker
	conkey  string
	server  *http.Server
	routerf func() http.Handler
}

func NewServer(opts ...ServerOption) *Server {
	hs := &Server{
		StateWorker: ext.NewStateWorker(context.TODO()),
		routerf: func() http.Handler {
			return http.DefaultServeMux
		},
		conkey: "server",
	}
	for _, opt := range opts {
		opt(hs)
	}
	return hs
}

func (h *Server) OnInit() error {
	mkey := func(k string) string {
		return "http." + h.conkey + "." + k
	}
	viper.SetDefault(mkey("address"), "0.0.0.0:8000")
	opts := Options{
		address: viper.GetString(mkey("address")),
		tlscert: viper.GetString(mkey("tlsCert")),
		tlskey:  viper.GetString(mkey("tlsKey")),
	}
	h.server = &http.Server{
		Addr:    opts.address,
		Handler: h.routerf(),
	}
	h.AddStateTask("http-server", func(ctx context.Context) error {
		h.server.BaseContext = func(l net.Listener) context.Context {
			return context.WithValue(ctx, "conn.address", opts.address)
		}
		var err error
		if opts.tlscert != "" && opts.tlskey != "" {
			flow.Log().Infof("server listen serve[TLS], addr: %s", opts.address)
			err = h.server.ListenAndServeTLS(opts.tlscert, opts.tlskey)
		} else {
			flow.Log().Infof("server listen serve, addr: %s", opts.address)
			err = h.server.ListenAndServe()
		}
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return fmt.Errorf("server listener, addr: %s, error: %w", opts.address, err)
		}
		return nil
	})
	return nil
}

func (h *Server) Server() *http.Server {
	return h.server
}

func (h *Server) ServerHandler() http.Handler {
	return h.server.Handler
}

func (h *Server) Startup(ctx context.Context) error {
	return h.StateWorker.Startup(ctx)
}

func (h *Server) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		flow.Log().Errorf("http-server shutdown, error: %s", err)
	}
	return h.StateWorker.Shutdown(ctx)
}

func WithRouterFactory(factory func() http.Handler) ServerOption {
	return func(s *Server) {
		s.routerf = factory
	}
}

func WithConfigKey(key string) ServerOption {
	return func(s *Server) {
		s.conkey = key
	}
}
