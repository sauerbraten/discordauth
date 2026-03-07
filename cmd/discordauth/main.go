package main

import (
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/sauerbraten/maitred/v2/internal/db"
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

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go s.Listen()

	<-interrupt
	close(stop) // disconnects from Discord

	err = db.Close()
	if err != nil {
		log.Fatalln("error closing users database:", err)
	}
}
