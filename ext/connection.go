package ext

import (
	"bytes"
	"context"
	"fmt"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
	"net"
	"strings"
	"time"
)

type (
	OnNetDialFunc  func(config NetConfig) (net.Conn, error)
	OnNetOpenFunc  func(conn net.Conn, config NetConfig) error
	OnNetReadFunc  func(conn net.Conn) (*bytes.Buffer, error)
	OnNetRecvFunc  func(conn net.Conn, buffer *bytes.Buffer)
	OnNetErrorFunc func(err error)
)

type NetConfig struct {
	Network       string `toml:"network"`
	RemoteAddress string `toml:"address"`
	BindAddress   string `toml:"bind"`
	// Reconnect
	RetryMax   int           `toml:"retry-max"`
	RetryDelay time.Duration `toml:"retry-delay"`
	// TCP
	TCPNoDelay   bool `toml:"tcp-no-delay"`
	TCPKeepAlive uint `toml:"tcp-keep-alive"`
}

type NetOptions func(c *NetConnection)

type NetConnection struct {
	onOpenFunc OnNetOpenFunc
	onReadFunc OnNetReadFunc
	onRecvFunc OnNetRecvFunc
	onErrFunc  OnNetErrorFunc
	onDialFunc OnNetDialFunc
	conn       net.Conn
	config     NetConfig
	downctx context.Context
	downfun context.CancelFunc
}

func NewNetConnection(config NetConfig, opts ...NetOptions) *NetConnection {
	ctx, fun := context.WithCancel(context.TODO())
	nc := &NetConnection{
		downctx:    ctx,
		downfun:    fun,
		config:     config,
		onDialFunc: OnTCPDialFunc,
		onOpenFunc: OnTCPOpenFunc,
	}
	for _, opt := range opts {
		opt(nc)
	}
	return nc
}

func (tc *NetConnection) SetDialFunc(f OnNetDialFunc) *NetConnection {
	tc.onDialFunc = f
	return tc
}

func (tc *NetConnection) SetOpenFunc(f OnNetOpenFunc) *NetConnection {
	tc.onOpenFunc = f
	return tc
}

func (tc *NetConnection) SetReadFunc(f OnNetReadFunc) *NetConnection {
	tc.onReadFunc = f
	return tc
}

func (tc *NetConnection) SetRecvFunc(f OnNetRecvFunc) *NetConnection {
	tc.onRecvFunc = f
	return tc
}

func (tc *NetConnection) OnErrorFunc(f OnNetErrorFunc) *NetConnection {
	tc.onErrFunc = f
	return tc
}

func (tc *NetConnection) Listen() {
	runv.AssertNNil(tc.onRecvFunc, "'onRecvFunc' is required")
	runv.AssertNNil(tc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(tc.onErrFunc, "'onErrFunc' is required")
	runv.AssertNNil(tc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(tc.onOpenFunc, "'onOpenFunc' is required")
	// 通过Connect信号来控制自动重连
	signals := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)
	count := int(0)
	retry := func() {
		if tc.config.RetryMax > 0 && count >= tc.config.RetryMax {
			tc.onErrFunc(fmt.Errorf("connect max retry: %d", count))
			closed <- struct{}{}
		} else {
			count++
			time.Sleep(tc.delay(tc.config.RetryDelay, time.Second, time.Second))
			signals <- struct{}{}
		}
	}
	defer tc.Close()
	delay := time.Millisecond * 5
	signals <- struct{}{} // first connect
	for {
		select {
		case <-tc.downctx.Done():
			return

		case <-signals:
			addr := fmt.Sprintf("%s://%s", tc.config.Network, tc.config.RemoteAddress)
			flow.Log().Infof("connecting to: %s", addr)
			if conn, err := tc.onDialFunc(tc.config); err != nil {
				tc.onErrFunc(fmt.Errorf("dial connection error: %w", err))
				retry()
			} else {
				if err := tc.onOpenFunc(conn, tc.config); err != nil {
					tc.onErrFunc(err)
					retry()
				} else {
					count = 0
					flow.Log().Infof("connected: %s OK", addr)
					tc.conn = conn
				}
			}

		case <-closed:
			return

		default:
			if buffer, err := tc.onReadFunc(tc.conn); err == nil {
				tc.onRecvFunc(tc.conn, buffer)
			} else if ne, ok := err.(net.Error); ok && (ne.Temporary() || ne.Timeout()) {
				delay = tc.delay(delay, time.Millisecond*5, time.Millisecond*100)
				flow.Log().Warnf("connection read error: %v; retrying in %v", err, delay)
				time.Sleep(delay)
			} else {
				tc.onErrFunc(err)
				retry()
			}
		}
	}
}

func (tc *NetConnection) Shutdown() {
	tc.downfun()
}

func (tc *NetConnection) Close() {
	if tc.conn == nil {
		return
	}
	if err := tc.conn.Close(); err != nil {
		tc.onErrFunc(fmt.Errorf("connection close error: %w", err))
	}
}

func (tc *NetConnection) delay(delay time.Duration, def, max time.Duration) time.Duration {
	if delay == 0 {
		delay = def
	} else {
		delay *= 2
	}
	if delay > max {
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

func WithDailFunc(f OnNetDialFunc) NetOptions {
	return func(c *NetConnection) {
		c.onDialFunc = f
	}
}

func OnTCPOpenFunc(conn net.Conn, config NetConfig) (err error) {
	network := conn.RemoteAddr().Network()
	socket, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("conn is not tcp connection, was: %s", network)
	}
	if strings.HasPrefix(network, "tcp") {
		if err = socket.SetNoDelay(config.TCPNoDelay); err != nil {
			return err
		}
		if config.TCPKeepAlive > 0 {
			_ = socket.SetKeepAlive(true)
			err = socket.SetKeepAlivePeriod(time.Second * time.Duration(config.TCPKeepAlive))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func OnTCPDialFunc(opts NetConfig) (net.Conn, error) {
	if !strings.HasPrefix(opts.Network, "tcp") {
		return nil, fmt.Errorf("'network' requires 'tcp' protocols")
	}
	laddr, _ := netResolveTCPAddr(opts.Network, opts.BindAddress)
	raddr, err := netResolveTCPAddr(opts.Network, opts.RemoteAddress)
	if err != nil {
		return nil, err
	}
	return net.DialTCP(opts.Network, laddr, raddr)
}

func netResolveTCPAddr(network, address string) (*net.TCPAddr, error) {
	if address == "" {
		return nil, fmt.Errorf("'address' is required")
	}
	if addr, err := net.ResolveTCPAddr(network, address); err != nil {
		return nil, err
	} else {
		return addr, nil
	}
}
