package main

import (
	"fmt"
	"regexp"
	"strings"

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
		if m.GuildID == "" {
			sendBotMessage(s, m.ChannelID, "Points can only be changed inside a Discord server.")
			return
		}
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
		if m.GuildID == "" {
			sendBotMessage(s, m.ChannelID, "DuckBot server settings and leaderboards are only available inside a Discord server.")
			return
		}
		if handleManagerCommand(s, m) {
			return
		}
		if handleCounterCommand(s, m) {
			return
		}
		if handleLeaderboardDeleteCommand(s, m) {
			return
		}
		if handleLeaderboardVisibilityCommand(s, m) {
			return
		}

		println("Someone tagged me! I wonder if they want the LeaderBoard... ")
		//Check for "LeaderBoard" with word boundaries
		leaderboardMatch, _ := regexp.MatchString(".*\\bLEADERBOARD\\b.*", strings.ToUpper(m.Content))
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
		s.ChannelMessageSend(m.ChannelID, "I have a bad feeling about this...")
		return
	}
	println("Updating Score for" + item)
	var score int
	var updateErr error
	if user == nil {
		score, updateErr = updateScore(item, operation, m.GuildID, false)
	} else {
		score, updateErr = updateScore(item, operation, m.GuildID, true)
	}
	if updateErr != nil {
		fmt.Printf("Unable to update score for %s in guild %s: %s\n", item, m.GuildID, updateErr)
		sendBotMessage(s, m.ChannelID, "I couldn't update that score. Please try again.")
		return
	}

	counter, err := getCounter(m.GuildID)
	if err != nil {
		fmt.Printf("Unable to load counter for guild %s: %s\n", m.GuildID, err)
		counter = defaultCounter()
	}
	counterName := counterNameForScore(counter, score)
	if user == nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%[1]s has %[2]d %[3]s", item, score, counterName))
		if strings.ToUpper(item) == "SPINNYGORILLA" {
			s.ChannelMessageSend(m.ChannelID, "https://giphy.com/gifs/afvpets-afv-gorilla-KPgOYtIRnFOOk")
		}
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%[1]s> has %[2]d %[3]s", item, score, counterName))
	}

}

func handleLeaderboard(s *discordgo.Session, m *discordgo.Message) {
	guildName, err := discordGuildName(s, m.GuildID)
	if err != nil {
		fmt.Printf("Unable to load Discord name for guild %s: %s\n", m.GuildID, err)
		guildName = "this server"
	}

	public, visibilityErr := isGuildLeaderboardPublic(m.GuildID)
	if visibilityErr != nil {
		fmt.Printf("Unable to load leaderboard visibility for guild %s: %s\n", m.GuildID, visibilityErr)
		public = false
	}
	winnerEmoji, emojiErr := getGuildLeaderboardWinnerEmoji(m.GuildID)
	if emojiErr != nil {
		fmt.Printf("Unable to load leaderboard winner emoji for guild %s: %s\n", m.GuildID, emojiErr)
		winnerEmoji = defaultWinnerEmoji
	}

	things, thingsErr := getTopInGuild(m.GuildID, false)
	if thingsErr != nil {
		fmt.Printf("Unable to load thing leaderboard for guild %s: %s\n", m.GuildID, thingsErr)
		sendBotMessage(s, m.ChannelID, "I couldn't load this server's leaderboard.")
		return
	}
	members, membersErr := getTopInGuild(m.GuildID, true)
	if membersErr != nil {
		fmt.Printf("Unable to load member leaderboard for guild %s: %s\n", m.GuildID, membersErr)
		sendBotMessage(s, m.ChannelID, "I couldn't load this server's leaderboard.")
		return
	}
	memberIDs, membersErr := discordGuildMemberIDs(s, m.GuildID)
	if membersErr != nil {
		fmt.Printf("Unable to verify leaderboard members for guild %s: %s\n", m.GuildID, membersErr)
		sendBotMessage(s, m.ChannelID, "I couldn't load this server's leaderboard.")
		return
	}
	members = filterCurrentGuildMembers(members, memberIDs)

	embed := buildLeaderboardEmbed(
		m.GuildID,
		guildName,
		things,
		members,
		winnerEmoji,
		public,
		m.Author,
	)
	_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embed: embed,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	})
	if err != nil {
		fmt.Printf("Unable to send leaderboard response to channel %s: %s\n", m.ChannelID, err)
	}
}
