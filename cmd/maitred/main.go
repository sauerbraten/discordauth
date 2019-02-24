package main

import (
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
