package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	discordToken       string
	storageAccount     string
	storageAccessToken string
	storagePointTable  string
	storageMemberTable string
)

func init() {
	storageAccount = os.Getenv("DUCKBOT_STORAGEACCOUNT_NAME")
	storagePointTable = os.Getenv("DUCKBOT_STORAGEACCOUNT_POINTTABLE")
	storageMemberTable = os.Getenv("DUCKBOT_STORAGEACCOUNT_MEMBERTABLE")
	storageAccessToken = os.Getenv("DUCKBOT_STORAGEACCOUNT_TOKEN")
	discordToken = os.Getenv("DUCKBOT_DISCORD_TOKEN")

}

func main() {

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("error creating Discord	session,", err)
		return
	}

	dg.AddHandler(messageCreate)
	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	} else {
		userMessageHandler(s, m.Message)
	}
}
