package main

import (
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/sauerbraten/jsonfile"
)

func main() {
	var users []*User
	err := jsonfile.ParseFile("users.json", &users)
	if err != nil {
		log.Fatalln("error parsing users.json:", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", ":28787")
	if err != nil {
		log.Fatalln("error starting to listen on 0.0.0.0:28787:", err)
	}

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	stop := make(chan struct{})
	go Listen(addr, users, stop)

	<-interrupt
	close(stop)
}
