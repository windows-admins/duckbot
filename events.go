package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
	"strings"
)

const (
	//MinimumCharactersOnID ...
	MinimumCharactersOnID int = 16
)

var (
	//RegexUserPatternID ...
	RegexUserPatternID *regexp.Regexp = regexp.MustCompile(fmt.Sprintf(`^(<@!(\d{%d,})>)$`, MinimumCharactersOnID))
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
		user, _ := s.User(item)
		if operation == "++" || operation == "--" {
			handlePlusMinus(item, operation, s, m, user)
		}
		return
	}

	// parameters := strings.Split(m.Content, " ")
	// if RegexUserPatternID.MatchString(parameters[0]) {
	// 	if strings.ToUpper(parameters[1]) == "LEADERBOARD" {
	// 		s.ChannelMessageSend(m.ChannelID, "Here is the leaderboard!")
	// 	}
	// 	s.ChannelMessageSend(m.ChannelID, "Quack!")
	// }

}

func handleQuack(s *discordgo.Session, m *discordgo.Message) {
	s.ChannelMessageSend(m.ChannelID, "Quack!")
	return
}

func handlePlusMinus(item string, operation string, s *discordgo.Session, m *discordgo.Message, user *discordgo.User) {
	println("Updating Score for" + item)
	score := updateScore(item, operation, m.GuildID)
	var plural string
	if score == 1 {
		plural = ""
	} else {
		plural = "s"
	}
	if user == nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%[1]s has %[2]d vacination%[3]s", item, score, plural))
	} else {
		if user.ID == m.Author.ID {
			s.ChannelMessageSend(m.ChannelID, "Really now? Don't try to steal points!")
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%[1]s> has %[2]d vacination%[3]s", item, score, plural))
	}

}
