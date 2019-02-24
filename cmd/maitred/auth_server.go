package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/internal/db"
	"github.com/sauerbraten/maitred/pkg/auth"
)

// master server protocol constants
// exported constants can be sent to the master server
const (
	//registerServer         = "regserv"
	//registrationSuccessful = "succreg"
	//registrationFailed     = "failreg"

	//addBan    = "addgban"
	//clearBans = "cleargbans"

	requestAuthChallenge = "reqauth"
	challengeAuth        = "chalauth"
	confirmAuthAnswer    = "confauth"
	authSuccesful        = "succauth"
	authFailed           = "failauth"
)

// request holds the data we need to remember between
// generating a challenge and checking the response.
type request struct {
	id       uint
	name     string
	solution string
}

type AuthServer struct {
	conn        *net.TCPConn
	in          *bufio.Scanner
	db          *db.Database
	pendingByID map[uint]*request
}

func Listen(listenAddr *net.TCPAddr, db *db.Database, stop <-chan struct{}) {
	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Printf("error starting to listen on %s: %v", listenAddr, err)
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("error accepting connection: %v", err)
		}

		s := &AuthServer{
			conn:        conn,
			in:          bufio.NewScanner(conn),
			db:          db,
			pendingByID: map[uint]*request{},
		}

		go s.run(stop)
	}
}

func (s *AuthServer) run(stop <-chan struct{}) {
	incoming := make(chan string)
	go func() {
		for s.in.Scan() {
			incoming <- s.in.Text()
		}
		if err := s.in.Err(); err != nil {
			log.Println(err)
		}
	}()

	for {
		select {
		case msg := <-incoming:
			s.handle(msg)
		case <-stop:
			log.Println("received stop signal")
			s.conn.Close()
			return
		}
	}
}

func (s *AuthServer) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd)+1:]

	switch cmd {
	case requestAuthChallenge:
		s.handleRequestAuthChallenge(args)

	case confirmAuthAnswer:
		s.handleConfirmAuthAnswer(args)

	default:
		log.Printf("no handler for command %s in '%s'", cmd, msg)
	}
}

func (s *AuthServer) respond(format string, args ...interface{}) {
	response := fmt.Sprintf(format, args...)
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := s.conn.Write([]byte((response + "\n")))
	if err != nil {
		log.Printf("failed to send '%s': %v", response, err)
	}
}

func (s *AuthServer) handleRequestAuthChallenge(args string) {
	var (
		requestID uint
		name      string
	)
	_, err := fmt.Sscanf(args, "%d %s", &requestID, &name)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", requestAuthChallenge, args, err)
		return
	}

	log.Printf("generating challenge for '%s' (request %d)", name, requestID)

	challenge, err := s.generateChallenge(requestID, name)
	if err != nil {
		log.Printf("could not generate challenge for request %d (%s): %v", requestID, name, err)
		s.respond("%s %d", authFailed, requestID)
		return
	}

	s.respond("%s %d %s", challengeAuth, requestID, challenge)
}

func (s *AuthServer) handleConfirmAuthAnswer(args string) {
	var (
		requestID uint
		answer    string
	)
	_, err := fmt.Sscanf(args, "%d %s", &requestID, &answer)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", confirmAuthAnswer, args, err)
		return
	}

	if s.checkAnswer(requestID, answer) {
		s.respond("%s %d", authSuccesful, requestID)
		log.Println("request", requestID, "completed successfully")
	} else {
		s.respond("%s %d", authFailed, requestID)
		log.Println("request", requestID, "failed")
	}
}

func (s *AuthServer) generateChallenge(requestID uint, name string) (challenge string, err error) {
	pubkey, err := s.db.GetPublicKey(name)
	if err != nil {
		return "", err
	}

	challenge, solution, err := auth.GenerateChallenge(pubkey)
	if err != nil {
		delete(s.pendingByID, requestID)
		return "", fmt.Errorf("could not generate challenge using pubkey %s of %s: %v", pubkey, name, err)
	}

	s.pendingByID[requestID] = &request{
		id:       requestID,
		name:     name,
		solution: solution,
	}

	return challenge, nil
}

func (s *AuthServer) checkAnswer(requestID uint, answer string) bool {
	defer delete(s.pendingByID, requestID)
	req, ok := s.pendingByID[requestID]
	return ok && answer == req.solution
}
