package server

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

type client struct {
	id   int64
	addr *net.UDPAddr
}

type handler struct {
	*clientConn

	client               client // the game server this handler is for
	pendingRequests      map[uint]*request
	authedUsersByRequest map[uint]string

	adminName string
	solution  string
	isAdmin   bool
}

func newHandler(conn *clientConn) *handler {
	return &handler{
		clientConn:           conn,
		client:               client{id: -1},
		pendingRequests:      map[uint]*request{},
		authedUsersByRequest: map[uint]string{},
	}
}

func (h *handler) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd)+1:]

	// unregistered servers have to register themselves before doing anything else
	if h.client.id < 0 && cmd != protocol.RegServ {
		log.Printf("unregistered server %s sent disallowed command '%s' (args: '%s')", h.conn.RemoteAddr(), cmd, args)
		h.conn.Close()
		return
	}

	switch cmd {
	case protocol.RegServ:
		h.handleRegServ(args)

	case protocol.ReqAuth:
		h.handleReqAuth(args)

	case protocol.ConfAuth:
		h.handleConfAuth(args)

	case protocol.Stats:
		h.handleStats(args)

	case protocol.ReqAdmin:
		h.handleReqAdmin(args)

	case protocol.ConfAdmin:
		h.handleConfAdmin(args)

	case protocol.AddAuth:
		if h.isAdmin {
			h.handleAddAuth(args)
		}

	case protocol.DelAuth:
		if h.isAdmin {
			h.handleDelAuth(args)
		}

	default:
		log.Printf("no handler for command %s in '%s'", cmd, msg)
	}
}

func (h *handler) handleRegServ(args string) {
	var port int
	_, err := fmt.Sscanf(args, "%d", &port)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.RegServ, args, err)
		h.respond("%s %s", protocol.FailReg, "invalid port")
		return
	}

	ip, _, err := net.SplitHostPort(h.conn.RemoteAddr().String())
	if err != nil {
		log.Printf("error extracting IP from connection to %s: %v", h.conn.RemoteAddr(), err)
		h.respond("%s %s", protocol.FailReg, "internal error")
		return
	}

	h.client.addr, err = net.ResolveUDPAddr("udp", ip+":"+strconv.Itoa(port))
	if err != nil {
		log.Printf("error resolving UDP address %s: %v", ip+":"+strconv.Itoa(port), err)
		h.respond("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	gamehandler, err := extinfo.NewServer(*h.client.addr, 10*time.Second)
	if err != nil {
		log.Printf("error resolving extinfo UDP address of %s: %v", h.client.addr, err)
		h.respond("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	info, err := gamehandler.GetBasicInfo()
	if err != nil {
		log.Printf("error querying basic info of %s: %v", h.client.addr, err)
		h.respond("%s %s", protocol.FailReg, "failed pinging server")
		return
	}

	mod, err := gamehandler.GetServerMod()
	if err != nil {
		log.Printf("error querying server mod ID of %s: %v", h.client.addr, err)
		// not a problem, don't fail registration
	}

	h.client.id = h.db.GetServerID(h.client.addr.IP.String(), h.client.addr.Port, info.Description, mod)
	h.db.UpdateServerLastActive(h.client.id)

	h.respond("%s", protocol.SuccReg)
}

func (h *handler) handleReqAuth(args string) {
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

		challenge, err := h.generateChallenge(requestID, name)
		if err != nil {
			log.Printf("could not generate challenge for request %d (%s): %v", requestID, name, err)
			h.respond("%s %d", protocol.FailAuth, requestID)
			return
		}

		h.respond("%s %d %s", protocol.ChalAuth, requestID, challenge)
	}
}

func (h *handler) generateChallenge(requestID uint, name string) (challenge string, err error) {
	var solution string
	challenge, solution, err = h.clientConn.generateChallenge(name)
	if err != nil {
		return
	}

	h.pendingRequests[requestID] = &request{
		id:       requestID,
		name:     name,
		solution: solution,
	}

	return
}

func (h *handler) handleConfAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for r.Len() > 0 {
		var requestID uint
		var answer string
		_, err := fmt.Fscanf(r, "%d %s", &requestID, &answer)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAuth, args, err)
			return
		}

		req, ok := h.pendingRequests[requestID]

		if ok && answer == req.solution {
			h.authedUsersByRequest[requestID] = req.name
			h.respond("%s %d", protocol.SuccAuth, requestID)
			log.Println("request", requestID, "completed successfully")
			err := h.db.UpdateUserLastAuthed(req.name)
			if err != nil {
				log.Println(err)
			}
		} else {
			h.respond("%s %d", protocol.FailAuth, requestID)
			log.Println("request", requestID, "failed")
		}

		delete(h.pendingRequests, requestID)
	}
}

