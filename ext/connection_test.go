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
	}, WithErrorFunc(func(conn *NetConnection, err error) (continued bool) {
		fmt.Println(err)
		return true
	}), WithRecvFunc(func(conn net.Conn) error {
		r := bufio.NewReader(conn)
		line, _, err := r.ReadLine()
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		buf.Write(line)
		fmt.Println(buf.String())
		return nil
	}))

	time.AfterFunc(time.Second*30, func() {
		conn.Shutdown()
	})
	defer conn.Close()
	conn.Listen()
}
