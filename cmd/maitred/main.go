package main

import (
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/sauerbraten/maitred/internal/db"
)

func main() {
	db, err := db.New("maitred.sqlite")
	if err != nil {
		log.Fatalln("error opening users database:", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:28787")
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

		ch := NewConnHandler(db, conn)

		ch.handleFirstMessage(stop)
	}
}
