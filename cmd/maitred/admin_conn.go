package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/sauerbraten/maitred/pkg/protocol"
)

type AdminConn struct {
	*ConnHandler
	adminName     string
	solution      string
	authenticated bool
}

func NewAdminConn(ch *ConnHandler) *AdminConn {
	return &AdminConn{
		ConnHandler: ch,
	}
}

func (ac *AdminConn) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd)+1:]

	switch cmd {
	case protocol.ReqAdmin:
		ac.handleReqAdmin(args)

	case protocol.ConfAdmin:
		ac.handleConfAdmin(args)

	case protocol.AddAuth:
		if !ac.authenticated {
			return
		}
		ac.handleAddAuth(args)

	case protocol.DelAuth:
		if !ac.authenticated {
			return
		}
		ac.handleDelAuth(args)

	default:
		log.Printf("no handler for command %s in '%s'", cmd, msg)
	}
}

func (ac *AdminConn) handleReqAdmin(args string) {
	var adminName string
	_, err := fmt.Sscanf(args, "%s", &adminName)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAdmin, args, err)
		ac.conn.Close()
		return
	}

	challenge, solution, err := ac.generateChallenge(adminName)
	if err != nil {
		log.Printf("could not generate challenge to authenticate '%s' as admin: %v", adminName, err)
		ac.conn.Close()
		return
	}

	ac.adminName = adminName
	ac.solution = solution

	ac.respond("%s %s", protocol.ChalAdmin, challenge)
}

func (ac *AdminConn) handleConfAdmin(args string) {
	var solution string
	_, err := fmt.Sscanf(args, "%s", &solution)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAdmin, args, err)
		ac.respond("%s %s", protocol.FailAdmin, "could not parse solution")
		ac.conn.Close()
		return
	}

	if solution == ac.solution {
		ac.authenticated = true
		ac.respond("%s", protocol.SuccAdmin)
		log.Printf("connection from %s successfully authenticated as admin '%s'", ac.conn.RemoteAddr(), ac.adminName)
	} else {
		ac.respond("%s", protocol.FailAdmin)
		ac.conn.Close()
		log.Printf("connection from %s failed to authenticate as admin '%s'", ac.conn.RemoteAddr(), ac.adminName)
	}
}

func (ac *AdminConn) handleAddAuth(args string) {
	var name string
	var pubkey string
	_, err := fmt.Sscanf(args, "%s %s", &name, &pubkey)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.AddAuth, args, err)
		ac.conn.Close()
		return
	}

	err = ac.db.AddUser(name, pubkey)
	if err != nil {
		log.Println(err)
		ac.respond("failed to add entry: %v", err)
		return
	}

	ac.respond("ok")
	log.Printf("admin '%s' (%s) added auth entry '%s' (pubkey '%s')", ac.adminName, ac.conn.RemoteAddr(), name, pubkey)
}

func (ac *AdminConn) handleDelAuth(args string) {
	var name string
	_, err := fmt.Sscanf(args, "%s", &name)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.DelAuth, args, err)
		ac.conn.Close()
		return
	}

	err = ac.db.DelUser(name)
	if err != nil {
		log.Println(err)
		ac.respond("failed to delete entry: %v", err)
		return
	}

	ac.respond("ok")
	log.Printf("admin '%s' (%s) deleted auth entry '%s'", ac.adminName, ac.conn.RemoteAddr(), name)
}
