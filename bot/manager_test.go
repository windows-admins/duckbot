package main

import "testing"

func TestParseManagerCommand(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantAction string
		wantRoleID string
		wantFound  bool
		wantErr    bool
	}{
		{
			name:       "show",
			content:    "<@123> manager",
			wantAction: "show",
			wantFound:  true,
		},
		{
			name:       "show status",
			content:    "<@123> manager status",
			wantAction: "show",
			wantFound:  true,
		},
		{
			name:       "set role",
			content:    "<@!123> manager <@&987654321012345678>",
			wantAction: "set",
			wantRoleID: "987654321012345678",
			wantFound:  true,
		},
		{
			name:       "reset",
			content:    "<@123> MANAGER RESET",
			wantAction: "reset",
			wantFound:  true,
		},
		{
			name:      "reject user mention",
			content:   "<@123> manager <@987654321012345678>",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "not manager command",
			content:   "<@123> leaderboard",
			wantFound: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, found, err := parseManagerCommand(test.content, "123")
			if found != test.wantFound {
				t.Fatalf("found = %t, want %t", found, test.wantFound)
			}
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
			if command.Action != test.wantAction {
				t.Errorf("action = %q, want %q", command.Action, test.wantAction)
			}
			if command.RoleID != test.wantRoleID {
				t.Errorf("role ID = %q, want %q", command.RoleID, test.wantRoleID)
			}
		})
	}
}

func TestMemberHasRole(t *testing.T) {
	if !memberHasRole([]string{"111", "222"}, "222") {
		t.Fatal("expected member to have configured role")
	}
	if memberHasRole([]string{"111", "222"}, "333") {
		t.Fatal("expected member not to have unconfigured role")
	}
}

func TestManagerDefaultsToUnconfigured(t *testing.T) {
	roleID, err := getGuildManagerRole("")
	if err != nil {
		t.Fatalf("default manager role returned an error: %v", err)
	}
	if roleID != "" {
		t.Fatalf("default manager role = %q, want empty", roleID)
	}
}

func TestMainduckUserID(t *testing.T) {
	if mainduckUserID != "281125480072085515" {
		t.Fatalf("Mainduck user ID = %q", mainduckUserID)
	}
}
