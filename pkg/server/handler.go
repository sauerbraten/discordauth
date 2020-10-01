package server

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/extinfo"

	"github.com/sauerbraten/maitred/v2/internal/db"
	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
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
	*protocol.Conn
	db   *db.Database
	stop <-chan struct{}

	client               client // the game server this handler is for
	pendingRequests      map[uint]*request
	authedUsersByRequest map[uint]string

	adminReqID uint
	isAdmin    bool
}

func newHandler(conn *protocol.Conn, db *db.Database, stop <-chan struct{}) *handler {
	return &handler{
		Conn:                 conn,
		db:                   db,
		stop:                 stop,
		client:               client{id: -1},
		pendingRequests:      map[uint]*request{},
		authedUsersByRequest: map[uint]string{},
	}
}

func (h *handler) generateChallenge(name string) (challenge, solution string, err error) {
	pubkey, err := h.db.GetPublicKey(name)
	if err != nil {
		return "", "", err
	}

	challenge, solution, err = auth.GenerateChallenge(pubkey)
	if err != nil {
		err = fmt.Errorf("could not generate challenge using pubkey %s of %s: %v", pubkey, name, err)
	}
	return
}

func (h *handler) makeAndRememberChallenge(reqID uint, name string) (challenge string, err error) {
	var solution string
	challenge, solution, err = h.generateChallenge(name)
	if err != nil {
		return
	}

	h.pendingRequests[reqID] = &request{
		id:       reqID,
		name:     name,
		solution: solution,
	}

	return
}

func (h *handler) run() {
	for {
		select {
		case msg, ok := <-h.Incoming():
			if !ok {
				log.Println(h.Conn.RemoteAddr(), "closed the connection")
				return
			}
			h.handle(msg)
		case <-h.stop:
			log.Println("closing connection to", h.RemoteAddr())
			h.Close()
			return
		}
	}
}

func (h *handler) handle(msg string) {
	if msg == "" {
		log.Printf("server %s sent empty message", h.RemoteAddr())
		h.Close()
		return
	}

	cmd := strings.Split(msg, " ")[0]
	if len(cmd) >= len(msg) {
		log.Printf("server %s sent message without arguments", h.RemoteAddr())
		h.Close()
		return
	}
	args := msg[len(cmd)+1:]

	switch cmd {
	case protocol.RegServ:
		h.handleRegServ(args)
		return

	case protocol.ReqAdmin:
		h.handleReqAdmin(args)
		return

	case protocol.ConfAdmin:
		h.handleConfAdmin(args)
		return

	default:
		// unknown clients have to register themselves before doing anything else
		if h.client.id < 0 && !h.isAdmin {
			log.Printf("unknown client %s sent disallowed command '%s' (args: '%s')", h.RemoteAddr(), cmd, args)
			h.Close()
			return
		}
	}

	switch cmd {
	case protocol.ReqAuth:
		h.handleReqAuth(args)

	case protocol.ConfAuth:
		h.handleConfAuth(args)

	case protocol.Stats:
		h.handleStats(args)

	case protocol.Lookup:
		h.handleLookup(args)

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
		h.Send("%s %s", protocol.FailReg, "invalid port")
		return
	}

	serverTCPAddr, _ := net.ResolveTCPAddr(h.RemoteAddr().Network(), h.RemoteAddr().String())

	serverUDPAddr := &net.UDPAddr{
		IP:   serverTCPAddr.IP,
		Port: port,
	}

	h.client.addr, err = net.ResolveUDPAddr(serverUDPAddr.Network(), serverUDPAddr.String())
	if err != nil {
		log.Printf("error resolving UDP address %s: %v", serverUDPAddr, err)
		h.Send("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	gamehandler, err := extinfo.NewServer(*h.client.addr, 10*time.Second)
	if err != nil {
		log.Printf("error resolving extinfo UDP address of %s: %v", h.client.addr, err)
		h.Send("%s %s", protocol.FailReg, "failed resolving ip")
		return
	}

	info, err := gamehandler.GetBasicInfo()
	if err != nil {
		log.Printf("error querying basic info of %s: %v", h.client.addr, err)
		h.Send("%s %s", protocol.FailReg, "failed pinging server")
		return
	}

	mod, err := gamehandler.GetServerMod()
	if err != nil {
		log.Printf("error querying server mod ID of %s: %v", h.client.addr, err)
		// not a problem, don't fail registration
	}

	h.client.id = h.db.GetServerID(h.client.addr.IP.String(), h.client.addr.Port, info.Description, mod)
	h.db.UpdateServerLastActive(h.client.id)

	h.Send("%s", protocol.SuccReg)
}

func (h *handler) handleReqAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for r.Len() > 0 {
		var reqID uint
		var name string
		_, err := fmt.Fscanf(r, "%d %s", &reqID, &name)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAuth, args, err)
			return
		}

		log.Printf("generating challenge for '%s' (request %d)", name, reqID)

		challenge, err := h.makeAndRememberChallenge(reqID, name)
		if err != nil {
			log.Printf("could not generate challenge for request %d (%s): %v", reqID, name, err)
			h.Send("%s %d", protocol.FailAuth, reqID)
			return
		}

		h.Send("%s %d %s", protocol.ChalAuth, reqID, challenge)
	}
}

