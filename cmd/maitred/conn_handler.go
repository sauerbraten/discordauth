package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/internal/db"
	"github.com/sauerbraten/maitred/pkg/auth"
	"github.com/sauerbraten/maitred/pkg/protocol"
)

type ConnHandler struct {
	db   *db.Database
	conn *net.TCPConn
	in   *bufio.Scanner
}

func NewConnHandler(db *db.Database, conn *net.TCPConn) *ConnHandler {
	return &ConnHandler{
		db:   db,
		conn: conn,
		in:   bufio.NewScanner(conn),
	}
}

func (ch *ConnHandler) handleFirstMessage(stop <-chan struct{}) {
	if !ch.in.Scan() {
		log.Println("error handling first message:", ch.in.Err())
		return
	}

	msg := ch.in.Text()

	log.Println("received first message", msg)

	if strings.HasPrefix(msg, protocol.ReqAdmin) {
		ac := NewAdminConn(ch)
		go ac.run(stop, ac.handle)
		ac.handle(msg)
	} else if strings.HasPrefix(msg, protocol.RegServ) {
		sc := NewServerConn(ch)
		go sc.run(stop, sc.handle)
		sc.handle(msg)
	} else {
		log.Printf("unexpected first message '%s'", msg)
	}
}

// called by wrapping types AdminConn and ServerConn
func (ch *ConnHandler) run(stop <-chan struct{}, handle func(string)) {
	incoming := make(chan string)
	go func() {
		for ch.in.Scan() {
			incoming <- ch.in.Text()
		}
		if err := ch.in.Err(); err != nil {
			log.Println(err)
		}
		close(incoming)
	}()

	for {
		select {
		case msg, ok := <-incoming:
			if !ok {
				log.Println(ch.conn.RemoteAddr(), "closed the connection")
				return
			}
			handle(msg)
		case <-stop:
			log.Println("closing connection to", ch.conn.RemoteAddr())
			ch.conn.Close()
			return
		}
	}
}

func (ch *ConnHandler) respond(format string, args ...interface{}) {
	response := fmt.Sprintf(format, args...)
	log.Println("responding with", response)
	ch.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := ch.conn.Write([]byte(response + "\n"))
	if err != nil {
		log.Printf("failed to send '%s': %v", response, err)
	}
}

func (ch *ConnHandler) generateChallenge(name string) (challenge, solution string, err error) {
	pubkey, err := ch.db.GetPublicKey(name)
	if err != nil {
		return "", "", err
	}

	challenge, solution, err = auth.GenerateChallenge(pubkey)
	if err != nil {
		err = fmt.Errorf("could not generate challenge using pubkey %s of %s: %v", pubkey, name, err)
	}

	return
}
