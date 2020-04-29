package main

import (
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

func userMessageHandler(s *discordgo.Session, m *discordgo.Message) {
	duckMatch, _ := regexp.MatchString(".*[Qq][Uu][Aa][Cc][Kk]*.", m.Content)
	if duckMatch {
		handleQuack(s, m)

	}
	pointsData := extractPlusMinusEventData(m.Content)
	if pointsData != nil {
		item := pointsData[0]
		operation := pointsData[1]
		if operation == "++" || operation == "--" {
			handlePlusMinus(item, operation, s, m)
		}
		return
	}

}

func handleQuack(s *discordgo.Session, m *discordgo.Message) {
	s.ChannelMessageSend(m.ChannelID, "Quack!")
	return
}

func handlePlusMinus(item string, operation string, s *discordgo.Session, m *discordgo.Message) {
	println("Updating Score for" + item)
	recordID := updateScore(item, operation, m.GuildID)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%[1]s> has %[2]d points", item, getPointsByID(recordID[0])))

}