func (h *handler) handleConfAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for r.Len() > 0 {
		var reqID uint
		var answer string
		_, err := fmt.Fscanf(r, "%d %s", &reqID, &answer)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAuth, args, err)
			return
		}

		req, ok := h.pendingRequests[reqID]

		if ok && answer == req.solution {
			h.authedUsersByRequest[reqID] = req.name
			h.Send("%s %d", protocol.SuccAuth, reqID)
			log.Println("request", reqID, "completed successfully")
			err := h.db.UpdateUserLastAuthed(req.name)
			if err != nil {
				log.Println(err)
			}
		} else {
			h.Send("%s %d", protocol.FailAuth, reqID)
			log.Println("request", reqID, "failed")
		}

		delete(h.pendingRequests, reqID)
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
			reqID     uint
			name      string
			frags     int64
			deaths    int64
			damage    int64
			potential int64
			flags     int64
		)
		_, err := fmt.Fscanf(r, "%d %s %d %d %d %d %d", &reqID, &name, &frags, &deaths, &damage, &potential, &flags)
		if err != nil {
			log.Printf("error scanning user stats: %v", err)
			return
		}

		if authedName, ok := h.authedUsersByRequest[reqID]; !ok || authedName != name {
			log.Printf("ignoring stats for unauthenticated user '%s' (request %d)", name, reqID)
			h.Send("%s %d %s", protocol.FailStats, reqID, "not authenticated")
			continue
		}

		err = h.db.AddStats(gameID, name, frags, deaths, damage, potential, flags)
		if err != nil {
			log.Printf("failed to save stats for '%s' (request %d) in database: %v", name, reqID, err)
			h.Send("%s %d %s", protocol.FailStats, reqID, "internal error")
			continue
		}

		h.Send("%s %d", protocol.SuccStats, reqID)
	}
}

func (h *handler) handleLookup(args string) {
	var reqID uint
	var name string
	_, err := fmt.Sscanf(args, "%d %s", &reqID, &name)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.Lookup, args, err)
		h.Send("%s %d %s", protocol.FailLookup, reqID, err.Error())
		h.Close()
		return
	}

	exists, err := h.db.UserExists(name)
	if err != nil {
		log.Printf("failed to look up '%s' (request %d) in database: %v", name, reqID, err)
		h.Send("%s %d %s", protocol.FailLookup, reqID, "internal error")
		h.Close()
		return
	}

	if exists {
		h.Send("%s %d", protocol.SuccLookup, reqID)
	} else {
		h.Send("%s %d %s", protocol.FailLookup, reqID, "user does not exist")
	}
}

func (h *handler) handleReqAdmin(args string) {
	var reqID uint
	var adminName string
	_, err := fmt.Sscanf(args, "%d %s", &reqID, &adminName)
	if err != nil {
		log.Printf("malformed %s message from client: '%s': %v", protocol.ReqAdmin, args, err)
		h.Send("%s %d %s", protocol.FailAdmin, reqID, err.Error())
		h.Close()
		return
	}

	challenge, err := h.makeAndRememberChallenge(reqID, adminName)
	if err != nil {
		log.Printf("could not generate challenge to authenticate '%s' as admin: %v", adminName, err)
		h.Send("%s %d %s", protocol.FailAdmin, reqID, err.Error())
		h.Close()
		return
	}

	h.adminReqID = reqID

	h.Send("%s %d %s", protocol.ChalAdmin, reqID, challenge)
}

func (h *handler) handleConfAdmin(args string) {
	log.Println("confirming admin request:", args)

	var reqID uint
	var answer string

	fail := func(reason string) {
		log.Println("request", reqID, "failed:", reason)
		h.Send("%s %d %s", protocol.FailAdmin, reqID, reason)
		h.Close()
	}

	_, err := fmt.Sscanf(args, "%d %s", &reqID, &answer)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAdmin, args, err)
		fail("could not parse solution")
		return
	}

	defer delete(h.pendingRequests, reqID)

	if reqID != h.adminReqID {
		fail("not an admin request")
		return
	}

	req, ok := h.pendingRequests[reqID]
	if !ok {
		fail("unknown request")
		return
	}

	if answer != req.solution {
		log.Printf("connection from %s failed to authenticate as admin '%s'", h.RemoteAddr(), req.name)
		fail("wrong answer")
		return
	}

	log.Println("request", reqID, "completed successfully")
	log.Printf("connection from %s successfully authenticated as admin '%s'", h.RemoteAddr(), req.name)
	h.isAdmin = true
	h.authedUsersByRequest[reqID] = req.name
	h.Send("%s %d", protocol.SuccAdmin, reqID)

	err = h.db.UpdateUserLastAuthed(req.name)
	if err != nil {
		log.Println(err)
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
		h.Close()
		return
	}

	err = h.db.AddUser(name, pubkey)
	if err != nil {
		log.Println(err)
		h.Send("%s %d %v", protocol.FailAddAuth, reqID, err)
		return
	}

	h.Send("%s %d", protocol.SuccAddAuth, reqID)
	log.Printf("admin '%s' (%s) added auth entry '%s' (pubkey '%s')", h.authedUsersByRequest[h.adminReqID], h.RemoteAddr(), name, pubkey)
}

func (h *handler) handleDelAuth(args string) {
	var (
		reqID uint32
		name  string
	)
	_, err := fmt.Sscanf(args, "%d %s", &reqID, &name)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", protocol.DelAuth, args, err)
		h.Close()
		return
	}

	exists, err := h.db.UserExists(name)
	if err != nil {
		log.Println(err)
		h.Send("%s %d %v", protocol.FailDelAuth, reqID, err)
		return
	}
	if !exists {
		h.Send("%s %d user doesn't exist", protocol.FailDelAuth, reqID)
		return
	}

	err = h.db.DelUser(name)
	if err != nil {
		log.Println(err)
		h.Send("%s %d %v", protocol.FailDelAuth, reqID, err)
		return
	}

	h.Send("%s %d", protocol.SuccDelAuth, reqID)
	log.Printf("admin '%s' (%s) deleted auth entry '%s'", h.authedUsersByRequest[h.adminReqID], h.RemoteAddr(), name)
}
