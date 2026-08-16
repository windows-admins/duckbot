package main

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

func TestParseLeaderboardDeleteCommand(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantAction string
		wantItem   string
		wantFound  bool
		wantErr    bool
	}{
		{
			name:       "request deletion",
			content:    "<@123> leaderboard delete beans",
			wantAction: "request",
			wantItem:   "BEANS",
			wantFound:  true,
		},
		{
			name:       "confirm deletion",
			content:    "<@123> leaderboard confirm-delete",
			wantAction: "confirm",
			wantFound:  true,
		},
		{
			name:       "cancel deletion",
			content:    "<@123> leaderboard cancel-delete",
			wantAction: "cancel",
			wantFound:  true,
		},
		{
			name:       "delete item named confirm",
			content:    "<@123> leaderboard delete confirm",
			wantAction: "request",
			wantItem:   "CONFIRM",
			wantFound:  true,
		},
		{
			name:       "delete item named dash confirm",
			content:    "<@123> leaderboard delete --confirm",
			wantAction: "request",
			wantItem:   "--CONFIRM",
			wantFound:  true,
		},
		{
			name:      "missing item",
			content:   "<@123> leaderboard delete",
			wantFound: true,
			wantErr:   true,
		},
		{
			name:      "other leaderboard command",
			content:   "<@123> leaderboard status",
			wantFound: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, found, err := parseLeaderboardDeleteCommand(test.content, "123")
			if found != test.wantFound {
				t.Fatalf("found = %t, want %t", found, test.wantFound)
			}
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
			if command.Action != test.wantAction {
				t.Errorf("action = %q, want %q", command.Action, test.wantAction)
			}
			if command.Item != test.wantItem {
				t.Errorf("item = %q, want %q", command.Item, test.wantItem)
			}
		})
	}
}

func TestIsStorageStatusSupportsValueErrors(t *testing.T) {
	err := storage.AzureStorageServiceError{StatusCode: http.StatusNotFound}
	if !isStorageStatus(err, http.StatusNotFound) {
		t.Fatal("expected Azure value error to match status")
	}

}

func TestIsETagConflictSupportsSDKWrappedError(t *testing.T) {
	if !isETagConflict(errors.New("Etag didn't match: storage service returned 412")) {
		t.Fatal("expected SDK-wrapped ETag error to match")
	}

}

func TestRetryStorageThrottling(t *testing.T) {
	attempts := 0
	err := retryStorageThrottling(func() error {
		attempts++
		if attempts < 3 {
			return storage.AzureStorageServiceError{StatusCode: http.StatusTooManyRequests}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retry returned an error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestNormalizeLeaderboardItem(t *testing.T) {
	item, err := normalizeLeaderboardItem(" beans ")
	if err != nil {
		t.Fatalf("normalize returned an error: %v", err)
	}
	if item != "BEANS" {
		t.Fatalf("item = %q, want BEANS", item)
	}

	if _, err := normalizeLeaderboardItem("bad/item"); err == nil {
		t.Fatal("expected unsupported item to fail validation")
	}
}

func TestPendingLeaderboardDeletionExpiry(t *testing.T) {
	now := time.Now().UTC()
	pending := pendingLeaderboardDeletion{
		Item:      "BEANS",
		ETag:      "W/\"datetime'2026-08-15T00%3A00%3A00Z'\"",
		ExpiresAt: now.Add(leaderboardDeleteTimeout),
	}
	if !now.Before(pending.ExpiresAt) {
		t.Fatal("pending deletion should not be expired")
	}
	if !errors.Is(errNoPendingDeletion, errNoPendingDeletion) {
		t.Fatal("pending deletion sentinel must support errors.Is")
	}
}
