package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sauerbraten/extinfo"

	"github.com/sauerbraten/maitred/pkg/protocol"
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

type ServerConn struct {
	*ConnHandler
	server               Server
	pendingRequests      map[uint]*request
	authedUsersByRequest map[uint]string
}

func NewServerConn(ch *ConnHandler) *ServerConn {
	return &ServerConn{
		ConnHandler:          ch,
		server:               Server{id: -1},
		pendingRequests:      map[uint]*request{},
		authedUsersByRequest: map[uint]string{},
	}
}

func (sc *ServerConn) run(stop <-chan struct{}, handle func(string)) {
	incoming := make(chan string)
	go func() {
		for sc.in.Scan() {
			incoming <- sc.in.Text()
		}
		if err := sc.in.Err(); err != nil {
			log.Println(err)
		}
		close(incoming)
	}()

	for {
		select {
		case msg, ok := <-incoming:
			if !ok {
				log.Println(sc.conn.RemoteAddr(), "closed the connection")
				return
			}
			handle(msg)
		case <-stop:
			log.Println("closing connection to", sc.conn.RemoteAddr())
			sc.conn.Close()
			return
		}
	}
}

func (sc *ServerConn) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd)+1:]

	// unregistered servers have to register themselves before doing anything else
	if sc.server.id < 0 && cmd != protocol.RegServ {
		log.Printf("unregistered server %s sent disallowed command '%s' (args: '%s')", sc.conn.RemoteAddr(), cmd, args)
		sc.conn.Close()
		return
	}

	switch cmd {
	case protocol.RegServ:
		sc.handleRegServ(args)

	case protocol.ReqAuth:
		sc.handleReqAuth(args)

	case protocol.ConfAuth:
		sc.handleConfAuth(args)

	case protocol.Stats:
		sc.handleStats(args)

	default:
		log.Printf("no handler for command %s in '%s'", cmd, msg)
	}
}

func (sc *ServerConn) handleRegServ(args string) {
	var port int
	_, err := fmt.Sscanf(args, "%d", &port)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.RegServ, args, err)
		sc.respond("%s %s", protocol.FailReg, "invalid port")
		return
	}

	ip, _, err := net.SplitHostPort(sc.conn.RemoteAddr().String())
	if err != nil {
		log.Printf("error extracting IP from connection to %s: %v", sc.conn.RemoteAddr(), err)
		sc.respond("%s %s", protocol.FailReg, "internal error")
		return
	}

	sc.server.addr, err = net.ResolveUDPAddr("udp", ip+":"+strconv.Itoa(port))
	if err != nil {
		log.Printf("error resolving UDP address %s: %v", ip+":"+strconv.Itoa(port), err)
		sc.respond("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	gameServer, err := extinfo.NewServer(*sc.server.addr, 10*time.Second)
	if err != nil {
		log.Printf("error resolving extinfo UDP address of %s: %v", sc.server.addr, err)
		sc.respond("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	info, err := gameServer.GetBasicInfo()
	if err != nil {
		log.Printf("error querying basic info of %s: %v", sc.server.addr, err)
		sc.respond("%s %s", protocol.FailReg, "failed pinging server")
		return
	}

	mod, err := gameServer.GetServerMod()
	if err != nil {
		log.Printf("error querying server mod ID of %s: %v", sc.server.addr, err)
		// not a problem, don't fail registration
	}

	sc.server.id = sc.db.GetServerID(sc.server.addr.IP.String(), sc.server.addr.Port, info.Description, mod)
	sc.db.UpdateServerLastActive(sc.server.id)

	sc.respond("%s", protocol.SuccReg)
}

func (sc *ServerConn) handleReqAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for r.Len() > 0 {
		var requestID uint
		var name string
		_, err := fmt.Fscanf(r, "%d %s", &requestID, &name)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAuth, args, err)
			return
		}

		log.Printf("generating challenge for '%s' (request %d)", name, requestID)

		challenge, err := sc.generateChallenge(requestID, name)
		if err != nil {
			log.Printf("could not generate challenge for request %d (%s): %v", requestID, name, err)
			sc.respond("%s %d", protocol.FailAuth, requestID)
			return
		}

		sc.respond("%s %d %s", protocol.ChalAuth, requestID, challenge)
	}
}

func (sc *ServerConn) generateChallenge(requestID uint, name string) (challenge string, err error) {
	var solution string
	challenge, solution, err = sc.ConnHandler.generateChallenge(name)
	if err != nil {
		return
	}

	sc.pendingRequests[requestID] = &request{
		id:       requestID,
		name:     name,
		solution: solution,
	}

	return
}

func (sc *ServerConn) handleConfAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for r.Len() > 0 {
		var requestID uint
		var answer string
		_, err := fmt.Fscanf(r, "%d %s", &requestID, &answer)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAuth, args, err)
			return
		}

		req, ok := sc.pendingRequests[requestID]

		if ok && answer == req.solution {
			sc.authedUsersByRequest[requestID] = req.name
			sc.respond("%s %d", protocol.SuccAuth, requestID)
			log.Println("request", requestID, "completed successfully")
		} else {
			sc.respond("%s %d", protocol.FailAuth, requestID)
			log.Println("request", requestID, "failed")
		}

		delete(sc.pendingRequests, requestID)
	}
}

func (sc *ServerConn) handleStats(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var gamemode int64
	var mapname string
	_, err := fmt.Fscanf(r, "%d %s", &gamemode, &mapname)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.Stats, args, err)
		return
	}

	gameID, err := sc.db.AddGame(sc.server.id, gamemode, mapname)
	if err != nil {
		log.Println(err)
		return
	}

	for r.Len() > 0 {
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

		if authedName, ok := sc.authedUsersByRequest[requestID]; !ok || authedName != name {
			log.Printf("ignoring stats for unauthenticated user '%s' (request %d)", name, requestID)
			sc.respond("%s %d %s", protocol.FailStats, requestID, "user not authenticated")
			continue
		}

		err = sc.db.AddStats(gameID, name, frags, deaths, damage, shotDamage, flags)
		if err != nil {
			log.Printf("failed to save stats for '%s' (request %d) in database: %v", name, requestID, err)
			sc.respond("%s %d %s", protocol.FailStats, requestID, "internal error")
			continue
		}

		sc.respond("%s %d", protocol.SuccStats, requestID)
	}
}
