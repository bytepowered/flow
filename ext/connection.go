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
	OnNetDailFunc  func(opts NetOptions) (net.Conn, error)
	OnNetOpenFunc  func(conn net.Conn)
	OnNetReadFunc  func(conn net.Conn) (*bytes.Buffer, error)
	OnNetRecvFunc  func(conn net.Conn, buffer *bytes.Buffer)
	OnNetErrorFunc func(err error)
)

type NetOptions struct {
	Network       string `toml:"network"`
	RemoteAddress string `toml:"address"`
	BindAddress   string `toml:"bind"`
}

type NetConnection struct {
	onOpenFunc OnNetOpenFunc
	onReadFunc OnNetReadFunc
	onRecvFunc OnNetRecvFunc
	onErrFunc  OnNetErrorFunc
	onDailFunc OnNetDailFunc
	conn       net.Conn
	addr       string
	opts       NetOptions
	closectx   context.Context
	closefun   context.CancelFunc
}

func (tc *NetConnection) OnOpenFunc(f OnNetOpenFunc) *NetConnection {
	tc.onOpenFunc = f
	return tc
}

func (tc *NetConnection) OnReadFunc(f OnNetReadFunc) *NetConnection {
	tc.onReadFunc = f
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
	conn, err := tc.onDailFunc(tc.opts)
	if err != nil {
		tc.onErrFunc(fmt.Errorf("open connection: %s, error: %w", tc.addr, err))
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
