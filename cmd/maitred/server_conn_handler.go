package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/pkg/protocol"

	"github.com/sauerbraten/extinfo"
)

// request holds the data we need to remember between
// generating a challenge and checking the response.
type request struct {
	id       uint
	name     string
	solution string
}

type Server struct {
	id   int64
	addr *net.UDPAddr
}

type ServerConnHandler struct {
	*ConnHandler
	server               Server
	pendingRequests      map[uint]*request
	authedUsersByRequest map[uint]string
}

func NewServerConnHandler(ch *ConnHandler) *ServerConnHandler {
	return &ServerConnHandler{
		ConnHandler:          ch,
		server:               Server{id: -1},
		pendingRequests:      map[uint]*request{},
		authedUsersByRequest: map[uint]string{},
	}
}

func (sch *ServerConnHandler) run(stop <-chan struct{}, handle func(string)) {
	incoming := make(chan string)
	go func() {
		for sch.in.Scan() {
			incoming <- sch.in.Text()
		}
		if err := sch.in.Err(); err != nil {
			log.Println(err)
		}
		close(incoming)
	}()

	for {
		select {
		case msg, ok := <-incoming:
			if !ok {
				log.Println(sch.conn.RemoteAddr(), "closed the connection")
				return
			}
			handle(msg)
		case <-stop:
			log.Println("closing connection to", sch.conn.RemoteAddr())
			sch.conn.Close()
			return
		}
	}
}

func (sch *ServerConnHandler) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd)+1:]

	// unregistered servers have to register themselves before doing anything else
	if sch.server.id < 0 {
		if cmd != protocol.RegServ {
			log.Printf("unregistered server %s sent disallowed command '%s' (args: '%s')", sch.conn.RemoteAddr(), cmd, args)
			sch.conn.Close()
			return
		}
		sch.handleRegisterServer(args)
	} else {
		switch cmd {
		case protocol.RegServ:
			sch.handleRegisterServer(args)

		case protocol.ReqAuth:
			sch.handleRequestAuthChallenge(args)

		case protocol.ConfAuth:
			sch.handleConfirmAuthAnswer(args)

		case protocol.Stats:
			sch.handleStats(args)

		default:
			log.Printf("no handler for command %s in '%s'", cmd, msg)
		}
	}
}

func (sch *ServerConnHandler) handleRegisterServer(args string) {
	var port int
	_, err := fmt.Sscanf(args, "%d", &port)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.RegServ, args, err)
		sch.respond("%s %s", protocol.FailReg, "invalid port")
		return
	}

	ip, _, err := net.SplitHostPort(sch.conn.RemoteAddr().String())
	if err != nil {
		log.Printf("error extracting IP from connection to %s: %v", sch.conn.RemoteAddr(), err)
		sch.respond("%s %s", protocol.FailReg, "internal error")
		return
	}

	sch.server.addr, err = net.ResolveUDPAddr("udp", ip+":"+strconv.Itoa(port))
	if err != nil {
		log.Printf("error resolving UDP address %s: %v", ip+":"+strconv.Itoa(port), err)
		sch.respond("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	// identify server
	gameServer, err := extinfo.NewServer(*sch.server.addr, 10*time.Second)
	if err != nil {
		log.Printf("error resolving extinfo UDP address of %s: %v", sch.server.addr, err)
		sch.respond("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	info, err := gameServer.GetBasicInfo()
	if err != nil {
		log.Printf("error querying basic info of %s: %v", sch.server.addr, err)
		sch.respond("%s %s", protocol.FailReg, "failed pinging server")
		return
	}

	mod, err := gameServer.GetServerMod()
	if err != nil {
		log.Printf("error querying server mod ID of %s: %v", sch.server.addr, err)
		// not a problem, don't fail registration
	}

	sch.server.id = sch.db.GetServerID(sch.server.addr.IP.String(), sch.server.addr.Port, info.Description, mod)
	sch.db.UpdateServerLastActive(sch.server.id)

	sch.respond("%s", protocol.SuccReg)
}

func (sch *ServerConnHandler) handleRequestAuthChallenge(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for len(args) > 0 {
		var requestID uint
		var name string
		_, err := fmt.Fscanf(r, "%d %s", &requestID, &name)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAuth, args, err)
			return
		}

		log.Printf("generating challenge for '%s' (request %d)", name, requestID)

		challenge, err := sch.generateChallenge(requestID, name)
		if err != nil {
			log.Printf("could not generate challenge for request %d (%s): %v", requestID, name, err)
			sch.respond("%s %d", protocol.FailAuth, requestID)
			return
		}

		sch.respond("%s %d %s", protocol.ChalAuth, requestID, challenge)
	}
}

func (sch *ServerConnHandler) generateChallenge(requestID uint, name string) (challenge string, err error) {
	var solution string
	challenge, solution, err = sch.ConnHandler.generateChallenge(name)
	if err != nil {
		return
	}

	sch.pendingRequests[requestID] = &request{
		id:       requestID,
		name:     name,
		solution: solution,
	}

	return
}

func (sch *ServerConnHandler) handleConfirmAuthAnswer(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for len(args) > 0 {
		var requestID uint
		var answer string
		_, err := fmt.Fscanf(r, "%d %s", &requestID, &answer)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAuth, args, err)
			return
		}

		req, ok := sch.pendingRequests[requestID]

		if ok && answer == req.solution {
			sch.authedUsersByRequest[requestID] = req.name
			sch.respond("%s %d", protocol.SuccAuth, requestID)
			log.Println("request", requestID, "completed successfully")
		} else {
			sch.respond("%s %d", protocol.FailAuth, requestID)
			log.Println("request", requestID, "failed")
		}

		delete(sch.pendingRequests, requestID)
	}
}

func (sch *ServerConnHandler) handleStats(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var gamemode int64
	var mapname string
	_, err := fmt.Fscanf(r, "%d %s", &gamemode, &mapname)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.Stats, args, err)
		return
	}

	gameID, err := sch.db.AddGame(sch.server.id, gamemode, mapname)
	if err != nil {
		log.Println(err)
		return
	}

	for len(args) > 0 {
		var (
			requestID  uint
			name       string
			frags      int64
			deaths     int64
			damage     int64
			shotDamage int64
			flags      int64
		)
		_, err := fmt.Fscanf(r, "%d %s %d %d %d %d %d", &requestID, &name, &frags, &deaths, &damage, &shotDamage, &flags)
		if err != nil {
			log.Printf("error scanning user stats: %v", err)
			return
		}

		if authedName, ok := sch.authedUsersByRequest[requestID]; !ok || authedName != name {
			log.Printf("ignoring stats for unauthenticated user '%s' (request %d)", name, requestID)
			sch.respond("%s %d %s %s", protocol.FailStats, requestID, name, "user not authenticated")
			continue
		}

		err = sch.db.AddStats(gameID, name, frags, deaths, damage, shotDamage, flags)
		if err != nil {
			log.Printf("failed to save stats for '%s' (request %d) in database: %v", name, requestID, err)
			sch.respond("%s %d %s %s", protocol.FailStats, requestID, name, "internal error")
			continue
		}

		sch.respond("%s %d %s", protocol.SuccStats, requestID, name)
	}
}
