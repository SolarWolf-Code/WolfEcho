package main

import (
	"strings"
	"regexp"
	"time"
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"fmt"
)
func reminderLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			// check if there are any reminders that need to be sent
			rows, err := Db.Query("SELECT messageid, time, authorid, channelid, message FROM reminders WHERE time <= ?", time.Now().Unix())
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
				SendMessage(channelid, fmt.Sprintf("<@%s>, <t:%d:R>: %s", authorid, time, message))

				// delete reminder
				WriteQueue <- DbOperation{
					query: "DELETE FROM reminders WHERE messageid = ?",
					args: []interface{}{id},
				}
			}
			rows.Close()
		}
	}
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

func Remind(m *discordgo.MessageCreate) {
	commandArgs := strings.Split(m.Content, " ")
	if len(commandArgs) < 3 {
		SendMessage(m.ChannelID, fmt.Sprintf("Usage: `%sreminder <time> <message>`", CommandPrefix))
		return
	}
	timeStr := commandArgs[1]
	message := strings.Join(commandArgs[2:], " ")

	reminderTime, err := parseTime(timeStr)
	if err != nil {
		SendMessage(m.ChannelID, "Error parsing time. Please use a valid time format.")
		return
	}

	// write to db
	WriteQueue <- DbOperation{
		// log the message id
		query: "INSERT INTO reminders (messageid, authorid, channelid, time, message) VALUES (?, ?, ?, ?, ?)",
		args: []interface{}{m.ID, m.Author.ID, m.ChannelID, reminderTime, message},
	}

	SendMessage(m.ChannelID, fmt.Sprintf("Reminding you about: %s in %s", message, timeStr))
}