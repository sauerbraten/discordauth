package client

import (
	"github.com/sauerbraten/maitred/pkg/auth"
)

type Client interface {
	auth.Provider
	Register()
	Send(string, ...interface{})
	Handle(string)
	Log(string, ...interface{})
}
