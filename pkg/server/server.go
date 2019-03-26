package server

import (
	"log"
	"net"
	"time"

	"github.com/sauerbraten/maitred/internal/db"
)

type Server struct {
	listenAddr *net.TCPAddr
	db         *db.Database
	stop       <-chan struct{}
}

func New(listenAddr *net.TCPAddr, db *db.Database, stop <-chan struct{}) *Server {
	return &Server{
		listenAddr: listenAddr,
		db:         db,
		stop:       stop,
	}
}

func (s *Server) Listen() {
	listener, err := net.ListenTCP("tcp", s.listenAddr)
	if err != nil {
		log.Printf("error starting to listen on %s: %v", s.listenAddr, err)
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("error accepting connection: %v", err)
		}

		conn.SetKeepAlive(true)
		conn.SetKeepAlivePeriod(2 * time.Minute)

		sc := newHandler(newClientConn(s.db, conn))

		sc.run(s.stop, sc.handle)
	}
}
