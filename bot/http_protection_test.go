package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLeaderboardResponseCache(t *testing.T) {
	cache := newLeaderboardResponseCache()
	now := time.Now()
	cache.set("guild:things", []byte("response"), now)

	body, found := cache.get("guild:things", now.Add(time.Second))
	if !found || string(body) != "response" {
		t.Fatal("expected cached leaderboard response")
	}
	cache.invalidateGuild("guild")
	if _, found := cache.get("guild:things", now.Add(time.Second)); found {
		t.Fatal("expected guild cache to be invalidated")
	}

	cache.entries["expired:things"] = cachedLeaderboardResponse{
		body:      []byte("expired"),
		expiresAt: now.Add(-time.Second),
	}
	cache.lastCleanup = time.Time{}
	cache.set("fresh:things", []byte("fresh"), now)
	if _, found := cache.entries["expired:things"]; found {
		t.Fatal("expired cache entry was not swept")
	}
}

func TestIPRateLimiter(t *testing.T) {
	limiter := newIPRateLimiter()
	now := time.Now()
	for request := 0; request < apiRateLimit; request++ {
		if !limiter.allow("192.0.2.1", now) {
			t.Fatalf("request %d was rejected before the limit", request+1)
		}
	}
	if limiter.allow("192.0.2.1", now) {
		t.Fatal("request above the limit was allowed")
	}
	if !limiter.allow("192.0.2.1", now.Add(apiRateWindow)) {
		t.Fatal("request was not allowed after the rate window reset")
	}
}

func TestRateLimitRequests(t *testing.T) {
	limiter := newIPRateLimiter()
	handler := rateLimitRequests(limiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.1:1234"

	for count := 0; count < apiRateLimit; count++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("request %d returned %d", count+1, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited request returned %d", response.Code)
	}
}

func TestRequestClientIPUsesOriginalForwardedAddress(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.4:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.1:4321")
	if actual := requestClientIP(request); actual != "198.51.100.1" {
		t.Fatalf("client IP = %q", actual)
	}
}

func TestRequestClientIPRejectsForwardingChains(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.4:1234"
	request.Header.Set("X-Forwarded-For", "rotating-value, 198.51.100.1")
	if actual := requestClientIP(request); actual != "" {
		t.Fatalf("client IP = %q, want rejection", actual)
	}
}

func TestNormalizeForwardedIPRejectsInvalidValues(t *testing.T) {
	if actual := normalizeForwardedIP("rotating-value"); actual != "" {
		t.Fatalf("invalid forwarded value normalized to %q", actual)
	}

}

func TestRateLimitRequestsRejectsInvalidForwarding(t *testing.T) {
	handler := rateLimitRequests(newIPRateLimiter(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.4:1234"
	request.Header.Set("X-Forwarded-For", "rotating-value, 198.51.100.1")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid forwarding returned %d, want 400", response.Code)
	}

}

func TestRateLimitRequestsRejectsMissingForwardingFromTrustedProxy(t *testing.T) {
	handler := rateLimitRequests(newIPRateLimiter(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.4:1234"
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing forwarding returned %d, want 400", response.Code)
	}
}
