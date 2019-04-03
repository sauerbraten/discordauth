package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/chef/pkg/ips"
	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/protocol/role"

	"github.com/sauerbraten/maitred/pkg/auth"
	"github.com/sauerbraten/maitred/pkg/protocol"
)

type VanillaClient struct {
	raddr      *net.TCPAddr
	listenPort int
	bans       *bans.BanManager

	conn       *net.TCPConn
	inc        chan<- string
	pingFailed bool

	*auth.RemoteProvider
	authInc chan<- string
	authOut <-chan string

	onReconnect func() // executed when the game server reconnects to the remote master server
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func NewVanilla(addr string, listenPort int, bans *bans.BanManager, authRole role.ID, onReconnect func()) (*VanillaClient, <-chan string, error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("master (%s): error resolving server address (%s): %v", raddr, addr, err)
	}

	inc := make(chan string)
	authInc, authOut := make(chan string), make(chan string)

	c := &VanillaClient{
		raddr:      raddr,
		listenPort: listenPort,
		bans:       bans,

		inc: inc,

		RemoteProvider: auth.NewRemoteProvider(authInc, authOut, authRole),
		authInc:        authInc,
		authOut:        authOut,

		onReconnect: onReconnect,
	}

	err = c.connect()
	if err != nil {
		return nil, nil, err
	}

	return c, inc, nil
}

func (c *VanillaClient) connect() error {
	conn, err := net.DialTCP("tcp", nil, c.raddr)
	if err != nil {
		return fmt.Errorf("master (%s): error connecting to master server: %v", c.raddr, err)
	}

	c.conn = conn

	sc := bufio.NewScanner(c.conn)

	go func() {
		for sc.Scan() {
			c.inc <- sc.Text()
		}
		if err := sc.Err(); err != nil {
			log.Println(err)
		} else {
			c.Log("EOF while scanning input")
			if !c.pingFailed {
				c.reconnect(io.EOF)
			}
		}
	}()

	go func() {
		for msg := range c.authOut {
			err := c.Send(msg)
			if err != nil {
				c.Log("remote auth: %v", err)
			}
		}
	}()

	c.Register()

	return nil
}

func (c *VanillaClient) reconnect(err error) {
	c.conn = nil

	try, maxTries := 1, 10
	for err != nil && try <= maxTries {
		time.Sleep(time.Duration(try) * 30 * time.Second)
		c.Log("trying to reconnect (attempt %d)", try)

		err = c.connect()
		try++
	}

	if err == nil {
		c.Log("reconnected successfully")
		c.onReconnect()
	} else {
		c.Log("could not reconnect: %v", err)
	}
}

func (c *VanillaClient) Log(format string, args ...interface{}) {
	log.Println(fmt.Sprintf("master (%s):", c.raddr), fmt.Sprintf(format, args...))
}

func (c *VanillaClient) Register() {
	if c.pingFailed {
		return
	}
	c.Log("registering")
	err := c.Send("%s %d", protocol.RegServ, c.listenPort)
	if err != nil {
		c.Log("registration failed: %v", err)
		return
	}
}

func (c *VanillaClient) Send(format string, args ...interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("master (%s): not connected", c.raddr)
	}

	err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	if err != nil {
		c.Log("write failed: %v", err)
		return err
	}
	_, err = c.conn.Write([]byte(fmt.Sprintf(format+"\n", args...)))
	if err != nil {
		c.Log("write failed: %v", err)
	}
	return err
}

func (c *VanillaClient) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.SuccReg:
		c.Log("registration succeeded")

	case protocol.FailReg:
		c.Log("registration failed: %v", args)
		if args == "failed pinging server" {
			c.Log("disabling reconnecting")
			c.pingFailed = true // stop trying
		}

	case protocol.ClearBans:
		c.bans.ClearGlobalBans()

	case protocol.AddBan:
		c.handleAddGlobalBan(args)

	case protocol.ChalAuth, protocol.SuccAuth, protocol.FailAuth:
		c.authInc <- msg

	default:
		c.Log("received and not handled: %v", msg)
	}
}

func (c *VanillaClient) handleAddGlobalBan(args string) {
	var ip string
	_, err := fmt.Sscanf(args, "%s", &ip)
	if err != nil {
		c.Log("malformed %s message from game server: '%s': %v", protocol.AddBan, args, err)
		return
	}

	network := ips.GetSubnet(ip)

	c.bans.AddBan(network, fmt.Sprintf("banned by master server (%s)", c.raddr), time.Time{}, true)
}
