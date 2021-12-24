package ext

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestConnection(t *testing.T) {
	conn := NewNetConnection(NetConfig{
		Network:       "tcp",
		RemoteAddress: "127.0.0.1:8888",
		RetryMax:      -1,
		RetryDelay:    time.Second,
	}, WithErrorFunc(func(err error) {
		fmt.Println(err)
	}), WithReadFunc(func(conn net.Conn) (*bytes.Buffer, error) {
		r := bufio.NewReader(conn)
		line, _, err := r.ReadLine()
		if err != nil {
			return nil, err
		}
		buf := new(bytes.Buffer)
		buf.Write(line)
		return buf, nil
	}), WithRecvFunc(func(conn net.Conn, buffer *bytes.Buffer) {
		fmt.Println(buffer.String())
	}))
	time.AfterFunc(time.Second*30, func() {
		conn.Shutdown()
	})
	defer conn.Close()
	conn.Listen()
}
