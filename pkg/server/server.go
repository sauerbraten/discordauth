package server

import (
	"log"
	"net"

	"github.com/sauerbraten/maitred/internal/db"
	"github.com/sauerbraten/maitred/pkg/protocol"
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
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("error accepting connection: %v", err)
		}

		conn := protocol.NewConn(nil)
		conn.Start(tcpConn)

		h := newHandler(conn, s.db, s.stop)
		go h.run()
	}
}
