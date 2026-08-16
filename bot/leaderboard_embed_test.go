package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatLeaderboardEntries(t *testing.T) {
	entries := []PointItem{
		{Item: "BEANS", Points: 1234},
		{Item: "<:EDGE:854378059272421486", Points: -2},
	}

	got := formatLeaderboardEntries(entries, false, "")
	if !strings.Contains(got, "**1.** BEANS — **1,234**") {
		t.Errorf("missing formatted first entry: %q", got)
	}
	if !strings.Contains(got, "<:EDGE:854378059272421486>") {
		t.Errorf("Discord emoji was not normalized: %q", got)
	}
}

func TestFormatMemberLeaderboardEntry(t *testing.T) {
	got := formatLeaderboardEntries([]PointItem{{Item: "281125480072085515", Points: 42}}, true, ":high_heel:")
	if !strings.Contains(got, "<@281125480072085515>") {
		t.Errorf("member ID was not formatted as a mention: %q", got)
	}
	if !strings.Contains(got, ":high_heel:") {
		t.Errorf("top member did not include winner emoji: %q", got)
	}
}

func TestFormatEmptyLeaderboard(t *testing.T) {
	if got := formatLeaderboardEntries(nil, false, ""); got != "_No points yet._" {
		t.Errorf("empty leaderboard = %q", got)
	}
}

func TestFormatBottomLeaderboardEntries(t *testing.T) {
	entries := make([]PointItem, 12)
	for index := range entries {
		entries[index] = PointItem{
			Item:   fmt.Sprintf("%018d", index+1),
			Points: float64(12 - index),
		}
	}

	got := formatBottomLeaderboardEntries(entries, true)
	lines := strings.Split(got, "\n")
	if len(lines) != discordLeaderboardLimit {
		t.Fatalf("bottom leaderboard has %d entries, want %d: %q", len(lines), discordLeaderboardLimit, got)
	}
	if !strings.Contains(lines[0], "**12.**") || !strings.Contains(lines[0], "— **1**") {
		t.Errorf("worst member was not listed first: %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "**3.**") || !strings.Contains(lines[len(lines)-1], "— **10**") {
		t.Errorf("bottom-ten boundary was incorrect: %q", lines[len(lines)-1])
	}
}

func TestFormatEmptyBottomLeaderboard(t *testing.T) {
	if got := formatBottomLeaderboardEntries(nil, false); got != "_No points yet._" {
		t.Errorf("empty bottom leaderboard = %q", got)
	}
}

func TestBuildLeaderboardEmbedPublicLink(t *testing.T) {
	publicEmbed := buildLeaderboardEmbed("618712310185197588", "WinAdmins", nil, nil, defaultWinnerEmoji, true, nil)
	if publicEmbed.URL == "" || !strings.Contains(publicEmbed.Description, "View the full leaderboard") {
		t.Fatal("public leaderboard did not include the website link")
	}

	privateEmbed := buildLeaderboardEmbed("618712310185197588", "WinAdmins", nil, nil, defaultWinnerEmoji, false, nil)
	if privateEmbed.URL != "" || strings.Contains(privateEmbed.Description, "View the full leaderboard") {
		t.Fatal("private leaderboard exposed the website link")
	}
}

func TestBuildLeaderboardEmbedIncludesWorstMembers(t *testing.T) {
	things := []PointItem{
		{Item: "BEANS", Points: 12},
		{Item: "EDGE", Points: -10},
	}
	members := []PointItem{
		{Item: "281125480072085515", Points: 42},
		{Item: "618712310185197588", Points: -5},
	}

	embed := buildLeaderboardEmbed("618712310185197588", "WinAdmins", things, members, defaultWinnerEmoji, false, nil)
	if len(embed.Fields) != 4 {
		t.Fatalf("embed has %d fields, want 4", len(embed.Fields))
	}
	if embed.Fields[2].Name != "Worst of the Worst: Things" {
		t.Fatalf("bottom things field name = %q", embed.Fields[2].Name)
	}
	if !strings.HasPrefix(embed.Fields[2].Value, "**2.** EDGE — **-10**") {
		t.Errorf("bottom things did not start with the worst item: %q", embed.Fields[2].Value)
	}
	if embed.Fields[3].Name != "Worst of the Worst: Members" {
		t.Fatalf("bottom members field name = %q", embed.Fields[3].Name)
	}
	if !strings.HasPrefix(embed.Fields[3].Value, "**2.** <@618712310185197588> — **-5**") {
		t.Errorf("bottom members did not start with the worst member: %q", embed.Fields[3].Value)
	}
}
