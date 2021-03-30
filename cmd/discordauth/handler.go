package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

// pending holds the data we need to remember between
// generating a challenge and checking the response.
type pending struct {
	name     string
	solution string
}

type newUser struct {
	name   string
	pubkey auth.PublicKey
}

type handler struct {
	*protocol.Conn
	stop <-chan struct{}

	pendingChallenges map[uint32]pending

	pubkeyByName         func(name string) (*auth.PublicKey, bool)
	updateUserLastAuthed func(name string)
}

func newHandler(conn *protocol.Conn, stop <-chan struct{}, pubkeyByName func(name string) (*auth.PublicKey, bool), updateUserLastAuthed func(name string)) *handler {
	return &handler{
		Conn: conn,
		stop: stop,

		pendingChallenges:    map[uint32]pending{},
		pubkeyByName:         pubkeyByName,
		updateUserLastAuthed: updateUserLastAuthed,
	}
}

func (h *handler) generateChallenge(reqID uint32, name string) (challenge string, err error) {
	pubkey, ok := h.pubkeyByName(name)
	if !ok {
		return "", errors.New("user not found")
	}

	challenge, solution, err := auth.GenerateChallenge(*pubkey)
	if err != nil {
		return "", fmt.Errorf("could not generate challenge using pubkey %s of %s: %v", pubkey, name, err)
	}

	h.pendingChallenges[reqID] = pending{
		name:     name,
		solution: solution,
	}

	return challenge, nil
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
	case protocol.ReqAuth:
		h.handleReqAuth(args)

	case protocol.ConfAuth:
		h.handleConfAuth(args)

	default:
		log.Printf("unknown command %s in '%s'", cmd, msg)
		h.Close()
		return
	}
}

func (h *handler) handleReqAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))
	for r.Len() > 0 {
		var reqID uint32
		var name string
		_, err := fmt.Fscanf(r, "%d %s", &reqID, &name)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ReqAuth, args, err)
			return
		}

		log.Printf("generating challenge for '%s' (request %d)", name, reqID)

		challenge, err := h.generateChallenge(reqID, name)
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
		var reqID uint32
		var answer string
		_, err := fmt.Fscanf(r, "%d %s", &reqID, &answer)
		if err != nil {
			log.Printf("malformed %s message from game server: '%s': %v", protocol.ConfAuth, args, err)
			return
		}

		req, ok := h.pendingChallenges[reqID]

		if ok && answer == req.solution {
			go h.updateUserLastAuthed(req.name)
			h.Send("%s %d", protocol.SuccAuth, reqID)
			log.Println("request", reqID, "by", req.name, "completed successfully")
		} else {
			h.Send("%s %d", protocol.FailAuth, reqID)
			log.Println("request", reqID, "by", req.name, "failed")
		}

		delete(h.pendingChallenges, reqID)
	}
}
