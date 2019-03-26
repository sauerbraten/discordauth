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
	"github.com/sauerbraten/maitred/pkg/auth"
	"github.com/sauerbraten/maitred/pkg/protocol"

	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
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
			log.Printf("master (%s): EOF while scanning input", c.raddr)
			if !c.pingFailed {
				c.reconnect(io.EOF)
			}
		}
	}()

	go func() {
		for msg := range c.authOut {
			err := c.Send(msg)
			if err != nil {
				log.Printf("master (%s): remote auth: %v", c.raddr, err)
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
		log.Printf("master (%s): trying to reconnect (attempt %d)", c.raddr, try)

		err = c.connect()
		try++
	}

	if err == nil {
		log.Printf("master (%s): reconnected successfully", c.raddr)
		c.onReconnect()
	} else {
		log.Printf("master (%s): could not reconnect: %v", c.raddr, err)
	}
}

func (c *VanillaClient) Register() {
	if c.pingFailed {
		return
	}
	log.Printf("master (%s): registering", c.raddr)
	err := c.Send("%s %d", protocol.RegServ, c.listenPort)
	if err != nil {
		log.Printf("master (%s): registration failed: %v", c.raddr, err)
		return
	}
}

func (c *VanillaClient) Send(format string, args ...interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("master (%s): not connected", c.raddr)
	}

	err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	if err != nil {
		log.Printf("master (%s): write failed: %v", c.raddr, err)
		return err
	}
	_, err = c.conn.Write([]byte(fmt.Sprintf(format+"\n", args...)))
	if err != nil {
		log.Printf("master (%s): write failed: %v", c.raddr, err)
	}
	return err
}

func (c *VanillaClient) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.SuccReg:
		log.Printf("master (%s): registration succeeded", c.raddr)

	case protocol.FailReg:
		log.Printf("master (%s): registration failed: %v", c.raddr, args)
		if args == "failed pinging server" {
			log.Printf("master (%s): disabling reconnecting", c.raddr)
			c.pingFailed = true // stop trying
		}

	case protocol.ClearBans:
		c.bans.ClearGlobalBans()

	case protocol.AddBan:
		c.handleAddGlobalBan(args)

	case protocol.ChalAuth, protocol.SuccAuth, protocol.FailAuth:
		c.authInc <- msg

	default:
		log.Printf("master (%s): received and not handled: %v", c.raddr, msg)
	}
}

func (c *VanillaClient) handleAddGlobalBan(args string) {
	var ip string
	_, err := fmt.Sscanf(args, "%s", &ip)
	if err != nil {
		log.Printf("master (%s): malformed %s message from game server: '%s': %v", c.raddr, protocol.AddBan, args, err)
		return
	}

	network := ips.GetSubnet(ip)

	c.bans.AddBan(network, fmt.Sprintf("banned by master server (%s)", c.raddr), time.Time{}, true)
}
