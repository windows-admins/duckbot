package main

import "testing"

func TestLeaderboardDefaultsToPrivate(t *testing.T) {
	public, err := isGuildLeaderboardPublic("")
	if err != nil {
		t.Fatalf("default visibility returned an error: %v", err)
	}
	if public {
		t.Fatal("an unconfigured leaderboard must be private")
	}
}

func TestParseLeaderboardVisibilityCommand(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantAction string
		wantPublic bool
		wantFound  bool
		wantErr    bool
	}{
		{
			name:      "plain leaderboard keeps existing behavior",
			content:   "<@123> leaderboard",
			wantFound: false,
		},
		{
			name:       "set public",
			content:    "<@123> leaderboard public",
			wantAction: "set",
			wantPublic: true,
			wantFound:  true,
		},
		{
			name:       "set private case insensitive",
			content:    "<@!123> LEADERBOARD PRIVATE",
			wantAction: "set",
			wantFound:  true,
		},
		{
			name:       "show status",
			content:    "<@123> leaderboard status",
			wantAction: "show",
			wantFound:  true,
		},
		{
			name:      "reject unsupported action",
			content:   "<@123> leaderboard enabled",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "not leaderboard command",
			content:   "<@123> counter",
			wantFound: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, found, err := parseLeaderboardVisibilityCommand(test.content, "123")
			if found != test.wantFound {
				t.Fatalf("found = %t, want %t", found, test.wantFound)
			}
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
			if command.Action != test.wantAction {
				t.Errorf("action = %q, want %q", command.Action, test.wantAction)
			}
			if command.Public != test.wantPublic {
				t.Errorf("public = %t, want %t", command.Public, test.wantPublic)
			}
		})
	}
}
