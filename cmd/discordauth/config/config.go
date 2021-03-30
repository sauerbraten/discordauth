package config

import (
	"log"
	"os"
)

var (
	DiscordToken = mustEnv("DISCORD_TOKEN")
)

func mustEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s not set\n", name)
	}
	return value
}
