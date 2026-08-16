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

type guildVisibilityChecker func(string) (bool, error)
type guildPointsLoader func(string, bool) []PointItem

func guildHandler(w http.ResponseWriter, r *http.Request) {
	guildHandlerWithDependencies(isGuildLeaderboardPublic, getTopInGuild).ServeHTTP(w, r)
}

func guildHandlerWithDependencies(isPublic guildVisibilityChecker, loadPoints guildPointsLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		guildID := vars["guild"]
		if !isDiscordGuildID(guildID) {
			http.NotFound(w, r)
			return
		}

		public, err := isPublic(guildID)
		if err != nil {
			log.Printf("Unable to check leaderboard visibility for guild %s: %s", guildID, err)
			http.Error(w, "Unable to load leaderboard.", http.StatusInternalServerError)
			return
		}
		if !public {
			http.NotFound(w, r)
			return
		}

		list := loadPoints(guildID, getMembers)
		response, err := json.Marshal(list)
		if err != nil {
			log.Printf("Unable to encode leaderboard for guild %s: %s", guildID, err)
			http.Error(w, "Unable to load leaderboard.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

func isDiscordGuildID(guildID string) bool {
	if len(guildID) < 17 || len(guildID) > 20 {
		return false
	}
	for _, character := range guildID {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func allowDuckPointsCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"https://duckpoints.com":     true,
		"https://www.duckpoints.com": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func runSite() {
	router := mux.NewRouter()
	router.HandleFunc("/guild/{guild}/{type}", guildHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	srv := &http.Server{
		Handler: allowDuckPointsCORS(router),
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
