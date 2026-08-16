package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/bwmarrin/discordgo"
)

const (
	leaderboardDeleteConfigPrefix = "DELETE-"
	leaderboardDeleteTimeout      = 5 * time.Minute
)

var (
	errLeaderboardItemNotFound = errors.New("leaderboard item not found")
	errLeaderboardItemChanged  = errors.New("leaderboard item changed")
	errNoPendingDeletion       = errors.New("no pending leaderboard deletion")
)

type leaderboardDeleteCommand struct {
	Action string
	Item   string
}

type pendingLeaderboardDeletion struct {
	Item      string
	ETag      string
	ExpiresAt time.Time
}

type leaderboardThingSnapshot struct {
	PointItem
	ETag string
}

func parseLeaderboardDeleteCommand(content string, botID string) (leaderboardDeleteCommand, bool, error) {
	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	parts := strings.Fields(strings.TrimSpace(content))

	if len(parts) == 2 && strings.EqualFold(parts[0], "leaderboard") && strings.EqualFold(parts[1], "confirm-delete") {
		return leaderboardDeleteCommand{Action: "confirm"}, true, nil
	}
	if len(parts) == 2 && strings.EqualFold(parts[0], "leaderboard") && strings.EqualFold(parts[1], "cancel-delete") {
		return leaderboardDeleteCommand{Action: "cancel"}, true, nil
	}
	if len(parts) < 2 || !strings.EqualFold(parts[0], "leaderboard") || !strings.EqualFold(parts[1], "delete") {
		return leaderboardDeleteCommand{}, false, nil
	}
	if len(parts) < 3 {
		return leaderboardDeleteCommand{}, true, errors.New("use `leaderboard delete <item>`, `leaderboard confirm-delete`, or `leaderboard cancel-delete`")
	}

	item, err := normalizeLeaderboardItem(strings.Join(parts[2:], " "))
	if err != nil {
		return leaderboardDeleteCommand{}, true, err
	}
	return leaderboardDeleteCommand{Action: "request", Item: item}, true, nil
}

func normalizeLeaderboardItem(item string) (string, error) {
	item = strings.ToUpper(strings.TrimSpace(item))
	if item == "" {
		return "", errors.New("leaderboard item cannot be empty")
	}
	if len([]rune(item)) > 255 {
		return "", errors.New("leaderboard item cannot exceed 255 characters")
	}
	if strings.ContainsAny(item, "/\\#?") {
		return "", errors.New("leaderboard item contains unsupported characters")
	}
	return item, nil
}

