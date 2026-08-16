package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/bwmarrin/discordgo"
)

const (
	leaderboardConfigRow     = "LEADERBOARD"
	defaultLeaderboardPublic = false
)

type leaderboardVisibilityCommand struct {
	Action string
	Public bool
}

func parseLeaderboardVisibilityCommand(content string, botID string) (leaderboardVisibilityCommand, bool, error) {
	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	parts := strings.Fields(strings.TrimSpace(content))

	if len(parts) == 0 || !strings.EqualFold(parts[0], "leaderboard") {
		return leaderboardVisibilityCommand{}, false, nil
	}
	if len(parts) == 1 {
		return leaderboardVisibilityCommand{}, false, nil
	}
	if len(parts) != 2 {
		return leaderboardVisibilityCommand{}, true, errors.New("use `leaderboard public`, `leaderboard private`, or `leaderboard status`")
	}

	switch strings.ToLower(parts[1]) {
	case "public":
		return leaderboardVisibilityCommand{Action: "set", Public: true}, true, nil
	case "private":
		return leaderboardVisibilityCommand{Action: "set", Public: false}, true, nil
	case "status":
		return leaderboardVisibilityCommand{Action: "show"}, true, nil
	default:
		return leaderboardVisibilityCommand{}, true, errors.New("use `leaderboard public`, `leaderboard private`, or `leaderboard status`")
	}
}

func handleLeaderboardVisibilityCommand(s *discordgo.Session, m *discordgo.Message) bool {
	command, matched, err := parseLeaderboardVisibilityCommand(m.Content, s.State.User.ID)
	if !matched {
		return false
	}
	if err != nil {
		sendBotMessage(s, m.ChannelID, err.Error())
		return true
	}

	if command.Action == "show" {
		public, getErr := isGuildLeaderboardPublic(m.GuildID)
		if getErr != nil {
			fmt.Printf("Unable to load leaderboard visibility for guild %s: %s\n", m.GuildID, getErr)
			sendBotMessage(s, m.ChannelID, "I couldn't load this server's leaderboard setting.")
			return true
		}
		if public {
			sendBotMessage(s, m.ChannelID, "This server's leaderboard is public.")
		} else {
			sendBotMessage(s, m.ChannelID, "This server's leaderboard is private.")
		}
		return true
	}

	authorized, permissionErr := canManageGuildSettings(s, m)
	if permissionErr != nil {
		fmt.Printf("Unable to check leaderboard permissions for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, permissionErr)
		sendBotMessage(s, m.ChannelID, "I couldn't verify your server permissions.")
		return true
	}
	if !authorized {
		sendBotMessage(s, m.ChannelID, "Only server administrators and Mainduck can change leaderboard visibility.")
		return true
	}

	if saveErr := saveGuildLeaderboardVisibility(m.GuildID, command.Public); saveErr != nil {
		fmt.Printf("Unable to save leaderboard visibility for guild %s: %s\n", m.GuildID, saveErr)
		sendBotMessage(s, m.ChannelID, "I couldn't save this server's leaderboard setting.")
		return true
	}

	if command.Public {
		sendBotMessage(s, m.ChannelID, "This server's leaderboard is now public.")
	} else {
		sendBotMessage(s, m.ChannelID, "This server's leaderboard is now private.")
	}
	return true
}

func isGuildLeaderboardPublic(guildID string) (bool, error) {
	if guildID == "" {
		return defaultLeaderboardPublic, nil
	}

	entity, err := leaderboardConfigEntity(guildID)
	if err != nil {
		return false, err
	}
	err = entity.Get(30, storage.NoMetadata, nil)
	if err != nil {
		var storageErr *storage.AzureStorageServiceError
		if errors.As(err, &storageErr) && storageErr.StatusCode == http.StatusNotFound {
			return defaultLeaderboardPublic, nil
		}
		return false, fmt.Errorf("get leaderboard config entity: %w", err)
	}

	public, ok := entity.Properties["Public"].(bool)
	if !ok {
		return false, errors.New("leaderboard config entity has invalid Public property")
	}
	return public, nil
}

func saveGuildLeaderboardVisibility(guildID string, public bool) error {
	if guildID == "" {
		return errors.New("leaderboard settings require a Discord server")
	}

	entity, err := leaderboardConfigEntity(guildID)
	if err != nil {
		return err
	}
	entity.Properties = map[string]interface{}{"Public": public}
	if err := entity.InsertOrReplace(nil); err != nil {
		return fmt.Errorf("save leaderboard config entity: %w", err)
	}
	return nil
}

func leaderboardConfigEntity(guildID string) (*storage.Entity, error) {
	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	return table.GetEntityReference(guildID+"-CONFIG", leaderboardConfigRow), nil
}
