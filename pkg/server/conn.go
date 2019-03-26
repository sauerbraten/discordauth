package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sauerbraten/maitred/internal/db"
	"github.com/sauerbraten/maitred/pkg/auth"
)

type clientConn struct {
	db   *db.Database
	conn *net.TCPConn
	in   *bufio.Scanner
}

func newClientConn(db *db.Database, conn *net.TCPConn) *clientConn {
	return &clientConn{
		db:   db,
		conn: conn,
		in:   bufio.NewScanner(conn),
	}
}

// called by wrapping types Adminconn and Serverconn
func (cc *clientConn) run(stop <-chan struct{}, handle func(string)) {
	incoming := make(chan string)
	go func() {
		for cc.in.Scan() {
			incoming <- cc.in.Text()
		}
		if err := cc.in.Err(); err != nil {
			log.Println(err)
		}
		close(incoming)
	}()

	for {
		select {
		case msg, ok := <-incoming:
			if !ok {
				log.Println(cc.conn.RemoteAddr(), "closed the connection")
				return
			}
			handle(msg)
		case <-stop:
			log.Println("closing connection to", cc.conn.RemoteAddr())
			cc.conn.Close()
			return
		}
	}
}

func (cc *clientConn) respond(format string, args ...interface{}) {
	response := fmt.Sprintf(format, args...)
	log.Println("responding with", response)
	cc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := cc.conn.Write([]byte(response + "\n"))
	if err != nil {
		log.Printf("failed to send '%s': %v", response, err)
	}
}

func (cc *clientConn) generateChallenge(name string) (challenge, solution string, err error) {
	pubkey, err := cc.db.GetPublicKey(name)
	if err != nil {
		return "", "", err
	}

	challenge, solution, err = auth.GenerateChallenge(pubkey)
	if err != nil {
		err = fmt.Errorf("could not generate challenge using pubkey %s of %s: %v", pubkey, name, err)
	}

	return
}
