package client

import (
	"net"

	"github.com/sauerbraten/maitred/pkg/protocol"
)

type conn struct {
	addr *net.TCPAddr
	*protocol.Conn
	onConnect func()
}

func newConn(addr *net.TCPAddr, onConnect func(), onDisconnect func(error)) (*conn, <-chan string) {
	pConn, inc := protocol.NewConn(onDisconnect)

	c := &conn{
		addr:      addr,
		Conn:      pConn,
		onConnect: onConnect,
	}

	return c, inc
}

func (c *conn) connect() error {
	conn, err := net.DialTCP("tcp", nil, c.addr)
	if err != nil {
		return err
	}

	c.Conn.Start(conn)
	c.onConnect()
	return nil
}
