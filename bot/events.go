package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
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
		if operation == "++" || operation == "--" || operation == "—" {
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
	if item == m.Author.ID {
		s.ChannelMessageSend(m.ChannelID, "Really now? Don't try to steal points!")
		return
	}
	println("Updating Score for" + item)
	var score int
	if user == nil {
		score = updateScore(item, operation, m.GuildID, false)
	} else {
		score = updateScore(item, operation, m.GuildID, true)
	}

	var plural string
	if score == 1 {
		plural = ""
	} else {
		plural = "s"
	}
	if user == nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%[1]s has %[2]d vaccination%[3]s", item, score, plural))
		if strings.ToUpper(item) == "SPINNYGORILLA" {
			s.ChannelMessageSend(m.ChannelID, "https://giphy.com/gifs/afvpets-afv-gorilla-KPgOYtIRnFOOk")
		}
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%[1]s> has %[2]d vaccination%[3]s", item, score, plural))
	}

}
