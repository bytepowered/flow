package ext

import (
	"context"
	"errors"
	"fmt"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
	"io"
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

type ConnState uint

const (
	ConnStateConnecting ConnState = iota
	ConnStateConnected
	ConnStateDisconnecting
	ConnStateDisconnected
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
	onErrFunc   OnErrorFunc
	onDialFunc  OnDialFunc
	config      Config
	downctx     context.Context
	downfun     context.CancelFunc
	conn        net.Conn
	state       ConnState
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
	nc.onErrFunc = f
	return nc
}

func (nc *Connector) State() ConnState {
	return nc.state
}

func (nc *Connector) Serve() {
	runv.AssertNNil(nc.onDialFunc, "'onDialFunc' is required")
	runv.AssertNNil(nc.onErrFunc, "'onErrFunc' is required")
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
	retry := func() {
		if nc.config.RetryMax > 0 && count >= nc.config.RetryMax {
			nc.setState(ConnStateDisconnecting)
		} else {
			count++
			nc.setState(ConnStateConnecting)
			time.Sleep(nc.config.RetryDelay)
		}
	}
	checke := func(err error) {
		select {
		case <-nc.downctx.Done():
			nc.setState(ConnStateDisconnecting)
			return
		default:
			// next
		}
		if errors.Is(err, io.EOF) {
			retry()
		} else if nc.onErrFunc(nc, err) {
			return
		} else {
			retry()
		}
	}
	addr := fmt.Sprintf("%s://%s", nc.config.Network, nc.config.RemoteAddress)
	defer nc.Close()
	nc.setState(ConnStateConnecting)
	for {
		select {
		case <-nc.Done():
			return
		default:
			// next
		}
		switch nc.state {
		case ConnStateDisconnecting:
			return
		case ConnStateConnecting:
			flow.Log().Infof("re-connecting to: %s, retry: %d", addr, count)
			if conn, err := nc.onDialFunc(nc.config); err != nil {
				checke(fmt.Errorf("connection dial: %w", err))
			} else if err = nc.onOpenFunc(conn, nc.config); err != nil {
				checke(fmt.Errorf("connection open: %w", err))
			} else {
				reset(conn)
				nc.setState(ConnStateConnected)
				flow.Log().Infof("connected: %s OK", addr)
			}
		case ConnStateConnected:
			if err := nc.conn.SetReadDeadline(time.Now().Add(nc.config.ReadTimeout)); err != nil {
				checke(fmt.Errorf("connection set read options: %w", err))
			} else if err = nc.onRecvFunc(nc.conn); err != nil {
				var terr = err
				if werr := errors.Unwrap(err); werr != nil {
					terr = werr
				}
				if nerr, ok := terr.(net.Error); ok && (nerr.Temporary() || nerr.Timeout()) {
					time.Sleep(nc.inc(delay, time.Millisecond*5, time.Millisecond*100))
				} else {
					checke(err)
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

func (nc *Connector) setState(s ConnState) {
	if s == ConnStateDisconnecting && nc.conn != nil {
		_ = nc.conn.SetDeadline(time.Time{})
	}
	nc.state = s
}

func (nc *Connector) close0() {
	defer nc.setState(ConnStateDisconnected)
	if nc.conn == nil {
		return
	}
	defer nc.onCloseFunc(nc.conn)
	if err := nc.conn.Close(); err != nil && !strings.Contains(err.Error(), "closed network connection") {
		nc.onErrFunc(nc, fmt.Errorf("connection close: %w", err))
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
		c.onErrFunc = f
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
