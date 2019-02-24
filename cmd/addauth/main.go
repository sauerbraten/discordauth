package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sauerbraten/maitred/internal/db"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: addauth <name> <pubkey>")
		os.Exit(1)
		return
	}

	name, pubkey := os.Args[1], os.Args[2]

	db, err := db.New("users.sqlite")
	if err != nil {
		log.Fatalln("error opening users database:", err)
	}

	err = db.AddUser(name, pubkey)
	if err != nil {
		log.Fatalln("error adding user to database:", err)
	}
}
