package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
)

type dbOperation struct {
    query string
    args  []interface{}
}

var (
	Token         string
	CommandPrefix string = "we/"
	dbMutex       sync.Mutex
	db            *sql.DB
	writeQueue = make(chan dbOperation, 100) // adjust the size as needed
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Bool("d", false, "Debug mode")
	flag.Parse()

	go func() {
        for op := range writeQueue {
            err := execStatement(db, op.query, op.args...)
            if err != nil {
                log.Errorf("error writing to db: %s", err)
            }
        }
    }()

}

func main() {
	// check if we need to enable debug mode
	if flag.Lookup("d").Value.String() == "true" {
		log.SetLevel(log.DebugLevel)
	}

	// initialize database
	var err error
    db, err = sql.Open("sqlite3", "./wolfecho.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()


	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsAll

	dg.AddHandler(messageCreate)
	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Error("error opening connection,", err)
		return
	}
	defer dg.Close()

	// create reminder table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS reminders (id INTEGER PRIMARY KEY, authorid string, channelid string, time INTEGER, message TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	// delete old reminders
	deleteOldReminders()

	// start reminder loop
	go reminderLoop(dg)


	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func reminderLoop(s *discordgo.Session) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			// check if there are any reminders that need to be sent
			rows, err := db.Query("SELECT id, time, authorid, channelid, message FROM reminders WHERE time <= ?", time.Now().Unix())
			if err != nil {
				log.Errorf("error querying db: %s", err)
				continue
			}
			for rows.Next() {
				var id int
				var time int
				var authorid string
				var channelid string
				var message string
				err = rows.Scan(&id, &time, &authorid, &channelid, &message)
				if err != nil {
					log.Errorf("error scanning row: %s", err)
					continue
				}
				// send message
				sendMessage(s, channelid, fmt.Sprintf("<@%s>, <t:%d:R>: %s", authorid, time, message))

				// delete reminder
				writeQueue <- dbOperation{
					query: "DELETE FROM reminders WHERE id = ?",
					args: []interface{}{id},
				}
			}
			rows.Close()
		}
	}
}

func deleteOldReminders() {
	writeQueue <- dbOperation{
		query: "DELETE FROM reminders WHERE time < ?",
		args: []interface{}{time.Now().Unix()},
	}
	log.Debug("Added deleteOldReminders to writeQueue")
}

func execStatement(db *sql.DB, query string, args ...interface{}) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	return err
}

func sendMessage(s *discordgo.Session, channelid string, msg string) {
	log.Debugf("Sending msg '%s'", msg) // use another arg for what type of logging we need (i.e. debug, info, error, etc.)
	s.ChannelMessageSend(channelid, msg)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func info(s *discordgo.Session, m *discordgo.MessageCreate) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	sendMessage(s, m.ChannelID, fmt.Sprintf("```Allocated Memory: %v MB\nSystem Memory: %v MB\nNumber of Garbage Collections: %v\n```", bToMb(memStats.Alloc), bToMb(memStats.Sys), memStats.NumGC))
}

func ping(s *discordgo.Session, m *discordgo.MessageCreate) {
	latency := s.HeartbeatLatency().Seconds() * 1000
	res := fmt.Sprintf(":ping_pong: Pong! %dms", int(latency))
	sendMessage(s, m.ChannelID, res)
}

func parseTime(input string) (int64, error) {
	// Check if the input contains "h" or "hour" and extract the number
	if strings.Contains(input, "h") || strings.Contains(input, "hour") {
		re := regexp.MustCompile(`(\d+)\s*(h|hour)`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			hours, err := time.ParseDuration(matches[1] + "h")
			if err != nil {
				return 0, err
			}
			return time.Now().Add(hours).Unix(), nil
		}
	}

	// Check if the input contains "min" or "minute" and extract the number
	if strings.Contains(input, "min") || strings.Contains(input, "minute") {
		re := regexp.MustCompile(`(\d+)\s*(min|minute)`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			minutes, err := time.ParseDuration(matches[1] + "m")
			if err != nil {
				return 0, err
			}
			return time.Now().Add(minutes).Unix(), nil
		}
	}

	// Check if the input contains "s" or "second" and extract the number
	if strings.Contains(input, "s") || strings.Contains(input, "second") {
		re := regexp.MustCompile(`(\d+)\s*(s|second)`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			seconds, err := time.ParseDuration(matches[1] + "s")
			if err != nil {
				return 0, err
			}
			return time.Now().Add(seconds).Unix(), nil
		}
	}

	// Check if the input is a date in the format MM/DD/YYYY or MM/DD
	layout := "01/02/2006"
	if len(input) == len(layout) || len(input) == len("01/02") {
		t, err := time.Parse(layout, input)
		if err != nil {
			return 0, err
		}
		return t.Unix(), nil
	}

	// If none of the formats match, try parsing as RFC3339
	t, err := time.Parse(time.RFC3339, input)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func remind(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandArgs := strings.Split(m.Content, " ")
	if len(commandArgs) < 3 {
		sendMessage(s, m.ChannelID, fmt.Sprintf("Usage: `%sreminder <time> <message>`", CommandPrefix))
		return
	}
	timeStr := commandArgs[1]
	message := strings.Join(commandArgs[2:], " ")

	reminderTime, err := parseTime(timeStr)
	if err != nil {
		sendMessage(s, m.ChannelID, "Error parsing time. Please use a valid time format.")
		return
	}

	// write to db
	writeQueue <- dbOperation{
		query: "INSERT INTO reminders (authorid, channelid, time, message) VALUES (?, ?, ?, ?)",
		args: []interface{}{m.Author.ID, m.ChannelID, reminderTime, message},
	}

	sendMessage(s, m.ChannelID, fmt.Sprintf("Reminding you about: %s in %s", message, timeStr))
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
		sendMessage(s, m.ChannelID, fmt.Sprintf("Unknown command '%s'", msg))
	}

}
