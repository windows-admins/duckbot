package main

import (
	"strings"
	"testing"
)

func TestFormatLeaderboardEntries(t *testing.T) {
	entries := []PointItem{
		{Item: "BEANS", Points: 1234},
		{Item: "<:EDGE:854378059272421486", Points: -2},
	}

	got := formatLeaderboardEntries(entries, false)
	if !strings.Contains(got, "**1.** BEANS — **1,234**") {
		t.Errorf("missing formatted first entry: %q", got)
	}
	if !strings.Contains(got, "<:EDGE:854378059272421486>") {
		t.Errorf("Discord emoji was not normalized: %q", got)
	}
}

func TestFormatMemberLeaderboardEntry(t *testing.T) {
	got := formatLeaderboardEntries([]PointItem{{Item: "281125480072085515", Points: 42}}, true)
	if !strings.Contains(got, "<@281125480072085515>") {
		t.Errorf("member ID was not formatted as a mention: %q", got)
	}
}

func TestFormatEmptyLeaderboard(t *testing.T) {
	if got := formatLeaderboardEntries(nil, false); got != "_No points yet._" {
		t.Errorf("empty leaderboard = %q", got)
	}
}

func TestBuildLeaderboardEmbedPublicLink(t *testing.T) {
	publicEmbed := buildLeaderboardEmbed("618712310185197588", "WinAdmins", nil, nil, true, nil)
	if publicEmbed.URL == "" || !strings.Contains(publicEmbed.Description, "View the full leaderboard") {
		t.Fatal("public leaderboard did not include the website link")
	}

	privateEmbed := buildLeaderboardEmbed("618712310185197588", "WinAdmins", nil, nil, false, nil)
	if privateEmbed.URL != "" || strings.Contains(privateEmbed.Description, "View the full leaderboard") {
		t.Fatal("private leaderboard exposed the website link")
	}
}
