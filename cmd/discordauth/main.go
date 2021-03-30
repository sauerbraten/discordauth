package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/sauerbraten/maitred/v2/internal/db"
	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

func main() {
	addr, err := net.ResolveTCPAddr("tcp", ":28787")
	if err != nil {
		log.Fatalln("error starting to listen on :28787:", err)
	}

	db, err := db.New("users.sqlite")
	if err != nil {
		log.Fatalln("error opening users database:", err)
	}

	stop := make(chan struct{})

	s := newServer(addr, db, stop)

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	go s.Listen()

	<-interrupt
	close(stop) // disconnects from Discord

	err = db.Close()
	if err != nil {
		log.Fatalln("error closing users database:", err)
	}
}

type Server struct {
	// sauer server connections
	listenAddr *net.TCPAddr

	db *db.Database

	// shared
	stop <-chan struct{}
}

func newServer(listenAddr *net.TCPAddr, db *db.Database, stop <-chan struct{}) *Server {
	s := &Server{
		listenAddr: listenAddr,
		db:         db,
		stop:       stop,
	}

	stopDiscord := setupDiscord(s.addUser)

	go func() {
		<-stop
		stopDiscord()
	}()

	return s
}

func (s *Server) Listen() {
	listener, err := net.ListenTCP("tcp", s.listenAddr)
	if err != nil {
		log.Printf("error starting to listen on %s: %v", s.listenAddr, err)
	}

	for {
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("error accepting connection: %v", err)
		}

		conn := protocol.NewConn(nil)
		conn.Start(tcpConn)

		h := newHandler(conn, s.stop, s.getPublicKey, s.updateUserLastAuthed)
		go h.run()
	}
}

func (s *Server) addUser(name, pubkey string, override bool) error {
	// let's ensure we don't process garbage
	_, err := auth.ParsePublicKey(pubkey)
	if err != nil {
		return fmt.Errorf("parsing public key: %w", err)
	}
	return s.db.AddUser(name, pubkey, override)
}

func (s *Server) getPublicKey(name string) (*auth.PublicKey, bool) {
	pubkey, err := s.db.GetPublicKey(name)
	if err != nil {
		return nil, false
	}
	pk, err := auth.ParsePublicKey(pubkey)
	if err != nil {
		return nil, false
	}
	return &pk, true
}

func (s *Server) updateUserLastAuthed(name string) {
	err := s.db.UpdateUserLastAuthed(name)
	if err != nil {
		log.Println(err)
	}
}
