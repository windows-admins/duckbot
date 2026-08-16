package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestAllowDuckPointsCORS(t *testing.T) {
	handler := allowDuckPointsCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "primary domain",
			origin:         "https://duckpoints.com",
			expectedOrigin: "https://duckpoints.com",
		},
		{
			name:           "www domain",
			origin:         "https://www.duckpoints.com",
			expectedOrigin: "https://www.duckpoints.com",
		},
		{
			name:   "unapproved domain",
			origin: "https://example.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/guild/618712310185197588/things", nil)
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if actual := response.Header().Get("Access-Control-Allow-Origin"); actual != test.expectedOrigin {
				t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", test.expectedOrigin, actual)
			}
		})
	}
}

func TestGuildHandlerRequiresPublicGuild(t *testing.T) {
	tests := []struct {
		name            string
		guildID         string
		leaderboardType string
		isPublic        func(string) (bool, error)
		guildNameError  error
		expectedStatus  int
		expectedBody    string
	}{
		{
			name:            "public guild",
			guildID:         "618712310185197588",
			leaderboardType: "things",
			isPublic: func(string) (bool, error) {
				return true, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"guild":{"id":"618712310185197588","name":"Windows Admins"},"items":[{"item":"DUCK","points":5,"isUser":false}]}`,
		},
		{
			name:            "private guild",
			guildID:         "618712310185197588",
			leaderboardType: "things",
			isPublic: func(string) (bool, error) {
				return false, nil
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:            "visibility storage error",
			guildID:         "618712310185197588",
			leaderboardType: "members",
			isPublic: func(string) (bool, error) {
				return false, errors.New("storage unavailable")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:            "Discord guild lookup error",
			guildID:         "618712310185197588",
			leaderboardType: "things",
			isPublic: func(string) (bool, error) {
				return true, nil
			},
			guildNameError: errors.New("Discord unavailable"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:            "invalid guild ID",
			guildID:         "not-a-guild",
			leaderboardType: "things",
			isPublic: func(string) (bool, error) {
				t.Fatal("visibility should not be checked for an invalid guild ID")
				return false, nil
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/guild/"+test.guildID+"/"+test.leaderboardType, nil)
			request = mux.SetURLVars(request, map[string]string{
				"guild": test.guildID,
				"type":  test.leaderboardType,
			})
			response := httptest.NewRecorder()
			loadPoints := func(guildID string, getMembers bool) ([]PointItem, error) {
				if test.expectedStatus != http.StatusOK {
					t.Fatal("points must not be loaded when the request cannot be served")
				}
				return []PointItem{{Item: "DUCK", Points: 5, IsUser: getMembers}}, nil
			}
			loadGuildName := func(guildID string) (string, error) {
				if test.expectedStatus == http.StatusNotFound {
					t.Fatal("guild name must not be loaded for private or invalid guilds")
				}
				if test.guildNameError != nil {
					return "", test.guildNameError
				}
				return "Windows Admins", nil
			}

			guildHandlerWithDependencies(
				test.isPublic,
				loadPoints,
				loadGuildName,
				func(string, string) (bool, error) { return true, nil },
				newLeaderboardResponseCache(),
			).ServeHTTP(response, request)

			if response.Code != test.expectedStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.expectedStatus)
			}

			if test.expectedBody != "" && response.Body.String() != test.expectedBody {
				t.Errorf("body = %q, want %q", response.Body.String(), test.expectedBody)
			}
		})
	}
}

func TestCORSAlwaysVariesByOrigin(t *testing.T) {
	handler := allowDuckPointsCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if actual := response.Header().Get("Vary"); actual != "Origin" {
		t.Fatalf("Vary = %q, want Origin", actual)
	}
}

func TestCachedLeaderboardStillChecksVisibility(t *testing.T) {
	public := true
	loadCount := 0
	handler := guildHandlerWithDependencies(
		func(string) (bool, error) { return public, nil },
		func(string, bool) ([]PointItem, error) {
			loadCount++
			return []PointItem{{Item: "DUCK", Points: 5}}, nil
		},
		func(string) (string, error) { return "Windows Admins", nil },
		func(string, string) (bool, error) { return true, nil },
		newLeaderboardResponseCache(),
	)
	request := httptest.NewRequest(http.MethodGet, "/guild/618712310185197588/things", nil)
	request = mux.SetURLVars(request, map[string]string{
		"guild": "618712310185197588",
		"type":  "things",
	})

	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, request)
	if firstResponse.Code != http.StatusOK || loadCount != 1 {
		t.Fatalf("first request status=%d loadCount=%d", firstResponse.Code, loadCount)
	}

	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, request)
	if secondResponse.Code != http.StatusOK || loadCount != 1 {
		t.Fatalf("cached request status=%d loadCount=%d", secondResponse.Code, loadCount)
	}

	public = false
	privateResponse := httptest.NewRecorder()
	handler.ServeHTTP(privateResponse, request)
	if privateResponse.Code != http.StatusNotFound {
		t.Fatalf("private cached request status=%d, want 404", privateResponse.Code)
	}
}

func TestGuildMemberLeaderboardExcludesDepartedUsers(t *testing.T) {
	handler := guildHandlerWithDependencies(
		func(string) (bool, error) { return true, nil },
		func(string, bool) ([]PointItem, error) {
			return []PointItem{
				{Item: "111111111111111111", Points: 5, IsUser: true},
				{Item: "222222222222222222", Points: 3, IsUser: true},
			}, nil
		},
		func(string) (string, error) { return "Windows Admins", nil },
		func(_ string, userID string) (bool, error) {
			return userID == "111111111111111111", nil
		},
		newLeaderboardResponseCache(),
	)
	request := httptest.NewRequest(http.MethodGet, "/guild/618712310185197588/members", nil)
	request = mux.SetURLVars(request, map[string]string{
		"guild": "618712310185197588",
		"type":  "members",
	})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	expected := `{"guild":{"id":"618712310185197588","name":"Windows Admins"},"items":[{"item":"111111111111111111","points":5,"isUser":true}]}`
	if response.Body.String() != expected {
		t.Fatalf("body = %q, want %q", response.Body.String(), expected)
	}
}

func TestIsDiscordGuildID(t *testing.T) {
	for _, guildID := range []string{"618712310185197588", "12345678901234567", "12345678901234567890"} {
		if !isDiscordGuildID(guildID) {
			t.Errorf("expected %q to be valid", guildID)
		}
	}
	for _, guildID := range []string{"", "123", "not-a-guild", "1234567890123456x"} {
		if isDiscordGuildID(guildID) {
			t.Errorf("expected %q to be invalid", guildID)
		}
	}
}
