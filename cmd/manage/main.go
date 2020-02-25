package main

import (
	"fmt"
	"os"

	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/client"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

var (
	adminName   string
	privkey     auth.PrivateKey
	address     string
	ids         = new(protocol.IDCycle)
	adminClient *client.AdminClient
)

func init() {
	adminName = os.Getenv("MAITRED_AUTHNAME")
	if adminName == "" {
		fmt.Fprintln(os.Stderr, "MAITRED_AUTHNAME environment variable not set")
		os.Exit(-1)
	}

	_privkey, ok := os.LookupEnv("MAITRED_AUTHKEY")
	if !ok {
		fmt.Fprintln(os.Stderr, "MAITRED_AUTHKEY environment variable not set")
		os.Exit(-1)
	}

	var err error
	privkey, err = auth.ParsePrivateKey(_privkey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
	var onUpgraded func()

	switch len(os.Args) {
	case 3:
		if os.Args[1] != protocol.DelAuth {
			usage()
		}
		onUpgraded = func() { deleteUser(os.Args[2]) }
	case 4:
		if os.Args[1] != protocol.AddAuth {
			usage()
		}
		onUpgraded = func() { addUser(os.Args[2], os.Args[3]) }
	default:
		usage()
	}

	vc, _, _, _ := client.NewVanilla(
		address,
		nil, // we don't need the 'connected' hook
		nil, // we don't expect reconnects
	)

	adminClient = client.NewAdmin(vc, adminName, privkey)
	adminClient.Start()
	adminClient.Upgrade(
		onUpgraded,
		func() {
			os.Exit(5)
		},
	)

	for msg := range adminClient.Incoming() {
		adminClient.Handle(msg)
	}
}

func addUser(name, pubkey string) {
	adminClient.AddAuth(name, pubkey, func(reason string) {
		if reason != "" {
			fmt.Fprintln(os.Stderr, "couldn't add user:", reason)
			os.Exit(5)
		}
		os.Exit(0)
	})
}

func deleteUser(name string) {
	adminClient.DelAuth(name, func(reason string) {
		if reason != "" {
			fmt.Fprintln(os.Stderr, "couldn't delete user:", reason)
			os.Exit(5)
		}
		os.Exit(0)
	})
}
