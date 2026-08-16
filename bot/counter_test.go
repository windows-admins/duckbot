package main

import "testing"

func TestParseCounterCommand(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantAction string
		wantFound  bool
		wantErr    bool
		singular   string
		plural     string
	}{
		{
			name:       "show",
			content:    "<@123> counter",
			wantAction: "show",
			wantFound:  true,
		},
		{
			name:       "set",
			content:    "<@!123> counter Rubber Duck | Rubber Ducks",
			wantAction: "set",
			wantFound:  true,
			singular:   "Rubber Duck",
			plural:     "Rubber Ducks",
		},
		{
			name:       "reset case insensitive",
			content:    "<@123> COUNTER RESET",
			wantAction: "reset",
			wantFound:  true,
		},
		{
			name:      "not a counter command",
			content:   "<@123> leaderboard",
			wantFound: false,
		},
		{
			name:      "missing separator",
			content:   "<@123> counter Duck",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "reject mentions",
			content:   "<@123> counter @everyone | Ducks",
			wantFound: true,
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, found, err := parseCounterCommand(test.content, "123")
			if found != test.wantFound {
				t.Fatalf("found = %t, want %t", found, test.wantFound)
			}
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
			if command.Action != test.wantAction {
				t.Errorf("action = %q, want %q", command.Action, test.wantAction)
			}
			if command.Singular != test.singular {
				t.Errorf("singular = %q, want %q", command.Singular, test.singular)
			}
			if command.Plural != test.plural {
				t.Errorf("plural = %q, want %q", command.Plural, test.plural)
			}
		})
	}
}

func TestCounterNameForScore(t *testing.T) {
	counter := counterConfig{Singular: "Duck", Plural: "Ducks"}
	if got := counterNameForScore(counter, 1); got != "Duck" {
		t.Errorf("score 1 returned %q", got)
	}
	for _, score := range []int{-1, 0, 2} {
		if got := counterNameForScore(counter, score); got != "Ducks" {
			t.Errorf("score %d returned %q", score, got)
		}
	}
}
