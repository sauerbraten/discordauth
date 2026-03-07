package auth

import (
	"strconv"
	"strings"
)

type Role int32

const (
	RoleNone Role = iota
	RoleMaster
	RoleAuth
	RoleAdmin
)

func ParseRole(s string) Role {
	switch strings.ToLower(s) {
	case "none":
		return RoleNone
	case "master":
		return RoleMaster
	case "auth":
		return RoleAuth
	case "admin":
		return RoleAdmin
	default:
		return -1
	}
}

func (r Role) String() string {
	switch r {
	case RoleNone:
		return "none"
	case RoleMaster:
		return "master"
	case RoleAuth:
		return "auth"
	case RoleAdmin:
		return "admin"
	default:
		return strconv.Itoa(int(r))
	}
}
