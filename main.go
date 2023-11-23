package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"runtime"
	"time"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

var (
	Token string
	CommandPrefix string = "we/"
)

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Bool("d", false, "Debug mode")
	flag.Parse()
}

func main() {
	// check if we need to enable debug mode
	if flag.Lookup("d").Value.String() == "true" {
		log.SetLevel(log.DebugLevel)
	}

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsAll

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Error("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func sendMessage(s *discordgo.Session, m *discordgo.MessageCreate, msg string) {
	log.Debugf("Sending msg '%s'", msg) // use another arg for what type of logging we need (i.e. debug, info, error, etc.)
	s.ChannelMessageSend(m.ChannelID, msg)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func info(s *discordgo.Session, m *discordgo.MessageCreate) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	sendMessage(s, m, fmt.Sprintf("```Allocated Memory: %v MB\nSystem Memory: %v MB\nNumber of Garbage Collections: %v\n```", bToMb(memStats.Alloc), bToMb(memStats.Sys), memStats.NumGC))
}

func ping(s *discordgo.Session, m *discordgo.MessageCreate) {
	latency := s.HeartbeatLatency().Seconds() * 1000
	res := fmt.Sprintf(":ping_pong: Pong! %dms", int(latency))
	sendMessage(s, m, res)
}


func parseTime(input string) (time.Time, error) {
	// Check if the input contains "h" or "hour" and extract the number
	if strings.Contains(input, "h") || strings.Contains(input, "hour") {
		re := regexp.MustCompile(`(\d+)\s*(h|hour)`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			hours, err := time.ParseDuration(matches[1] + "h")
			if err != nil {
				return time.Time{}, err
			}
			return time.Now().Add(hours), nil
		}
	}

	// Check if the input contains "min" or "minute" and extract the number
	if strings.Contains(input, "min") || strings.Contains(input, "minute") {
		re := regexp.MustCompile(`(\d+)\s*(min|minute)`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			minutes, err := time.ParseDuration(matches[1] + "m")
			if err != nil {
				return time.Time{}, err
			}
			return time.Now().Add(minutes), nil
		}
	}

	// Check if the input contains "s" or "second" and extract the number
	if strings.Contains(input, "s") || strings.Contains(input, "second") {
		re := regexp.MustCompile(`(\d+)\s*(s|second)`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			seconds, err := time.ParseDuration(matches[1] + "s")
			if err != nil {
				return time.Time{}, err
			}
			return time.Now().Add(seconds), nil
		}
	}

	// Check if the input is a date in the format MM/DD/YYYY or MM/DD
	layout := "01/02/2006"
	if len(input) == len(layout) || len(input) == len("01/02") {
		return time.Parse(layout, input)
	}

	// If none of the formats match, try parsing as RFC3339
	return time.Parse(time.RFC3339, input)
}

type Reminder struct {
	Time    time.Time
	Message string
}

var reminders []Reminder

func remind(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandArgs := strings.Split(m.Content, " ")
	if len(commandArgs) < 3 {
		sendMessage(s, m, fmt.Sprintf("Usage: `%sreminder <time> <message>`", CommandPrefix))
		return
	}
	timeStr := commandArgs[1]
	message := strings.Join(commandArgs[2:], " ")

	reminderTime, err := parseTime(timeStr)
	if err != nil {
		sendMessage(s, m, "Error parsing time. Please use a valid time format.")
		return
	}

	newReminder := Reminder{
		Time:    reminderTime,
		Message: message,
	}
	reminders = append(reminders, newReminder)

	sendMessage(s, m, fmt.Sprintf("Reminding you about: %s in %s", message, timeStr))

	// start a goroutine to wait for the reminder
	go func(reminder Reminder) {
		time.Sleep(reminder.Time.Sub(time.Now()))
		discordRelativeTime := fmt.Sprintf("%d", reminderTime.Unix())
		// mention the user who set the reminder
		sendMessage(s, m, fmt.Sprintf("<@%s>, <t:%s:R>: %s", m.Author.ID, discordRelativeTime,reminder.Message))
	}(newReminder)
}


func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
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
	case "ping":
		ping(s, m)
	case "remind":
		remind(s, m)
	case "info":
		info(s, m)
	default:
		//  send back we do not know that command
		sendMessage(s, m, fmt.Sprintf("Unknown command '%s'", msg))
	}


}
