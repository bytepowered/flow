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
	OnDialFunc  func(config Config) (net.Conn, error)
	OnOpenFunc  func(conn net.Conn, config Config) error
	OnCloseFunc func(conn net.Conn)
	OnRecvFunc  func(conn net.Conn) error
	OnErrorFunc func(conn *Connector, err error) (continued bool)
)

type kState uint

const (
	kStateReconnect kState = iota
	kStateConnected
	kStateClosed
)

type Config struct {
	Network       string `toml:"network"`
	RemoteAddress string `toml:"address"`
	BindAddress   string `toml:"bind"`
	// Reconnect
	RetryMax   int           `toml:"retry-max"`
	RetryDelay time.Duration `toml:"retry-delay"`
	// TCP
	TCPNoDelay   bool `toml:"tcp-no-delay"`
	TCPKeepAlive uint `toml:"tcp-keep-alive"`
	// Connector
	ReadTimeout time.Duration `toml:"read-timeout"`
}

type ConnectorOptions func(c *Connector)

type Connector struct {
	onOpenFunc  OnOpenFunc
	onCloseFunc OnCloseFunc
	onRecvFunc  OnRecvFunc
	onTestErr   OnErrorFunc
	onDialFunc  OnDialFunc
	config      Config
	downctx     context.Context
	downfun     context.CancelFunc
	conn        net.Conn
	state       kState
}

func NewConnector(config Config, opts ...ConnectorOptions) *Connector {
	ctx, fun := context.WithCancel(context.TODO())
	c := &Connector{
		downctx:     ctx,
		downfun:     fun,
		config:      config,
		onDialFunc:  OnTCPDialFunc,
		onOpenFunc:  OnTCPOpenFunc,
		onCloseFunc: func(_ net.Conn) {},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (nc *Connector) SetDialFunc(f OnDialFunc) *Connector {
	nc.onDialFunc = f
	return nc
}

func (nc *Connector) SetOpenFunc(f OnOpenFunc) *Connector {
	nc.onOpenFunc = f
	return nc
}

func (nc *Connector) SetCloseFunc(f OnCloseFunc) *Connector {
	nc.onCloseFunc = f
	return nc
}

func (nc *Connector) SetRecvFunc(f OnRecvFunc) *Connector {
	nc.onRecvFunc = f
	return nc
}

func (nc *Connector) OnErrorFunc(f OnErrorFunc) *Connector {
	nc.onTestErr = f
	return nc
}

func (nc *Connector) Serve() {
	runv.AssertNNil(nc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(nc.onTestErr, "'onTestErr' is required")
	runv.AssertNNil(nc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(nc.onOpenFunc, "'onOpenFunc' is required")
	runv.AssertNNil(nc.onCloseFunc, "'onCloseFunc' is required")
	nc.config.ReadTimeout = nc.inc(nc.config.ReadTimeout, time.Second, time.Second*10)
	nc.config.RetryDelay = nc.inc(nc.config.RetryDelay, time.Second, time.Second*5)
	var count int
	var delay time.Duration
	reset := func(conn net.Conn) {
		nc.conn = conn
		count = 0
		delay = time.Millisecond * 5
	}
	retry := func(err error) {
		select {
		case <-nc.downctx.Done():
			nc.state0(kStateClosed)
			return
		default:
			// next
		}
		// test errors
		if !nc.onTestErr(nc, err) {
			nc.state0(kStateClosed)
		} else if nc.config.RetryMax > 0 && count >= nc.config.RetryMax {
			nc.state0(kStateClosed)
		} else {
			count++
			time.Sleep(nc.config.RetryDelay)
			nc.state0(kStateReconnect)
		}
	}
	addr := fmt.Sprintf("%s://%s", nc.config.Network, nc.config.RemoteAddress)
	defer nc.Close()
	nc.state0(kStateReconnect)
	for {
		select {
		case <-nc.Done():
			return
		default:
			// next
		}
		switch nc.state {
		case kStateClosed:
			return
		case kStateReconnect:
			flow.Log().Infof("re-connecting to: %s, retry: %d", addr, count)
			if conn, err := nc.onDialFunc(nc.config); err != nil {
				retry(fmt.Errorf("connection dial: %w", err))
			} else if err = nc.onOpenFunc(conn, nc.config); err != nil {
				retry(fmt.Errorf("connection open: %w", err))
			} else {
				reset(conn)
				nc.state0(kStateConnected)
				flow.Log().Infof("connected: %s OK", addr)
			}
		case kStateConnected:
			if err := nc.conn.SetReadDeadline(time.Now().Add(nc.config.ReadTimeout)); err != nil {
				retry(fmt.Errorf("connection set read options: %w", err))
			} else if err = nc.onRecvFunc(nc.conn); err != nil {
				if werr := errors.Unwrap(err); werr != nil {
					err = werr
				}
				if nerr, ok := err.(net.Error); ok && (nerr.Temporary() || nerr.Timeout()) {
					time.Sleep(nc.inc(delay, time.Millisecond*5, time.Millisecond*100))
				} else if err != nil {
					retry(fmt.Errorf("connection recv: %w", err))
				}
			}
		}
	}
}

func (nc *Connector) Shutdown() {
	nc.downfun()
}

func (nc *Connector) Done() <-chan struct{} {
	return nc.downctx.Done()
}

func (nc *Connector) Close() {
	nc.close0()
}

func (nc *Connector) state0(s kState) {
	if s == kStateClosed {
		_ = nc.conn.SetDeadline(time.Time{})
	}
	nc.state = s
}

func (nc *Connector) close0() {
	if nc.conn == nil {
		return
	}
	defer nc.onCloseFunc(nc.conn)
	if err := nc.conn.Close(); err != nil && !strings.Contains(err.Error(), "closed network connection") {
		nc.onTestErr(nc, fmt.Errorf("connection close: %w", err))
	}
}

func (nc *Connector) inc(v time.Duration, def, max time.Duration) time.Duration {
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

func WithOpenFunc(f OnOpenFunc) ConnectorOptions {
	return func(c *Connector) {
		c.onOpenFunc = f
	}
}

func WithRecvFunc(f OnRecvFunc) ConnectorOptions {
	return func(c *Connector) {
		c.onRecvFunc = f
	}
}

func WithErrorFunc(f OnErrorFunc) ConnectorOptions {
	return func(c *Connector) {
		c.onTestErr = f
	}
}

func WithDailFunc(f OnDialFunc) ConnectorOptions {
	return func(c *Connector) {
		c.onDialFunc = f
	}
}

func OnTCPOpenFunc(conn net.Conn, config Config) (err error) {
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

func OnTCPDialFunc(opts Config) (net.Conn, error) {
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
