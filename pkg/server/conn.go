package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

type conn struct {
	tcpConn  *net.TCPConn
	incoming chan string
}

func newConn(tcpConn *net.TCPConn) *conn {
	c := &conn{
		tcpConn:  tcpConn,
		incoming: make(chan string),
	}

	go c.run()

	return c
}

func (c *conn) run() {
	sc := bufio.NewScanner(c.tcpConn)
	for sc.Scan() {
		c.incoming <- sc.Text()
	}
	if err := sc.Err(); err != nil {
		log.Println(err)
	}
	close(c.incoming)
}

func (c *conn) respond(format string, args ...interface{}) {
	response := fmt.Sprintf(format, args...)
	log.Println("responding with", response)
	c.tcpConn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_, err := c.tcpConn.Write([]byte(response + "\n"))
	if err != nil {
		log.Printf("failed to send '%s': %v", response, err)
	}
}
