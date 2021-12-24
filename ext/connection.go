package ext

import (
	"context"
	"errors"
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
	OnNetCloseFunc func(conn net.Conn)
	OnNetRecvFunc  func(conn net.Conn) error
	OnNetErrorFunc func(conn *NetConnection, err error) (continued bool)
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
	ReadTimeout time.Duration `toml:"read-timeout"`
}

type NetOptions func(c *NetConnection)

type NetConnection struct {
	onOpenFunc  OnNetOpenFunc
	onCloseFunc OnNetCloseFunc
	onRecvFunc  OnNetRecvFunc
	onErrFunc   OnNetErrorFunc
	onDialFunc  OnNetDialFunc
	config      NetConfig
	downctx     context.Context
	downfun     context.CancelFunc
	conn        net.Conn
}

func NewNetConnection(config NetConfig, opts ...NetOptions) *NetConnection {
	ctx, fun := context.WithCancel(context.TODO())
	nc := &NetConnection{
		downctx:     ctx,
		downfun:     fun,
		config:      config,
		onDialFunc:  OnTCPDialFunc,
		onOpenFunc:  OnTCPOpenFunc,
		onCloseFunc: func(_ net.Conn) {},
	}
	for _, opt := range opts {
		opt(nc)
	}
	return nc
}

func (nc *NetConnection) SetDialFunc(f OnNetDialFunc) *NetConnection {
	nc.onDialFunc = f
	return nc
}

func (nc *NetConnection) SetOpenFunc(f OnNetOpenFunc) *NetConnection {
	nc.onOpenFunc = f
	return nc
}

func (nc *NetConnection) SetCloseFunc(f OnNetCloseFunc) *NetConnection {
	nc.onCloseFunc = f
	return nc
}

func (nc *NetConnection) SetRecvFunc(f OnNetRecvFunc) *NetConnection {
	nc.onRecvFunc = f
	return nc
}

func (nc *NetConnection) OnErrorFunc(f OnNetErrorFunc) *NetConnection {
	nc.onErrFunc = f
	return nc
}

var (
	ErrMaxRetry = errors.New("reconnect max retry")
)

func (nc *NetConnection) Listen() {
	runv.AssertNNil(nc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(nc.onErrFunc, "'onErrFunc' is required")
	runv.AssertNNil(nc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(nc.onOpenFunc, "'onOpenFunc' is required")
	runv.AssertNNil(nc.onCloseFunc, "'onCloseFunc' is required")
	nc.config.ReadTimeout = nc.duration(nc.config.ReadTimeout, time.Second, time.Second*10)
	nc.config.RetryDelay = nc.duration(nc.config.RetryDelay, time.Second, time.Second*5)
	// 通过Connect信号来控制自动重连
	reconnsigs := make(chan struct{}, 1)
	closesigs := make(chan struct{}, 1)
	var (
		count int
		delay time.Duration
	)
	sendclose := func() {
		closesigs <- struct{}{}
	}
	restate := func() {
		count = 0
		delay = time.Millisecond * 5
	}
	retry := func(err error) {
		select {
		case <-nc.downctx.Done():
			sendclose()
			return
		default:
			// next
		}
		// test errors
		if !nc.onErrFunc(nc, err) {
			sendclose()
			return
		}
		// retry
		if nc.config.RetryMax > 0 && count >= nc.config.RetryMax {
			nc.onErrFunc(nc, fmt.Errorf("max-retry=%d: %w", count, ErrMaxRetry))
			sendclose()
		} else {
			count++
			time.Sleep(nc.config.RetryDelay)
			reconnsigs <- struct{}{}
		}
	}
	defer nc.Close()
	reconnsigs <- struct{}{} // first connect
	for {
		select {
		case <-nc.downctx.Done():
			return

		case <-reconnsigs:
			addr := fmt.Sprintf("%s://%s", nc.config.Network, nc.config.RemoteAddress)
			flow.Log().Infof("connecting to: %s", addr)
			if conn, err := nc.onDialFunc(nc.config); err != nil {
				retry(fmt.Errorf("connection dial: %w", err))
			} else if err = nc.onOpenFunc(conn, nc.config); err != nil {
				retry(fmt.Errorf("connection open: %w", err))
			} else {
				restate()
				flow.Log().Infof("connected: %s OK", addr)
				nc.conn = conn
			}

		case <-closesigs:
			return

		default:
			if err := nc.conn.SetReadDeadline(time.Now().Add(nc.config.ReadTimeout)); err != nil {
				retry(fmt.Errorf("connection set read options: %w", err))
			} else if err = nc.onRecvFunc(nc.conn); err != nil {
				if ne, ok := err.(net.Error); ok && (ne.Temporary() || ne.Timeout()) {
					time.Sleep(nc.duration(delay, time.Millisecond*5, time.Millisecond*100))
				} else if err != nil {
					retry(fmt.Errorf("connection recv: %w", err))
				}
			}
		}
	}
}

func (nc *NetConnection) Shutdown() {
	nc.downfun()
}

func (nc *NetConnection) Done() <-chan struct{} {
	return nc.downctx.Done()
}

func (nc *NetConnection) Close() {
	if nc.conn == nil {
		return
	}
	defer nc.onCloseFunc(nc.conn)
	if err := nc.conn.Close(); err != nil && strings.Contains(err.Error(), "closed network connection") {
		nc.onErrFunc(nc, fmt.Errorf("connection close: %w", err))
	}
}

func (nc *NetConnection) duration(v time.Duration, def, max time.Duration) time.Duration {
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