func handleLeaderboardDeleteCommand(s *discordgo.Session, m *discordgo.Message) bool {
	command, matched, err := parseLeaderboardDeleteCommand(m.Content, s.State.User.ID)
	if !matched {
		return false
	}
	if err != nil {
		sendBotMessage(s, m.ChannelID, err.Error())
		return true
	}

	authorized, permissionErr := canManageGuildSettings(s, m)
	if permissionErr != nil {
		fmt.Printf("Unable to check leaderboard deletion permissions for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, permissionErr)
		sendBotMessage(s, m.ChannelID, "I couldn't verify your server permissions.")
		return true
	}
	if !authorized {
		sendBotMessage(s, m.ChannelID, "Only server administrators, Mainduck, and the DuckBot manager role can delete leaderboard items.")
		return true
	}

	switch command.Action {
	case "request":
		item, getErr := getLeaderboardThing(m.GuildID, command.Item)
		if errors.Is(getErr, errLeaderboardItemNotFound) {
			sendBotMessage(s, m.ChannelID, fmt.Sprintf("`%s` is not a thing on this server's leaderboard.", escapeDiscordMarkdown(command.Item)))
			return true
		}
		if getErr != nil {
			fmt.Printf("Unable to load leaderboard item %s in guild %s: %s\n", command.Item, m.GuildID, getErr)
			sendBotMessage(s, m.ChannelID, "I couldn't load that leaderboard item.")
			return true
		}

		pending := pendingLeaderboardDeletion{
			Item:      item.Item,
			ETag:      item.ETag,
			ExpiresAt: time.Now().UTC().Add(leaderboardDeleteTimeout),
		}
		if saveErr := savePendingLeaderboardDeletion(m.GuildID, m.Author.ID, pending); saveErr != nil {
			fmt.Printf("Unable to save pending leaderboard deletion for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, saveErr)
			sendBotMessage(s, m.ChannelID, "I couldn't create the deletion confirmation.")
			return true
		}

		sendBotMessage(s, m.ChannelID, fmt.Sprintf(
			"Delete `%s` with %s points? Run <@%s> `leaderboard confirm-delete` within five minutes, or `leaderboard cancel-delete`.",
			escapeDiscordMarkdown(item.Item),
			formatPointTotal(item.Points),
			s.State.User.ID,
		))
		return true

	case "confirm":
		pending, getErr := getPendingLeaderboardDeletion(m.GuildID, m.Author.ID, time.Now().UTC())
		if errors.Is(getErr, errNoPendingDeletion) {
			sendBotMessage(s, m.ChannelID, "You do not have a pending leaderboard deletion.")
			return true
		}
		if getErr != nil {
			fmt.Printf("Unable to load pending leaderboard deletion for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, getErr)
			sendBotMessage(s, m.ChannelID, "I couldn't load your deletion confirmation.")
			return true
		}

		if deleteErr := deleteLeaderboardThing(m.GuildID, pending.Item, pending.ETag); deleteErr != nil {
			if errors.Is(deleteErr, errLeaderboardItemNotFound) {
				sendBotMessage(s, m.ChannelID, "That leaderboard item no longer exists.")
			} else if errors.Is(deleteErr, errLeaderboardItemChanged) {
				sendBotMessage(s, m.ChannelID, "That leaderboard item changed after you requested deletion. Review it and request deletion again.")
			} else {
				fmt.Printf("Unable to delete leaderboard item %s in guild %s: %s\n", pending.Item, m.GuildID, deleteErr)
				sendBotMessage(s, m.ChannelID, "I couldn't delete that leaderboard item.")
			}
			return true
		}
		if clearErr := clearPendingLeaderboardDeletion(m.GuildID, m.Author.ID); clearErr != nil {
			fmt.Printf("Unable to clear pending leaderboard deletion for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, clearErr)
		}
		sendBotMessage(s, m.ChannelID, fmt.Sprintf("`%s` was deleted from the leaderboard.", escapeDiscordMarkdown(pending.Item)))
		return true

	case "cancel":
		if clearErr := clearPendingLeaderboardDeletion(m.GuildID, m.Author.ID); clearErr != nil {
			if errors.Is(clearErr, errNoPendingDeletion) {
				sendBotMessage(s, m.ChannelID, "You do not have a pending leaderboard deletion.")
			} else {
				fmt.Printf("Unable to cancel pending leaderboard deletion for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, clearErr)
				sendBotMessage(s, m.ChannelID, "I couldn't cancel your deletion confirmation.")
			}
			return true
		}
		sendBotMessage(s, m.ChannelID, "Your pending leaderboard deletion was cancelled.")
		return true
	}

	return true
}

func getLeaderboardThing(guildID string, item string) (leaderboardThingSnapshot, error) {
	entity, err := leaderboardThingEntity(guildID, item)
	if err != nil {
		return leaderboardThingSnapshot{}, err
	}
	if err := retryStorageThrottling(func() error {
		return entity.Get(30, storage.FullMetadata, nil)
	}); err != nil {
		if isStorageStatus(err, http.StatusNotFound) {
			return leaderboardThingSnapshot{}, errLeaderboardItemNotFound
		}
		return leaderboardThingSnapshot{}, fmt.Errorf("get leaderboard item: %w", err)
	}

	isUser, ok := entity.Properties["isUser"].(bool)
	if !ok {
		return leaderboardThingSnapshot{}, errors.New("leaderboard item has invalid isUser property")
	}
	if isUser {
		return leaderboardThingSnapshot{}, errLeaderboardItemNotFound
	}
	points, valueErr := pointPropertyValue(entity.Properties["Points"])
	if valueErr != nil {
		return leaderboardThingSnapshot{}, valueErr
	}
	if entity.OdataEtag == "" {
		return leaderboardThingSnapshot{}, errors.New("leaderboard item is missing an ETag")
	}
	return leaderboardThingSnapshot{
		PointItem: PointItem{Item: entity.RowKey, Points: points, IsUser: false},
		ETag:      entity.OdataEtag,
	}, nil
}

func deleteLeaderboardThing(guildID string, item string, expectedETag string) error {
	if expectedETag == "" {
		return errLeaderboardItemChanged
	}
	entity, err := leaderboardThingEntity(guildID, item)
	if err != nil {
		return err
	}
	entity.OdataEtag = expectedETag
	if err := retryStorageThrottling(func() error {
		return entity.Delete(false, nil)
	}); err != nil {
		if isStorageStatus(err, http.StatusNotFound) {
			return errLeaderboardItemNotFound
		}
		if isETagConflict(err) {
			return errLeaderboardItemChanged
		}
		return fmt.Errorf("delete leaderboard item: %w", err)
	}
	apiLeaderboardCache.invalidateGuild(guildID)
	return nil
}

func leaderboardThingEntity(guildID string, item string) (*storage.Entity, error) {
	if !isDiscordGuildID(guildID) {
		return nil, errors.New("invalid Discord guild ID")
	}
	normalizedItem, err := normalizeLeaderboardItem(item)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	return table.GetEntityReference(guildID, normalizedItem), nil
}

func savePendingLeaderboardDeletion(guildID string, userID string, pending pendingLeaderboardDeletion) error {
	entity, err := pendingLeaderboardDeletionEntity(guildID, userID)
	if err != nil {
		return err
	}
	entity.Properties = map[string]interface{}{
		"Item":      pending.Item,
		"ETag":      pending.ETag,
		"ExpiresAt": pending.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if err := retryStorageThrottling(func() error {
		return entity.InsertOrReplace(nil)
	}); err != nil {
		return fmt.Errorf("save pending leaderboard deletion: %w", err)
	}
	return nil
}

func getPendingLeaderboardDeletion(guildID string, userID string, now time.Time) (pendingLeaderboardDeletion, error) {
	entity, err := pendingLeaderboardDeletionEntity(guildID, userID)
	if err != nil {
		return pendingLeaderboardDeletion{}, err
	}
	if err := retryStorageThrottling(func() error {
		return entity.Get(30, storage.NoMetadata, nil)
	}); err != nil {
		if isStorageStatus(err, http.StatusNotFound) {
			return pendingLeaderboardDeletion{}, errNoPendingDeletion
		}
		return pendingLeaderboardDeletion{}, fmt.Errorf("get pending leaderboard deletion: %w", err)
	}

	item, itemOK := entity.Properties["Item"].(string)
	etag, etagOK := entity.Properties["ETag"].(string)
	expiresAtValue, expiresOK := entity.Properties["ExpiresAt"].(string)
	if !itemOK || !etagOK || etag == "" || !expiresOK {
		return pendingLeaderboardDeletion{}, errors.New("pending leaderboard deletion has invalid properties")
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtValue)
	if err != nil {
		return pendingLeaderboardDeletion{}, fmt.Errorf("parse pending leaderboard deletion expiry: %w", err)
	}
	if !now.Before(expiresAt) {
		_ = entity.Delete(true, nil)
		return pendingLeaderboardDeletion{}, errNoPendingDeletion
	}
	return pendingLeaderboardDeletion{Item: item, ETag: etag, ExpiresAt: expiresAt}, nil
}

func clearPendingLeaderboardDeletion(guildID string, userID string) error {
	entity, err := pendingLeaderboardDeletionEntity(guildID, userID)
	if err != nil {
		return err
	}
	if err := retryStorageThrottling(func() error {
		return entity.Delete(true, nil)
	}); err != nil {
		if isStorageStatus(err, http.StatusNotFound) {
			return errNoPendingDeletion
		}
		return fmt.Errorf("clear pending leaderboard deletion: %w", err)
	}
	return nil
}

func pendingLeaderboardDeletionEntity(guildID string, userID string) (*storage.Entity, error) {
	if !isDiscordGuildID(guildID) || !isDiscordGuildID(userID) {
		return nil, errors.New("invalid Discord ID")
	}

	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	return table.GetEntityReference(guildID+"-CONFIG", leaderboardDeleteConfigPrefix+userID), nil
}

func isStorageStatus(err error, statusCode int) bool {
	var storageErr storage.AzureStorageServiceError
	if errors.As(err, &storageErr) {
		return storageErr.StatusCode == statusCode
	}

	var storageErrPointer *storage.AzureStorageServiceError
	return errors.As(err, &storageErrPointer) && storageErrPointer.StatusCode == statusCode
}

func isETagConflict(err error) bool {
	if err == nil {
		return false
	}
	// Azure SDK v52 formats 412 errors with %v, so errors.As cannot unwrap them.
	return isStorageStatus(err, http.StatusPreconditionFailed) ||
		strings.Contains(err.Error(), "Etag didn't match")
}

func retryStorageThrottling(operation func() error) error {
	var err error
	for attempt := 0; attempt < storageOperationAttempts; attempt++ {
		err = operation()
		if !isStorageStatus(err, http.StatusTooManyRequests) {
			return err
		}
		waitForStorageRetry(attempt)
	}
	return err
}
