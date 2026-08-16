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
	"strings"
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

type GuildSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GuildLeaderboardResponse struct {
	Guild GuildSummary `json:"guild"`
	Items []PointItem  `json:"items"`
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
	go runSite(dg)
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
type guildPointsLoader func(string, bool) ([]PointItem, error)
type guildNameLoader func(string) (string, error)

var apiLeaderboardCache = newLeaderboardResponseCache()

func guildHandler(session *discordgo.Session) http.HandlerFunc {
	return guildHandlerWithDependencies(isGuildLeaderboardPublic, getTopInGuild, func(guildID string) (string, error) {
		return discordGuildName(session, guildID)
	}, apiLeaderboardCache)
}

func guildHandlerWithDependencies(isPublic guildVisibilityChecker, loadPoints guildPointsLoader, loadGuildName guildNameLoader, cache *leaderboardResponseCache) http.HandlerFunc {
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

		cacheKey := guildID + ":" + vars["type"]
		if response, found := cache.get(cacheKey, time.Now()); found {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(response)
			return
		}

		guildName, err := loadGuildName(guildID)
		if err != nil {
			log.Printf("Unable to load Discord name for guild %s: %s", guildID, err)
			http.Error(w, "Unable to load leaderboard.", http.StatusInternalServerError)
			return
		}

		list, loadErr := loadPoints(guildID, getMembers)
		if loadErr != nil {
			log.Printf("Unable to load leaderboard for guild %s: %s", guildID, loadErr)
			http.Error(w, "Unable to load leaderboard.", http.StatusInternalServerError)
			return
		}
		response, err := json.Marshal(GuildLeaderboardResponse{
			Guild: GuildSummary{
				ID:   guildID,
				Name: guildName,
			},
			Items: list,
		})
		if err != nil {
			log.Printf("Unable to encode leaderboard for guild %s: %s", guildID, err)
			http.Error(w, "Unable to load leaderboard.", http.StatusInternalServerError)
			return
		}
		cache.set(cacheKey, response, time.Now())
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(response)
	}
}

func discordGuildName(session *discordgo.Session, guildID string) (string, error) {
	guild, err := session.State.Guild(guildID)
	if err != nil {
		guild, err = session.Guild(guildID)
	}
	if err != nil {
		return "", fmt.Errorf("get Discord guild: %w", err)
	}

	name := strings.TrimSpace(guild.Name)
	if name == "" {
		return "", fmt.Errorf("Discord guild %s has no name", guildID)
	}
	return name, nil
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
		w.Header().Add("Vary", "Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		next.ServeHTTP(w, r)
	})
}

func runSite(session *discordgo.Session) {
	router := mux.NewRouter()
	router.HandleFunc("/guild/{guild}/{type}", guildHandler(session))
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	srv := &http.Server{
		Handler: allowDuckPointsCORS(rateLimitRequests(newIPRateLimiter(), router)),
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
