package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
)

func Pong(m *discordgo.MessageCreate) {
	latency := DiscordSession.HeartbeatLatency().Seconds() * 1000
	res := fmt.Sprintf(":ping_pong: Pong! %dms", int(latency))
	SendMessage(m.ChannelID, res)
}