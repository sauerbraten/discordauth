package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/sauerbraten/maitred/v2/cmd/discordauth/config"
	"github.com/sauerbraten/maitred/v2/internal/db"
)

func setupDiscord(addUser func(name, pubkey string, override bool) error) func() {
	d, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		log.Fatalf("discord: error creating session: %v\n", err)
	}

	d.AddHandler(func(_ *discordgo.Session, _ *discordgo.Connect) {
		log.Println("discord: connected")
	})

	d.AddHandler(func(_ *discordgo.Session, _ *discordgo.Disconnect) {
		log.Println("discord: disconnected")
	})

	d.AddHandler(func(d *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == d.State.User.ID {
			return
		}
		c, err := d.Channel(m.ChannelID)
		if err != nil {
			log.Printf("discord: getting channel info: %v\n", err)
			return
		}
		if c.Type != discordgo.ChannelTypeDM {
			return
		}
		name := m.Author.Username + "#" + m.Author.Discriminator
		content := m.Content
		override := false
		if strings.HasPrefix(content, "override ") {
			override = true
			content = content[len("override "):]
		}
		err = addUser(name, content, override)
		if err != nil {
			if existsErr := new(db.UserExistsError); errors.As(err, existsErr) {
				sendMessage(d, m.ChannelID, fmt.Sprintf("You are already registered (your public key is: %s).\nTo replace your registered public key, send `override %s`.", existsErr.PublicKey, existsErr.PublicKey))
			} else {
				log.Println("discord: adding user:", err)
				log.Printf("discord: ignoring message: %s\n", m.Content)
				sendMessage(d, m.ChannelID, fmt.Sprintf("That didn't work! :dizzy_face: To register, follow these steps:\n 1. in Sauerbraten, run `/authkey \"%s\" (genauthkey (rndstr 32)) p1x.pw; saveauthkeys; echo (getpubkey p1x.pw)`\n 2. send me the last line of output here (it's easiest to copy this from the command line window)\n", name))
			}
			return
		}
		log.Printf("discord: %s registered using public key %s\n", name, content)
		sendMessage(d, m.ChannelID, fmt.Sprintf("registered you as **%s**!", name))
	})

	d.Identify.Intents = discordgo.IntentsDirectMessages

	err = d.Open()
	if err != nil {
		log.Fatalf("discord: error opening session: %v\n", err)
	}

	return func() {
		err := d.Close()
		if err != nil {
			log.Printf("discord: error closing connection: %v\n", err)
		}
	}
}

func sendMessage(d *discordgo.Session, channelID, content string) {
	_, err := d.ChannelMessageSend(channelID, content)
	if err != nil {
		log.Printf("error replying with '%s': %v\n", content, err)
	}
}
