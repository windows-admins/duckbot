package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestFilterCurrentGuildMembers(t *testing.T) {
	entries := []PointItem{
		{Item: "111111111111111111", Points: 20, IsUser: true},
		{Item: "222222222222222222", Points: 10, IsUser: true},
		{Item: "333333333333333333", Points: 5, IsUser: true},
	}

	filtered := filterCurrentGuildMembers(entries, map[string]struct{}{
		"111111111111111111": {},
		"333333333333333333": {},
	})
	if len(filtered) != 2 || filtered[0].Item != entries[0].Item || filtered[1].Item != entries[2].Item {
		t.Fatalf("filtered members = %#v", filtered)
	}
}

func TestFilterTopCurrentGuildMembersStopsAtLimit(t *testing.T) {
	entries := []PointItem{
		{Item: "111111111111111111", Points: 20, IsUser: true},
		{Item: "222222222222222222", Points: 10, IsUser: true},
		{Item: "333333333333333333", Points: 5, IsUser: true},
		{Item: "444444444444444444", Points: 1, IsUser: true},
	}
	lookups := 0

	filtered, err := filterTopCurrentGuildMembers(
		"618712310185197588",
		entries,
		2,
		func(guildID string, userID string) (bool, error) {
			lookups++
			if guildID != "618712310185197588" {
				t.Fatalf("guild ID = %q", guildID)
			}
			return userID != "222222222222222222", nil
		},
	)
	if err != nil {
		t.Fatalf("filter top members: %v", err)
	}
	if len(filtered) != 2 || filtered[0].Item != entries[0].Item || filtered[1].Item != entries[2].Item {
		t.Fatalf("filtered members = %#v", filtered)
	}
	if lookups != 3 {
		t.Fatalf("member lookups = %d, want 3", lookups)
	}
}

func TestFilterTopCurrentGuildMembersReturnsLookupError(t *testing.T) {
	lookupErr := errors.New("Discord unavailable")
	_, err := filterTopCurrentGuildMembers(
		"618712310185197588",
		[]PointItem{{Item: "111111111111111111", IsUser: true}},
		10,
		func(string, string) (bool, error) {
			return false, lookupErr
		},
	)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("filter error = %v, want %v", err, lookupErr)
	}
}

func TestLoadDiscordGuildMemberIDsPaginates(t *testing.T) {
	callCount := 0
	memberIDs, err := loadDiscordGuildMemberIDs("618712310185197588", func(guildID string, after string, limit int) ([]*discordgo.Member, error) {
		callCount++
		if guildID != "618712310185197588" || limit != discordGuildMembersPageSize {
			t.Fatalf("unexpected page request guild=%q limit=%d", guildID, limit)
		}

		if callCount == 1 {
			if after != "" {
				t.Fatalf("first page after = %q", after)
			}
			members := make([]*discordgo.Member, discordGuildMembersPageSize)
			for index := range members {
				members[index] = &discordgo.Member{User: &discordgo.User{ID: fmt.Sprintf("%018d", index+1)}}
			}
			return members, nil
		}
		if after != "000000000000001000" {
			t.Fatalf("second page after = %q", after)
		}
		return []*discordgo.Member{
			{User: &discordgo.User{ID: "111111111111111111"}},
		}, nil
	})
	if err != nil {
		t.Fatalf("load member IDs: %v", err)
	}
	if callCount != 2 || len(memberIDs) != discordGuildMembersPageSize+1 {
		t.Fatalf("calls=%d members=%d", callCount, len(memberIDs))
	}
}

func TestLoadDiscordGuildMemberIDsReturnsLookupError(t *testing.T) {
	lookupErr := errors.New("Discord unavailable")
	_, err := loadDiscordGuildMemberIDs("618712310185197588", func(string, string, int) ([]*discordgo.Member, error) {
		return nil, lookupErr
	})
	if !errors.Is(err, lookupErr) {
		t.Fatalf("load error = %v, want %v", err, lookupErr)
	}
}

func TestIsDiscordSnowflake(t *testing.T) {
	for _, value := range []string{"618712310185197588", "12345678901234567", "12345678901234567890"} {
		if !isDiscordSnowflake(value) {
			t.Errorf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "123", "not-a-user", "1234567890123456x"} {
		if isDiscordSnowflake(value) {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}
