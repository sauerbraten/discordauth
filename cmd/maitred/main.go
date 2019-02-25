package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/sauerbraten/maitred/internal/db"
)

func main() {
	db, err := db.New("users.sqlite")
	if err != nil {
		log.Fatalln("error opening users database:", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", ":28787")
	if err != nil {
		log.Fatalln("error starting to listen on 0.0.0.0:28787:", err)
	}

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	stop := make(chan struct{})
	go Listen(addr, db, stop)

	<-interrupt
	close(stop)
}

func Listen(listenAddr *net.TCPAddr, db *db.Database, stop <-chan struct{}) {
	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Printf("error starting to listen on %s: %v", listenAddr, err)
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("error accepting connection: %v", err)
		}

		s := &AuthServer{
			conn:        conn,
			in:          bufio.NewScanner(conn),
			db:          db,
			pendingByID: map[uint]*request{},
		}

		go s.run(stop)
	}
}
