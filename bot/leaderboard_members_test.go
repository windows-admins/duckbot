package main

import (
	"errors"
	"testing"
)

func TestFilterCurrentGuildMembers(t *testing.T) {
	entries := []PointItem{
		{Item: "111111111111111111", Points: 20, IsUser: true},
		{Item: "222222222222222222", Points: 10, IsUser: true},
		{Item: "333333333333333333", Points: 5, IsUser: true},
	}

	filtered, err := filterCurrentGuildMembers("618712310185197588", entries, func(guildID string, userID string) (bool, error) {
		if guildID != "618712310185197588" {
			t.Fatalf("guild ID = %q", guildID)
		}
		return userID != "222222222222222222", nil
	})
	if err != nil {
		t.Fatalf("filter members: %v", err)
	}
	if len(filtered) != 2 || filtered[0].Item != entries[0].Item || filtered[1].Item != entries[2].Item {
		t.Fatalf("filtered members = %#v", filtered)
	}
}

func TestFilterCurrentGuildMembersReturnsLookupError(t *testing.T) {
	lookupErr := errors.New("Discord unavailable")
	_, err := filterCurrentGuildMembers(
		"618712310185197588",
		[]PointItem{{Item: "111111111111111111", IsUser: true}},
		func(string, string) (bool, error) {
			return false, lookupErr
		},
	)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("filter error = %v, want %v", err, lookupErr)
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
