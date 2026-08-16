package main

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/bwmarrin/discordgo"
)

const (
	managerConfigRow = "MANAGER"
	mainduckUserID   = "281125480072085515"
)

var roleMentionPattern = regexp.MustCompile(`^<@&(\d{17,20})>$`)

type managerCommand struct {
	Action string
	RoleID string
}

func parseManagerCommand(content string, botID string) (managerCommand, bool, error) {
	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	parts := strings.Fields(strings.TrimSpace(content))

	if len(parts) == 0 || !strings.EqualFold(parts[0], "manager") {
		return managerCommand{}, false, nil
	}
	if len(parts) == 1 || (len(parts) == 2 && strings.EqualFold(parts[1], "status")) {
		return managerCommand{Action: "show"}, true, nil
	}
	if len(parts) == 2 && strings.EqualFold(parts[1], "reset") {
		return managerCommand{Action: "reset"}, true, nil
	}
	if len(parts) != 2 {
		return managerCommand{}, true, errors.New("use `manager @role`, `manager reset`, or `manager status`")
	}

	match := roleMentionPattern.FindStringSubmatch(parts[1])
	if match == nil {
		return managerCommand{}, true, errors.New("manager must be a role mention")
	}
	return managerCommand{Action: "set", RoleID: match[1]}, true, nil
}

func handleManagerCommand(s *discordgo.Session, m *discordgo.Message) bool {
	command, matched, err := parseManagerCommand(m.Content, s.State.User.ID)
	if !matched {
		return false
	}
	if err != nil {
		sendBotMessage(s, m.ChannelID, err.Error())
		return true
	}

	if command.Action == "show" {
		roleID, getErr := getGuildManagerRole(m.GuildID)
		if getErr != nil {
			fmt.Printf("Unable to load manager role for guild %s: %s\n", m.GuildID, getErr)
			sendBotMessage(s, m.ChannelID, "I couldn't load this server's manager role.")
			return true
		}
		if roleID == "" {
			sendBotMessage(s, m.ChannelID, "This server does not have a DuckBot manager role.")
		} else {
			sendBotMessage(s, m.ChannelID, fmt.Sprintf("This server's DuckBot manager role is <@&%s>.", roleID))
		}
		return true
	}

	authorized, permissionErr := canDesignateGuildManager(s, m)
	if permissionErr != nil {
		fmt.Printf("Unable to check manager permissions for user %s in guild %s: %s\n", m.Author.ID, m.GuildID, permissionErr)
		sendBotMessage(s, m.ChannelID, "I couldn't verify your server permissions.")
		return true
	}
	if !authorized {
		sendBotMessage(s, m.ChannelID, "Only server administrators and Mainduck can designate the DuckBot manager role.")
		return true
	}

	roleID := command.RoleID
	if command.Action == "set" {
		if roleID == m.GuildID {
			sendBotMessage(s, m.ChannelID, "The @everyone role cannot manage DuckBot.")
			return true
		}
		if _, roleErr := getDiscordGuildRole(s, m.GuildID, roleID); roleErr != nil {
			fmt.Printf("Unable to verify manager role %s in guild %s: %s\n", roleID, m.GuildID, roleErr)
			sendBotMessage(s, m.ChannelID, "I couldn't verify that role in this server.")
			return true
		}
	}

	if saveErr := saveGuildManagerRole(m.GuildID, roleID); saveErr != nil {
		fmt.Printf("Unable to save manager role for guild %s: %s\n", m.GuildID, saveErr)
		sendBotMessage(s, m.ChannelID, "I couldn't save this server's manager role.")
		return true
	}

	if roleID == "" {
		sendBotMessage(s, m.ChannelID, "This server's DuckBot manager role has been reset.")
	} else {
		sendBotMessage(s, m.ChannelID, fmt.Sprintf("<@&%s> can now manage DuckBot settings.", roleID))
	}
	return true
}

func canDesignateGuildManager(s *discordgo.Session, m *discordgo.Message) (bool, error) {
	if m.Author.ID == mainduckUserID {
		return true, nil
	}
	if m.GuildID == "" {
		return false, nil
	}

	permissions, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID)
	if err != nil {
		return false, err
	}
	return permissions&discordgo.PermissionAdministrator != 0, nil
}

func canManageGuildSettings(s *discordgo.Session, m *discordgo.Message) (bool, error) {
	authorized, permissionErr := canDesignateGuildManager(s, m)
	if authorized {
		return true, nil
	}

	roleID, err := getGuildManagerRole(m.GuildID)
	if err != nil {
		return false, err
	}
	if roleID == "" {
		return false, permissionErr
	}

	member := m.Member
	if member == nil {
		member, err = s.GuildMember(m.GuildID, m.Author.ID)
		if err != nil {
			return false, fmt.Errorf("get Discord member: %w", err)
		}
	}
	if memberHasRole(member.Roles, roleID) {
		return true, nil
	}
	return false, permissionErr
}

func memberHasRole(roleIDs []string, roleID string) bool {
	for _, memberRoleID := range roleIDs {
		if memberRoleID == roleID {
			return true
		}
	}
	return false
}

func getDiscordGuildRole(s *discordgo.Session, guildID string, roleID string) (*discordgo.Role, error) {
	role, err := s.State.Role(guildID, roleID)
	if err == nil {
		return role, nil
	}

	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, fmt.Errorf("get Discord guild roles: %w", err)
	}
	for _, guildRole := range roles {
		if guildRole.ID == roleID {
			return guildRole, nil
		}
	}
	return nil, errors.New("role does not exist in guild")
}

func getGuildManagerRole(guildID string) (string, error) {
	if guildID == "" {
		return "", nil
	}

	entity, err := managerConfigEntity(guildID)
	if err != nil {
		return "", err
	}
	err = entity.Get(30, storage.NoMetadata, nil)
	if err != nil {
		var storageErr *storage.AzureStorageServiceError
		if errors.As(err, &storageErr) && storageErr.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", fmt.Errorf("get manager config entity: %w", err)
	}

	roleID, ok := entity.Properties["RoleID"].(string)
	if !ok {
		return "", errors.New("manager config entity has invalid RoleID property")
	}
	return roleID, nil
}

func saveGuildManagerRole(guildID string, roleID string) error {
	if guildID == "" {
		return errors.New("manager settings require a Discord server")
	}

	entity, err := managerConfigEntity(guildID)
	if err != nil {
		return err
	}
	entity.Properties = map[string]interface{}{"RoleID": roleID}
	if err := entity.InsertOrReplace(nil); err != nil {
		return fmt.Errorf("save manager config entity: %w", err)
	}
	return nil
}

func managerConfigEntity(guildID string) (*storage.Entity, error) {
	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	return table.GetEntityReference(guildID+"-CONFIG", managerConfigRow), nil
}
