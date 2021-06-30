package config

import (
	"log"
	"os"
	"strings"
)

var (
	DiscordToken = mustEnv("DISCORD_TOKEN")
	Admins       = parseListAsSet(mustEnv("ADMINS"))
)

func mustEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s not set\n", name)
	}
	return value
}

func parseListAsSet(s string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, elem := range strings.FieldsFunc(s, func(c rune) bool { return c == ',' }) {
		set[elem] = struct{}{}
	}
	return set
}
