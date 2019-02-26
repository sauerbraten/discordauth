package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/pkg/protocol"

	"github.com/sauerbraten/maitred/internal/db"
	"github.com/sauerbraten/maitred/pkg/auth"
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

func (h *ConnHandler) handleFirstMessage(stop <-chan struct{}) {
	if !h.in.Scan() {
		log.Println("error handling first message:", h.in.Err())
		return
	}

	msg := h.in.Text()

	log.Println("received first message", msg)

	if strings.HasPrefix(msg, protocol.ReqAdmin) {
		ach := NewAdminConnHandler(h)
		go ach.run(stop, ach.handle)
		ach.handle(msg)
	} else if strings.HasPrefix(msg, protocol.RegServ) {
		sch := NewServerConnHandler(h)
		go sch.run(stop, sch.handle)
		sch.handle(msg)
	} else {
		log.Printf("unexpected first message '%s'", msg)
	}
}

// called by wrapping types AdminConnHandler and ServerConnHandler
func (h *ConnHandler) run(stop <-chan struct{}, handle func(string)) {
	incoming := make(chan string)
	go func() {
		for h.in.Scan() {
			incoming <- h.in.Text()
		}
		if err := h.in.Err(); err != nil {
			log.Println(err)
		}
		close(incoming)
	}()

	for {
		select {
		case msg, ok := <-incoming:
			if !ok {
				log.Println(h.conn.RemoteAddr(), "closed the connection")
				return
			}
			handle(msg)
		case <-stop:
			log.Println("closing connection to", h.conn.RemoteAddr())
			h.conn.Close()
			return
		}
	}
}

func (h *ConnHandler) respond(format string, args ...interface{}) {
	response := fmt.Sprintf(format, args...)
	log.Println("responding with", response)
	h.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := h.conn.Write([]byte(response + "\n"))
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
