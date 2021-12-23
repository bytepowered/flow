package ext

import (
	"bytes"
	"context"
	"fmt"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
	"net"
	"time"
)

type (
	OnNetDailFunc  func(opts NetConfig) (net.Conn, error)
	OnNetOpenFunc  func(conn net.Conn)
	OnNetReadFunc  func(conn net.Conn) (*bytes.Buffer, error)
	OnNetRecvFunc  func(conn net.Conn, buffer *bytes.Buffer)
	OnNetErrorFunc func(err error)
)

type NetConfig struct {
	Network       string `toml:"network"`
	RemoteAddress string `toml:"address"`
	BindAddress   string `toml:"bind"`
}

type NetOptions func(c *NetConnection)

type NetConnection struct {
	onOpenFunc OnNetOpenFunc
	onReadFunc OnNetReadFunc
	onRecvFunc OnNetRecvFunc
	onErrFunc  OnNetErrorFunc
	onDailFunc OnNetDailFunc
	conn       net.Conn
	config     NetConfig
	closectx   context.Context
	closefun   context.CancelFunc
}

func NewNetConnection(config NetConfig, opts ...NetOptions) *NetConnection {
	ctx, fun := context.WithCancel(context.TODO())
	nc := &NetConnection{
		closectx: ctx,
		closefun: fun,
		config:   config,
	}
	for _, opt := range opts {
		opt(nc)
	}
	return nc
}

func (tc *NetConnection) OnDailFunc(f OnNetDailFunc) *NetConnection {
	tc.onDailFunc = f
	return tc
}

func (tc *NetConnection) OnOpenFunc(f OnNetOpenFunc) *NetConnection {
	tc.onOpenFunc = f
	return tc
}

func (tc *NetConnection) OnReadFunc(f OnNetReadFunc) *NetConnection {
	tc.onReadFunc = f
	return tc
}

func (tc *NetConnection) OnRecv(f OnNetRecvFunc) *NetConnection {
	tc.onRecvFunc = f
	return tc
}

func (tc *NetConnection) OnErrorFunc(f OnNetErrorFunc) *NetConnection {
	tc.onErrFunc = f
	return tc
}

func (tc *NetConnection) Listen() {
	runv.AssertNNil(tc.onRecvFunc, "'onRecvFunc' is required")
	runv.AssertNNil(tc.onDailFunc, "'onDailFunc' is required")
	runv.AssertNNil(tc.onErrFunc, "'onErrFunc' is required")
	runv.AssertNNil(tc.onDailFunc, "'onDailFunc' is required")
	runv.AssertNNil(tc.onOpenFunc, "'onOpenFunc' is required")
	conn, err := tc.onDailFunc(tc.config)
	if err != nil {
		tc.onErrFunc(fmt.Errorf("open connection error: %w", err))
		return
	}
	runv.AssertNNil(conn, "dail connection is required")
	defer tc.Close()
	tc.onOpenFunc(conn)
	delay := time.Millisecond * 10
	for {
		select {
		case <-tc.closectx.Done():
			return
		default:
			// next to read
		}
		if buffer, rerr := tc.onReadFunc(conn); rerr == nil {
			tc.onRecvFunc(conn, buffer)
		} else if ne, ok := rerr.(net.Error); ok && ne.Temporary() {
			delay = tc.delay(delay)
			flow.Log().Warnf("connection read error: %v; retrying in %v", rerr, delay)
			time.Sleep(delay)
		} else {
			tc.onErrFunc(rerr)
		}
	}
}

func (tc *NetConnection) Close() {
	if tc.conn == nil {
		return
	}
	if err := tc.conn.Close(); err != nil {
		tc.onErrFunc(fmt.Errorf("connection close error: %w", err))
	}
}

func (tc *NetConnection) delay(delay time.Duration) time.Duration {
	if delay == 0 {
		delay = 5 * time.Millisecond
	} else {
		delay *= 2
	}
	if max := 1 * time.Second; delay > max {
		delay = max
	}
	return delay
}

func WithOpenFunc(f OnNetOpenFunc) NetOptions {
	return func(c *NetConnection) {
		c.onOpenFunc = f
	}
}

func WithReadFunc(f OnNetReadFunc) NetOptions {
	return func(c *NetConnection) {
		c.onReadFunc = f
	}
}

func WithRecvFunc(f OnNetRecvFunc) NetOptions {
	return func(c *NetConnection) {
		c.onRecvFunc = f
	}
}

func WithErrorFunc(f OnNetErrorFunc) NetOptions {
	return func(c *NetConnection) {
		c.onErrFunc = f
	}
}

func WithDailFunc(f OnNetDailFunc) NetOptions {
	return func(c *NetConnection) {
		c.onDailFunc = f
	}
}
