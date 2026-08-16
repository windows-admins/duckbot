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
	defaultCounterSingular = "Loch Ness Goose"
	defaultCounterPlural   = "Loch Ness Geese"
	counterConfigRow       = "COUNTER"
	maxCounterNameLength   = 40
)

type counterConfig struct {
	Singular string
	Plural   string
}

type counterCommand struct {
	Action   string
	Singular string
	Plural   string
}

func defaultCounter() counterConfig {
	return counterConfig{
		Singular: defaultCounterSingular,
		Plural:   defaultCounterPlural,
	}
}

func counterNameForScore(counter counterConfig, score int) string {
	if score == 1 {
		return counter.Singular
	}
	return counter.Plural
}

func parseCounterCommand(content string, botID string) (counterCommand, bool, error) {
	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	content = strings.TrimSpace(content)

	parts := strings.Fields(content)
	if len(parts) == 0 || !strings.EqualFold(parts[0], "counter") {
		return counterCommand{}, false, nil
	}

	arguments := strings.TrimSpace(content[len(parts[0]):])
	if arguments == "" {
		return counterCommand{Action: "show"}, true, nil
	}
	if strings.EqualFold(arguments, "reset") {
		return counterCommand{Action: "reset"}, true, nil
	}

	names := strings.SplitN(arguments, "|", 2)
	if len(names) != 2 {
		return counterCommand{}, true, errors.New("use `counter singular | plural`")
	}

	singular := strings.TrimSpace(names[0])
	plural := strings.TrimSpace(names[1])
	if err := validateCounterName(singular); err != nil {
		return counterCommand{}, true, fmt.Errorf("invalid singular name: %w", err)
	}
	if err := validateCounterName(plural); err != nil {
		return counterCommand{}, true, fmt.Errorf("invalid plural name: %w", err)
	}

	return counterCommand{
		Action:   "set",
		Singular: singular,
		Plural:   plural,
	}, true, nil
}

func validateCounterName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if len([]rune(name)) > maxCounterNameLength {
		return fmt.Errorf("name cannot exceed %d characters", maxCounterNameLength)
	}
	if strings.ContainsAny(name, "@<>#`\r\n") {
		return errors.New("name contains unsupported characters")
	}
	return nil
}

func handleCounterCommand(s *discordgo.Session, m *discordgo.Message) bool {
	command, matched, err := parseCounterCommand(m.Content, s.State.User.ID)
	if !matched {
		return false
	}
	if err != nil {
		sendBotMessage(s, m.ChannelID, fmt.Sprintf("%s. Usage: `@DuckBot counter singular | plural`", err))
		return true
	}

	if command.Action == "show" {
		counter, getErr := getCounter(m.GuildID)
		if getErr != nil {
			fmt.Printf("Unable to load counter for guild %s: %s\n", m.GuildID, getErr)
			sendBotMessage(s, m.ChannelID, "I couldn't load this server's counter.")
			return true
		}
		sendBotMessage(s, m.ChannelID, fmt.Sprintf("This server's counter is `%s` / `%s`.", counter.Singular, counter.Plural))
		return true
	}

	authorized, permissionErr := canManageGuildSettings(s, m)
	if permissionErr != nil {
		fmt.Printf("Unable to check counter permissions for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, permissionErr)
		sendBotMessage(s, m.ChannelID, "I couldn't verify your server permissions.")
		return true
	}
	if !authorized {
		sendBotMessage(s, m.ChannelID, "Only server administrators and the DuckBot owner can customize the counter.")
		return true
	}

	counter := defaultCounter()
	if command.Action == "set" {
		counter = counterConfig{Singular: command.Singular, Plural: command.Plural}
	}
	if saveErr := saveCounter(m.GuildID, counter); saveErr != nil {
		fmt.Printf("Unable to save counter for guild %s: %s\n", m.GuildID, saveErr)
		sendBotMessage(s, m.ChannelID, "I couldn't save this server's counter.")
		return true
	}

	sendBotMessage(s, m.ChannelID, fmt.Sprintf("Counter updated to `%s` / `%s`.", counter.Singular, counter.Plural))
	return true
}

func sendBotMessage(s *discordgo.Session, channelID string, content string) {
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	})
	if err != nil {
		fmt.Printf("Unable to send bot response to channel %s: %s\n", channelID, err)
	}
}

func getCounter(guildID string) (counterConfig, error) {
	if guildID == "" {
		return defaultCounter(), nil
	}

	entity, err := counterEntity(guildID)
	if err != nil {
		return counterConfig{}, err
	}
	err = entity.Get(30, storage.NoMetadata, nil)
	if err != nil {
		var storageErr *storage.AzureStorageServiceError
		if errors.As(err, &storageErr) && storageErr.StatusCode == http.StatusNotFound {
			return defaultCounter(), nil
		}
		return counterConfig{}, fmt.Errorf("get counter entity: %w", err)
	}

	singular, singularOK := entity.Properties["Singular"].(string)
	plural, pluralOK := entity.Properties["Plural"].(string)
	if !singularOK || !pluralOK {
		return counterConfig{}, errors.New("counter entity has invalid properties")
	}
	return counterConfig{Singular: singular, Plural: plural}, nil
}

func saveCounter(guildID string, counter counterConfig) error {
	if guildID == "" {
		return errors.New("counter settings require a Discord server")
	}

	entity, err := counterEntity(guildID)
	if err != nil {
		return err
	}
	entity.Properties = map[string]interface{}{
		"Singular": counter.Singular,
		"Plural":   counter.Plural,
	}
	if err := entity.InsertOrReplace(nil); err != nil {
		return fmt.Errorf("save counter entity: %w", err)
	}
	return nil
}

func counterEntity(guildID string) (*storage.Entity, error) {
	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	return table.GetEntityReference(guildID+"-CONFIG", counterConfigRow), nil
}
