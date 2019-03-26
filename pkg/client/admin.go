package client

import (
	"fmt"
	"os"
	"strings"

	"github.com/sauerbraten/maitred/pkg/auth"
	"github.com/sauerbraten/maitred/pkg/protocol"
)

type AdminClient struct {
	Client
	isAdminConnection bool
	ids               *protocol.IDCycle
	callbacks         map[uint32]func(string)
}

func NewAdmin(client Client) *AdminClient {
	return &AdminClient{
		Client:    client,
		ids:       new(protocol.IDCycle),
		callbacks: map[uint32]func(string){},
	}
}

func (c *AdminClient) AddAuth(name, pubkey string, callback func(string)) {
	reqID := c.ids.Next()
	err := c.Send("%s %d %s %s", protocol.AddAuth, reqID, name, pubkey)
	if err != nil {
		callback(err.Error())
		return
	}
	c.callbacks[reqID] = callback
}

func (c *AdminClient) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.SuccReg:
		c.handleSuccReg(args)

	case protocol.ChalAdmin:
		c.handleChalAdmin(args)

	case protocol.SuccAdmin:
		c.handleSuccAdmin(args)

	case protocol.FailAdmin:
		c.handleFailAdmin(args)

	case protocol.SuccAddAuth:
		c.handleSuccAddAuth(args)

	case protocol.FailAddAuth:
		c.handleFailAddAuth(args)

	default:
		c.Client.Handle(msg)
	}
}

func (c *AdminClient) handleSuccReg(args string) {
	c.Client.Handle(protocol.SuccReg)

	if _, ok := os.LookupEnv("STATSAUTH_ADMIN_KEY"); ok {
		c.Log("trying to upgrade stats server connection")
		err := c.Client.Send("%s %s", protocol.ReqAdmin, os.Getenv("STATSAUTH_ADMIN_NAME"))
		if err != nil {
			c.Log("could not request admin challenge: %v", err)
			return
		}
	}
}

func (c *AdminClient) handleChalAdmin(args string) {
	var challenge string
	_, err := fmt.Sscanf(args, "%s", &challenge)
	if err != nil {
		c.Log("malformed %s message from stats server: '%s': %v", protocol.ChalAdmin, args, err)
		return
	}

	answer, err := auth.Solve(challenge, os.Getenv("STATSAUTH_ADMIN_KEY"))
	if err != nil {
		c.Log("could not solve admin challenge: %v", err)
		return
	}

	err = c.Client.Send("%s %s", protocol.ConfAdmin, answer)
	if err != nil {
		c.Log("could not send answer to admin challenge: %v", err)
		return
	}
}

func (c *AdminClient) handleSuccAdmin(args string) {
	c.isAdminConnection = true
	c.Log("successfully upgraded stats server connection to admin connection")
}

func (c *AdminClient) handleFailAdmin(args string) {
	c.Log("upgrading stats server connection to admin connection failed")
}

func (c *AdminClient) handleSuccAddAuth(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		c.Log("malformed %s message from stats server: '%s': %v", protocol.SuccAddAuth, args, err)
		return
	}

	if callback, ok := c.callbacks[reqID]; ok {
		callback("")
	}
}

func (c *AdminClient) handleFailAddAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var reqID uint32
	_, err := fmt.Fscanf(r, "%d", &reqID)
	if err != nil {
		c.Log("malformed %s message from stats server: '%s': %v", protocol.FailAddAuth, args, err)
		return
	}
	reason := args[len(args)-r.Len():] // unread portion of args
	reason = strings.TrimSpace(reason)

	if callback, ok := c.callbacks[reqID]; ok {
		callback(reason)
	}
}
