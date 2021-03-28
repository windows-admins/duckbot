package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

type PointItem struct {
	Item   string  `json:"item"`
	Points float64 `json:"points"`
	IsUser bool    `json:"isUser"`
}

func main() {
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal(1, fmt.Sprintf("error opening Discord Session, %[1]s", err))
		return
	}
	err = dg.Open()
	if err != nil {
		log.Fatal(1, fmt.Sprintf("error opening connection, %[1]s", err))
		return
	}
	go discordListener(dg)
	fmt.Println("Bot is now running.")
	go runSite()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func discordListener(dg *discordgo.Session) {

	dg.AddHandler(messageCreate)

}

func guildHandler(w http.ResponseWriter, r *http.Request) {
	var getMembers bool
	vars := mux.Vars(r)
	if vars["type"] == "members" {
		getMembers = true
	} else if vars["type"] == "things" {
		getMembers = false
	} else {
		http.NotFound(w, r)
		return
	}
	list := getTopInGuild(vars["guild"], getMembers)
	json, _ := json.Marshal(list)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(json))
}

func runSite() {
	router := mux.NewRouter()
	router.HandleFunc("/guild/{guild}/{type}", guildHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	srv := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf("0.0.0.0:%[1]s", port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	} else {
		go userMessageHandler(s, m.Message)
	}
}