func (h *handler) handleStats(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var gamemode int64
	var mapname string
	_, err := fmt.Fscanf(r, "%d %s", &gamemode, &mapname)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.Stats, args, err)
		return
	}

	gameID, err := h.db.AddGame(h.client.id, gamemode, mapname)
	if err != nil {
		log.Println(err)
		return
	}

	for r.Len() > 0 {
		var (
			requestID uint
			name      string
			frags     int64
			deaths    int64
			damage    int64
			potential int64
			flags     int64
		)
		_, err := fmt.Fscanf(r, "%d %s %d %d %d %d %d", &requestID, &name, &frags, &deaths, &damage, &potential, &flags)
		if err != nil {
			log.Printf("error scanning user stats: %v", err)
			return
		}

		if authedName, ok := h.authedUsersByRequest[requestID]; !ok || authedName != name {
			log.Printf("ignoring stats for unauthenticated user '%s' (request %d)", name, requestID)
			h.respond("%s %d %s", protocol.FailStats, requestID, "not authenticated")
			continue
		}

		err = h.db.AddStats(gameID, name, frags, deaths, damage, potential, flags)
		if err != nil {
			log.Printf("failed to save stats for '%s' (request %d) in database: %v", name, requestID, err)
			h.respond("%s %d %s", protocol.FailStats, requestID, "internal error")
			continue
		}

		h.respond("%s %d", protocol.SuccStats, requestID)
	}
}

func (h *handler) handleReqAdmin(args string) {
	var adminName string
	_, err := fmt.Sscanf(args, "%s", &adminName)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAdmin, args, err)
		h.conn.Close()
		return
	}

	challenge, solution, err := h.clientConn.generateChallenge(adminName)
	if err != nil {
		log.Printf("could not generate challenge to authenticate '%s' as admin: %v", adminName, err)
		h.conn.Close()
		return
	}

	h.adminName = adminName
	h.solution = solution

	h.respond("%s %s", protocol.ChalAdmin, challenge)
}

func (h *handler) handleConfAdmin(args string) {
	var solution string
	_, err := fmt.Sscanf(args, "%s", &solution)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAdmin, args, err)
		h.respond("%s %s", protocol.FailAdmin, "could not parse solution")
		h.conn.Close()
		return
	}

	if solution == h.solution {
		h.isAdmin = true
		h.respond("%s", protocol.SuccAdmin)
		log.Printf("connection from %s successfully authenticated as admin '%s'", h.conn.RemoteAddr(), h.adminName)
	} else {
		h.respond("%s", protocol.FailAdmin)
		h.conn.Close()
		log.Printf("connection from %s failed to authenticate as admin '%s'", h.conn.RemoteAddr(), h.adminName)
	}
}

func (h *handler) handleAddAuth(args string) {
	var (
		reqID        uint32
		name, pubkey string
	)
	_, err := fmt.Sscanf(args, "%d %s %s", &reqID, &name, &pubkey)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.AddAuth, args, err)
		h.conn.Close()
		return
	}

	err = h.db.AddUser(name, pubkey)
	if err != nil {
		log.Println(err)
		h.respond("%s %d %v", protocol.FailAddAuth, reqID, err)
		return
	}

	h.respond("%s %d", protocol.SuccAddAuth, reqID)
	log.Printf("admin '%s' (%s) added auth entry '%s' (pubkey '%s')", h.adminName, h.conn.RemoteAddr(), name, pubkey)
}

func (h *handler) handleDelAuth(args string) {
	var name string
	_, err := fmt.Sscanf(args, "%s", &name)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.DelAuth, args, err)
		h.conn.Close()
		return
	}

	err = h.db.DelUser(name)
	if err != nil {
		log.Println(err)
		h.respond("%s %v", protocol.FailDelAuth, err)
		return
	}

	h.respond(protocol.SuccDelAuth)
	log.Printf("admin '%s' (%s) deleted auth entry '%s'", h.adminName, h.conn.RemoteAddr(), name)
}
