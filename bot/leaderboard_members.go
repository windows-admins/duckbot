package main

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

const discordGuildMembersPageSize = 1000

type guildMemberIDLoader func(string) (map[string]struct{}, error)
type guildMemberChecker func(string, string) (bool, error)
type discordGuildMembersPageLoader func(string, string, int) ([]*discordgo.Member, error)

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

func discordGuildMemberIDs(session *discordgo.Session, guildID string) (map[string]struct{}, error) {
	return loadDiscordGuildMemberIDs(guildID, session.GuildMembers)
}

func loadDiscordGuildMemberIDs(guildID string, loadPage discordGuildMembersPageLoader) (map[string]struct{}, error) {
	memberIDs := make(map[string]struct{})
	after := ""
	for {
		members, err := loadPage(guildID, after, discordGuildMembersPageSize)
		if err != nil {
			return nil, fmt.Errorf("list Discord guild members: %w", err)
		}

		for _, member := range members {
			if member != nil && member.User != nil && isDiscordSnowflake(member.User.ID) {
				memberIDs[member.User.ID] = struct{}{}
			}
		}
		if len(members) < discordGuildMembersPageSize {
			return memberIDs, nil
		}

		lastMember := members[len(members)-1]
		if lastMember == nil || lastMember.User == nil || !isDiscordSnowflake(lastMember.User.ID) || lastMember.User.ID == after {
			return nil, fmt.Errorf("Discord guild member page has an invalid continuation member")
		}
		after = lastMember.User.ID
	}
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

func filterCurrentGuildMembers(entries []PointItem, memberIDs map[string]struct{}) []PointItem {
	filtered := make([]PointItem, 0, len(entries))
	for _, entry := range entries {
		if _, exists := memberIDs[entry.Item]; exists {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func filterTopCurrentGuildMembers(guildID string, entries []PointItem, limit int, memberExists guildMemberChecker) ([]PointItem, error) {
	filtered := make([]PointItem, 0, limit)
	for _, entry := range entries {
		exists, err := memberExists(guildID, entry.Item)
		if err != nil {
			return nil, err
		}
		if exists {
			filtered = append(filtered, entry)
			if len(filtered) == limit {
				break
			}
		}
	}
	return filtered, nil
}
