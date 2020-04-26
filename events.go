package main
import (
	"github.com/bwmarrin/discordgo"
	"regexp"
)
func userMessageHandler (s *discordgo.Session, m *discordgo.Message) {
	duckMatch, _ := regexp.MatchString(".*[Qq][Uu][Aa][Cc][Kk]*.", m.Content)
	if (duckMatch) {
		handleQuack(s,m)
		return
	}
	pointsData  := extractPlusMinusEventData(m.Content)
	if (pointsData != nil) {
		
		return
	}

}

func handleQuack(s *discordgo.Session, m *discordgo.Message) {
	s.ChannelMessageSend(m.ChannelID, "Quack!")
	return
}