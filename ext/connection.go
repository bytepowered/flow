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
	OnNetRecvFunc  func(conn net.Conn, data *bytes.Buffer) error
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
	// Connection
	WriteTimeout time.Duration `toml:"write-timeout"`
	ReadTimeout  time.Duration `toml:"read-timeout"`
}

type NetOptions func(c *NetConnection)

type NetConnection struct {
	onOpenFunc OnNetOpenFunc
	onReadFunc OnNetReadFunc
	onRecvFunc OnNetRecvFunc
	onErrFunc  OnNetErrorFunc
	onDialFunc OnNetDialFunc
	config     NetConfig
	downctx    context.Context
	downfun    context.CancelFunc
	conn       net.Conn
	sendq      chan []byte
	sendqs     uint
}

func NewNetConnection(config NetConfig, opts ...NetOptions) *NetConnection {
	ctx, fun := context.WithCancel(context.TODO())
	nc := &NetConnection{
		downctx:    ctx,
		downfun:    fun,
		config:     config,
		onDialFunc: OnTCPDialFunc,
		onOpenFunc: OnTCPOpenFunc,
		sendqs:     10,
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
	tc.config.WriteTimeout = tc.duration(tc.config.WriteTimeout, time.Second, time.Second*10)
	tc.config.ReadTimeout = tc.duration(tc.config.ReadTimeout, time.Second, time.Second*10)
	tc.config.RetryDelay = tc.duration(tc.config.RetryDelay, time.Second, time.Second*5)
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
			time.Sleep(tc.config.RetryDelay)
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
					// FIXME 处理存留未发送队列
					tc.sendq = make(chan []byte, tc.sendqs)
					flow.Log().Infof("connected: %s OK", addr)
					tc.conn = conn
				}
			}

		case <-closed:
			return

		case data := <-tc.sendq:
			if err := tc.conn.SetWriteDeadline(time.Now().Add(tc.config.WriteTimeout)); err != nil {
				tc.onErrFunc(fmt.Errorf("connection write error: %w", err))
			} else if n, err := tc.conn.Write(data); err != nil {
				tc.onErrFunc(fmt.Errorf("connection write bytes error, bytes: %d %w", n, err))
			}

		default:
			_ = tc.conn.SetReadDeadline(time.Now().Add(tc.config.ReadTimeout))
			if data, err := tc.onReadFunc(tc.conn); err == nil {
				if err := tc.onRecvFunc(tc.conn, data); err != nil {
					tc.onErrFunc(err)
				}
			} else if ne, ok := err.(net.Error); ok && (ne.Temporary() || ne.Timeout()) {
				time.Sleep(tc.duration(delay, time.Millisecond*5, time.Millisecond*100))
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

func (tc *NetConnection) Done() <-chan struct{} {
	return tc.downctx.Done()
}

func (tc *NetConnection) Send(data []byte) {
	select {
	case <-tc.downctx.Done():
		flow.Log().Errorf("send data to closed connection: %s", tc.config.RemoteAddress)
	default:
		tc.sendq <- data
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

func (tc *NetConnection) duration(v time.Duration, def, max time.Duration) time.Duration {
	if v == 0 {
		v = def
	} else {
		v *= 2
	}
	if v > max {
		v = max
	}
	return v
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

func WithSendSize(size uint) NetOptions {
	return func(c *NetConnection) {
		c.sendqs = size
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
