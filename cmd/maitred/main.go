package main

import (
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/sauerbraten/maitred/v2/internal/db"
	"github.com/sauerbraten/maitred/v2/pkg/server"
)

func main() {
	db, err := db.New("maitred.sqlite")
	if err != nil {
		log.Fatalln("error opening users database:", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", ":28787")
	if err != nil {
		log.Fatalln("error starting to listen on 0.0.0.0:28787:", err)
	}

	stop := make(chan struct{})

	s := server.New(addr, db, stop)

	go s.Listen()

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt
	close(stop)
}
