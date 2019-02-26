package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/sauerbraten/maitred/pkg/protocol"
)

type AdminConnHandler struct {
	*ConnHandler
	adminName     string
	solution      string
	authenticated bool
}

func NewAdminConnHandler(ch *ConnHandler) *AdminConnHandler {
	return &AdminConnHandler{
		ConnHandler: ch,
	}
}

func (ach *AdminConnHandler) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd)+1:]

	switch cmd {
	case protocol.ReqAdmin:
		ach.handleReqAdmin(args)

	case protocol.ConfAdmin:
		ach.handleConfAdmin(args)

	case protocol.AddAuth:
		if !ach.authenticated {
			return
		}
		ach.handleAddAuth(args)

	case protocol.DelAuth:
		if !ach.authenticated {
			return
		}
		ach.handleDelAuth(args)

	default:
		log.Printf("no handler for command %s in '%s'", cmd, msg)
	}
}

func (ach *AdminConnHandler) handleReqAdmin(args string) {
	var adminName string
	_, err := fmt.Sscanf(args, "%s", &adminName)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAdmin, args, err)
		ach.conn.Close()
		return
	}

	challenge, solution, err := ach.generateChallenge(adminName)
	if err != nil {
		log.Printf("could not generate challenge to authenticate '%s' as admin: %v", adminName, err)
		ach.conn.Close()
		return
	}

	ach.adminName = adminName
	ach.solution = solution

	ach.respond("%s %s", protocol.ChalAdmin, challenge)
}

func (ach *AdminConnHandler) handleConfAdmin(args string) {
	var solution string
	_, err := fmt.Sscanf(args, "%s", &solution)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAdmin, args, err)
		ach.respond("%s %s", protocol.FailAdmin, "could not parse solution")
		ach.conn.Close()
		return
	}

	if solution == ach.solution {
		ach.authenticated = true
		ach.respond("%s", protocol.SuccAdmin)
		log.Printf("connection from %s successfully authenticated as admin '%s'", ach.conn.RemoteAddr(), ach.adminName)
	} else {
		ach.respond("%s", protocol.FailAdmin)
		ach.conn.Close()
		log.Printf("connection from %s failed to authenticate as admin '%s'", ach.conn.RemoteAddr(), ach.adminName)
	}
}

func (ach *AdminConnHandler) handleAddAuth(args string) {
	var name string
	var pubkey string
	_, err := fmt.Sscanf(args, "%s %s", &name, &pubkey)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.AddAuth, args, err)
		ach.conn.Close()
		return
	}

	err = ach.db.AddUser(name, pubkey)
	if err != nil {
		log.Println(err)
		ach.respond("failed to add entry: %v", err)
		return
	}

	ach.respond("ok")
	log.Printf("admin '%s' (%s) added auth entry '%s' (pubkey '%s')", ach.adminName, ach.conn.RemoteAddr(), name, pubkey)
}

func (ach *AdminConnHandler) handleDelAuth(args string) {
	var name string
	_, err := fmt.Sscanf(args, "%s", &name)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.DelAuth, args, err)
		ach.conn.Close()
		return
	}

	err = ach.db.DelUser(name)
	if err != nil {
		log.Println(err)
		ach.respond("failed to delete entry: %v", err)
		return
	}

	ach.respond("ok")
	log.Printf("admin '%s' (%s) deleted auth entry '%s'", ach.adminName, ach.conn.RemoteAddr(), name)
}
