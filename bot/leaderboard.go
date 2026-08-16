package main

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/bwmarrin/discordgo"
)

const (
	leaderboardConfigRow     = "LEADERBOARD"
	defaultLeaderboardPublic = false
	defaultWinnerEmoji       = ":high_heel:"
)

var emojiShortcodePattern = regexp.MustCompile(`^:[A-Za-z0-9_+-]+:$`)

type leaderboardVisibilityCommand struct {
	Action string
	Public bool
	Emoji  string
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
	if strings.EqualFold(parts[1], "emoji") {
		if len(parts) != 3 {
			return leaderboardVisibilityCommand{}, true, errors.New("use `leaderboard emoji <emoji>` or `leaderboard emoji reset`")
		}
		if strings.EqualFold(parts[2], "reset") {
			return leaderboardVisibilityCommand{Action: "reset-emoji"}, true, nil
		}
		if err := validateLeaderboardEmoji(parts[2]); err != nil {
			return leaderboardVisibilityCommand{}, true, err
		}
		return leaderboardVisibilityCommand{Action: "set-emoji", Emoji: parts[2]}, true, nil
	}
	if len(parts) != 2 {
		return leaderboardVisibilityCommand{}, true, errors.New("use `leaderboard public`, `leaderboard private`, `leaderboard status`, or `leaderboard emoji <emoji>`")
	}

	switch strings.ToLower(parts[1]) {
	case "public":
		return leaderboardVisibilityCommand{Action: "set", Public: true}, true, nil
	case "private":
		return leaderboardVisibilityCommand{Action: "set", Public: false}, true, nil
	case "status":
		return leaderboardVisibilityCommand{Action: "show"}, true, nil
	default:
		return leaderboardVisibilityCommand{}, true, errors.New("use `leaderboard public`, `leaderboard private`, `leaderboard status`, or `leaderboard emoji <emoji>`")
	}
}

func validateLeaderboardEmoji(emoji string) error {
	if emojiShortcodePattern.MatchString(emoji) || discordEmojiPattern.MatchString(emoji) {
		return nil
	}

	runes := []rune(emoji)
	if len(runes) == 0 || len(runes) > 16 {
		return errors.New("leaderboard emoji must be a Unicode emoji, Discord emoji, or `:shortcode:`")
	}
	hasUnicode := false
	for _, character := range runes {
		if unicode.IsSpace(character) || strings.ContainsRune("@`*_~|\\", character) {
			return errors.New("leaderboard emoji contains unsupported characters")
		}
		if character > unicode.MaxASCII {
			hasUnicode = true
		}
	}
	if !hasUnicode {
		return errors.New("leaderboard emoji must be a Unicode emoji, Discord emoji, or `:shortcode:`")
	}
	return nil
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
		winnerEmoji, emojiErr := getGuildLeaderboardWinnerEmoji(m.GuildID)
		if emojiErr != nil {
			fmt.Printf("Unable to load leaderboard winner emoji for guild %s: %s\n", m.GuildID, emojiErr)
			sendBotMessage(s, m.ChannelID, "I couldn't load this server's leaderboard setting.")
			return true
		}
		if public {
			sendBotMessage(s, m.ChannelID, fmt.Sprintf("This server's leaderboard is public. Top-user emoji: %s", winnerEmoji))
		} else {
			sendBotMessage(s, m.ChannelID, fmt.Sprintf("This server's leaderboard is private. Top-user emoji: %s", winnerEmoji))
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
		sendBotMessage(s, m.ChannelID, "Only server administrators, Mainduck, and the DuckBot manager role can change leaderboard settings.")
		return true
	}

	if command.Action == "set-emoji" || command.Action == "reset-emoji" {
		winnerEmoji := command.Emoji
		if command.Action == "reset-emoji" {
			winnerEmoji = defaultWinnerEmoji
		}
		if saveErr := saveGuildLeaderboardWinnerEmoji(m.GuildID, winnerEmoji); saveErr != nil {
			fmt.Printf("Unable to save leaderboard winner emoji for guild %s: %s\n", m.GuildID, saveErr)
			sendBotMessage(s, m.ChannelID, "I couldn't save this server's leaderboard setting.")
			return true
		}
		sendBotMessage(s, m.ChannelID, fmt.Sprintf("The top-user emoji is now %s", winnerEmoji))
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
		if isStorageStatus(err, http.StatusNotFound) {
			return defaultLeaderboardPublic, nil
		}
		return false, fmt.Errorf("get leaderboard config entity: %w", err)
	}

	return leaderboardPublicProperty(entity.Properties)
}

func leaderboardPublicProperty(properties map[string]interface{}) (bool, error) {
	publicValue := properties["Public"]
	if publicValue == nil {
		return defaultLeaderboardPublic, nil
	}
	public, ok := publicValue.(bool)
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
	if err := entity.InsertOrMerge(nil); err != nil {
		return fmt.Errorf("save leaderboard config entity: %w", err)
	}
	return nil
}

func getGuildLeaderboardWinnerEmoji(guildID string) (string, error) {
	if guildID == "" {
		return defaultWinnerEmoji, nil
	}

	entity, err := leaderboardConfigEntity(guildID)
	if err != nil {
		return "", err
	}
	err = entity.Get(30, storage.NoMetadata, nil)
	if err != nil {
		if isStorageStatus(err, http.StatusNotFound) {
			return defaultWinnerEmoji, nil
		}
		return "", fmt.Errorf("get leaderboard config entity: %w", err)
	}

	winnerEmoji, ok := entity.Properties["WinnerEmoji"].(string)
	if !ok || winnerEmoji == "" {
		return defaultWinnerEmoji, nil
	}
	if err := validateLeaderboardEmoji(winnerEmoji); err != nil {
		return "", fmt.Errorf("leaderboard config entity has invalid WinnerEmoji property: %w", err)
	}
	return winnerEmoji, nil
}

func saveGuildLeaderboardWinnerEmoji(guildID string, winnerEmoji string) error {
	if guildID == "" {
		return errors.New("leaderboard settings require a Discord server")
	}
	if err := validateLeaderboardEmoji(winnerEmoji); err != nil {
		return err
	}

	entity, err := leaderboardConfigEntity(guildID)
	if err != nil {
		return err
	}
	entity.Properties = map[string]interface{}{"WinnerEmoji": winnerEmoji}
	if err := entity.InsertOrMerge(nil); err != nil {
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
