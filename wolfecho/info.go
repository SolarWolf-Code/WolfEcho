package main

import (
	"fmt"
	"runtime"
	"github.com/bwmarrin/discordgo"
)

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func Info(m *discordgo.MessageCreate) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	SendMessage(m.ChannelID, fmt.Sprintf("```Allocated Memory: %v MB\nSystem Memory: %v MB\nNumber of Garbage Collections: %v\n```", bToMb(memStats.Alloc), bToMb(memStats.Sys), memStats.NumGC))
}