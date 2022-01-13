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

func startDiscord(s *Server) func() {
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
		if m.Author == nil || m.Author.ID == d.State.User.ID {
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

		handleMessage(d, m.Message, s)
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

func handleMessage(d *discordgo.Session, m *discordgo.Message, s *Server) {
	authorName := removeWhitespace(m.Author.Username) + "#" + m.Author.Discriminator

	switch fields := strings.Fields(m.Content); fields[0] {
	case "ban":
		if _, ok := config.Admins[authorName]; !ok {
			return
		}
		for _, user := range m.Mentions {
			targetName := user.Username + "#" + user.Discriminator
			err := s.delUser(targetName)
			if err != nil {
				sendMessage(d, m.ChannelID, fmt.Sprintf(":boom: deleting %s: %v", targetName, err))
				continue
			}
			err = s.banUser(targetName)
			if err != nil {
				sendMessage(d, m.ChannelID, fmt.Sprintf(":boom: banning %s: %v", targetName, err))
				continue
			}
			sendMessage(d, m.ChannelID, fmt.Sprintf(":white_check_mark: banned %s", targetName))
		}
	case "unban":
		if _, ok := config.Admins[authorName]; !ok {
			return
		}
		for _, user := range m.Mentions {
			targetName := user.Username + "#" + user.Discriminator
			err := s.unbanUser(targetName)
			if err != nil {
				sendMessage(d, m.ChannelID, fmt.Sprintf(":boom: unbanning %s: %v", targetName, err))
				continue
			}
			sendMessage(d, m.ChannelID, fmt.Sprintf(":white_check_mark: unbanned %s", targetName))
		}
	default: // normal user registering
		content := m.Content
		override := false
		if strings.HasPrefix(content, "override ") {
			override = true
			content = content[len("override "):]
		}
		banned, err := s.isBanned(authorName)
		if err != nil {
			log.Printf("discord: checking if %s is banned: %v", authorName, err)
			log.Printf("discord: ignoring message: %s\n", m.Content)
			sendMessage(d, m.ChannelID, fmt.Sprintf("That didn't work! :thinking: I can't tell if you are banned or not."))
			return
		}
		if banned {
			return
		}
		err = s.addUser(authorName, content, override)
		if err != nil {
			if existsErr := new(db.UserExistsError); errors.As(err, existsErr) {
				sendMessage(d, m.ChannelID, fmt.Sprintf("You are already registered (your public key is: %s).\nTo replace your registered public key, send `override %s`.", existsErr.PublicKey, content))
			} else {
				log.Println("discord: adding user:", err)
				log.Printf("discord: ignoring message: %s\n", m.Content)
				sendMessage(d, m.ChannelID, fmt.Sprintf("That didn't work! :dizzy_face: To register, follow these steps:\n 1. in Sauerbraten, run `/authkey \"%s\" (genauthkey (rndstr 32)) p1x.pw; saveauthkeys; echo (getpubkey p1x.pw)`\n 2. send me the last line of output here (it's easiest to copy this from the command line window)\n", authorName))
			}
			return
		}
		log.Printf("discord: %s registered using public key %s\n", authorName, content)
		sendMessage(d, m.ChannelID, fmt.Sprintf("registered you as **%s**!", authorName))
	}
}

func removeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), "")
}
