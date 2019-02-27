package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/pkg/protocol"

	"github.com/sauerbraten/maitred/pkg/auth"
)

var (
	adminName string
	privkey   string
	address   string
)

func init() {
	adminName = os.Getenv("MAITRED_AUTHNAME")
	if adminName == "" {
		fmt.Fprintln(os.Stderr, "MAITRED_AUTHNAME environment variable not set")
		os.Exit(-1)
	}

	privkey = os.Getenv("MAITRED_AUTHKEY")
	if privkey == "" {
		fmt.Fprintln(os.Stderr, "MAITRED_AUTHKEY environment variable not set")
		os.Exit(-1)
	}

	address = os.Getenv("MAITRED_ADDRESS")
	if address == "" {
		fmt.Fprintln(os.Stderr, "MAITRED_ADDRESS environment variable not set")
		os.Exit(-1)
	}
}

func usage() {
	fmt.Println("Usage: manage addauth <name> <pubkey>")
	fmt.Println("       manage delauth <name>")
	os.Exit(1)
}

func main() {
	switch len(os.Args) {
	case 1, 2:
		usage()
	case 3:
		if os.Args[1] != protocol.DelAuth {
			usage()
		}
		deleteUser(os.Args[2])
	case 4:
		if os.Args[1] != protocol.AddAuth {
			usage()
		}
		addUser(os.Args[2], os.Args[3])
	}
}

func addUser(name, pubkey string) {
	exec(protocol.AddAuth, name, pubkey)
}

func deleteUser(name string) {
	exec(protocol.DelAuth, name)
}

func exec(cmd string, args ...string) {
	conn := connect()
	scanner := bufio.NewScanner(conn)

	authenticate(conn, scanner)

	err := send(conn, "%s %s", cmd, strings.Join(args, " "))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(4)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, scanner.Err())
		os.Exit(4)
	}
	response := scanner.Text()

	if response != "ok" {
		fmt.Fprintln(os.Stderr, "error running", cmd, "command:", response)
		os.Exit(4)
	}
}

func connect() *net.TCPConn {
	raddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	return conn
}

func send(conn *net.TCPConn, format string, args ...interface{}) error {
	_, err := conn.Write([]byte(fmt.Sprintf(format+"\n", args...)))
	return err
}

func authenticate(conn *net.TCPConn, scanner *bufio.Scanner) {
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	err := send(conn, "%s %s", protocol.ReqAdmin, adminName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, scanner.Err())
		os.Exit(3)
	}
	var challenge string
	_, err = fmt.Sscanf(scanner.Text(), protocol.ChalAdmin+" %s", &challenge)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	solution, err := auth.Solve(challenge, privkey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	err = send(conn, "%s %s", protocol.ConfAdmin, solution)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, scanner.Err())
		os.Exit(3)
	}
	response := scanner.Text()

	if response != protocol.SuccAdmin {
		fmt.Fprintln(os.Stderr, "could not authenticate as admin:", response)
		os.Exit(3)
	}
}
