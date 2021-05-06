package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
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

	//Check for ++ or -- or ==
	pointsData := extractPlusMinusEventData(m.Content)
	if pointsData != nil {
		item := pointsData[0]
		operation := pointsData[1]
		user, _ := s.User(item)
		if operation == "++" || operation == "--" || operation == "—" {
			handlePlusMinus(item, operation, s, m, user)
		}
	}

	//Check for Mention of this bot user ID in this message
	mentionMap := make(map[string]bool)
	for i := 0; i < len(m.Mentions); i++ {
		mentionMap[m.Mentions[i].ID] = true
	}
	if _, ok := mentionMap[s.State.User.ID]; ok {
		println("Someone tagged me! I wonder if they want the LeaderBoard... ")
		//Check for "LeaderBoard" with word boundaries
		leaderboardMatch, _ := regexp.MatchString(".*\\bLEADERBOARD\\b*.", strings.ToUpper(m.Content))
		if leaderboardMatch {
			println("They did!")
			handleLeaderboard(s, m)
		}
	}

	// parameters := strings.Split(m.Content, " ")
	// if RegexUserPatternID.MatchString(parameters[0]) {
	// 	if strings.ToUpper(parameters[1]) == "LEADERBOARD" {
	// 		s.ChannelMessageSend(m.ChannelID, "Here is the leaderboard!")
	// 	}
	// 	s.ChannelMessageSend(m.ChannelID, "Quack!")
	// }

	return
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

func handleLeaderboard(s *discordgo.Session, m *discordgo.Message) {
	// if item == m.Author.ID {
	// 	s.ChannelMessageSend(m.ChannelID, "Really now? Don't try to steal points!")
	// 	return
	// }
	println("Printing Leaderboard for " + m.Author.Username)
	s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		URL:         "",
		Type:        discordgo.EmbedTypeRich,
		Title:       "",
		Description: "Here are the leaderboards!",
		Timestamp:   time.Now().Local().String(),
		Color:       0,
		Footer:      &discordgo.MessageEmbedFooter{IconURL: m.Author.AvatarURL(""), Text: fmt.Sprintf("Invoked by %s", m.Author.Username)},
		Image:       &discordgo.MessageEmbedImage{},
		Thumbnail:   &discordgo.MessageEmbedThumbnail{},
		Video:       &discordgo.MessageEmbedVideo{},
		Provider:    &discordgo.MessageEmbedProvider{},
		Author:      &discordgo.MessageEmbedAuthor{},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Users",
				Value:  "[Leaderboard](https://duckbotdiscorduku6efjff3bps.azurewebsites.net/guild/618712310185197588/members)",
				Inline: true,
			},
			{
				Name:   "Things",
				Value:  "[Leaderboard](https://duckbotdiscorduku6efjff3bps.azurewebsites.net/guild/618712310185197588/things)",
				Inline: true,
			},
		},
	},
	)
}
