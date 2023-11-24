package main

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

var (
	Token string
	CommandPrefix string = "we/"
	DiscordSession *discordgo.Session
)

func init() {
	// load token from env
	Token = os.Getenv("TOKEN")
}

func SendMessage(channelid string, msg string) {
	log.Debugf("Sending msg '%s'", msg) // use another arg for what type of logging we need (i.e. debug, info, error, etc.)
	DiscordSession.ChannelMessageSend(channelid, msg)
}

func main() {
	InitializeDatabase()
	defer Db.Close()

	log.Info("Attempting to start bot...")
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	dg.Identify.Intents = discordgo.IntentsAll
	dg.AddHandler(messageCreate)
	DiscordSession = dg

	err = dg.Open()
	if err != nil {
		log.Error("error opening connection,", err)
		return
	}
	defer dg.Close()
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	
	reminderLoop()

	// Logic to stop the bot
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore messages from self and other bots
	if m.Author.ID == s.State.User.ID || m.Author.Bot{
		return
	}

	log.Debugf("Processing msg '%s' from '%s'", m.Content, m.Author.Username)

	msg := strings.TrimSpace(m.Content)
	// check if msg starts with CommandPrefix
	if !strings.HasPrefix(msg, CommandPrefix) {
		return
	}
	msg = strings.TrimPrefix(msg, CommandPrefix)
	// check if msg is empty
	if msg == "" {
		return
	}

	baseCommand := strings.Split(msg, " ")[0]
	// use switch statement to handle commands
	switch baseCommand {
	case "remind":
		Remind(m)
	case "info":
		Info(m)
	case "ping":
		Pong(m)
	default:
		//  send back we do not know that command
		SendMessage(m.ChannelID, fmt.Sprintf("Unknown command '%s'", msg))
	}
}