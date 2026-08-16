package main

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type guildMemberChecker func(string, string) (bool, error)

func discordGuildMemberExists(session *discordgo.Session, guildID string, userID string) (bool, error) {
	if !isDiscordSnowflake(userID) {
		return false, nil
	}

	member, err := session.GuildMember(guildID, userID)
	if err == nil {
		return member != nil, nil
	}

	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) &&
		restErr.Message != nil &&
		restErr.Message.Code == discordgo.ErrCodeUnknownMember {
		return false, nil
	}
	return false, fmt.Errorf("get Discord guild member %s: %w", userID, err)
}

func isDiscordSnowflake(value string) bool {
	if len(value) < 17 || len(value) > 20 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func filterCurrentGuildMembers(guildID string, entries []PointItem, memberExists guildMemberChecker) ([]PointItem, error) {
	filtered := make([]PointItem, 0, len(entries))
	for _, entry := range entries {
		exists, err := memberExists(guildID, entry.Item)
		if err != nil {
			return nil, err
		}
		if exists {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}
