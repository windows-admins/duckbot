package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	leaderboardCacheTTL = 15 * time.Second
	apiRateLimit        = 60
	apiRateWindow       = time.Minute
)

type cachedLeaderboardResponse struct {
	body      []byte
	expiresAt time.Time
}

type leaderboardResponseCache struct {
	mu          sync.RWMutex
	entries     map[string]cachedLeaderboardResponse
	lastCleanup time.Time
}

func newLeaderboardResponseCache() *leaderboardResponseCache {
	return &leaderboardResponseCache{entries: make(map[string]cachedLeaderboardResponse)}
}

func (cache *leaderboardResponseCache) get(key string, now time.Time) ([]byte, bool) {
	cache.mu.RLock()
	entry, found := cache.entries[key]
	cache.mu.RUnlock()
	if !found || !now.Before(entry.expiresAt) {
		if found {
			cache.mu.Lock()
			delete(cache.entries, key)
			cache.mu.Unlock()
		}
		return nil, false
	}
	return append([]byte(nil), entry.body...), true
}

func (cache *leaderboardResponseCache) set(key string, body []byte, now time.Time) {
	cache.mu.Lock()
	if cache.lastCleanup.IsZero() || now.Sub(cache.lastCleanup) >= leaderboardCacheTTL {
		for entryKey, entry := range cache.entries {
			if !now.Before(entry.expiresAt) {
				delete(cache.entries, entryKey)
			}
		}
		cache.lastCleanup = now
	}
	cache.entries[key] = cachedLeaderboardResponse{
		body:      append([]byte(nil), body...),
		expiresAt: now.Add(leaderboardCacheTTL),
	}
	cache.mu.Unlock()
}

func (cache *leaderboardResponseCache) invalidateGuild(guildID string) {
	cache.mu.Lock()
	delete(cache.entries, guildID+":members")
	delete(cache.entries, guildID+":things")
	cache.mu.Unlock()
}

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type ipRateLimiter struct {
	mu          sync.Mutex
	entries     map[string]rateLimitEntry
	lastCleanup time.Time
}

func newIPRateLimiter() *ipRateLimiter {
	return &ipRateLimiter{entries: make(map[string]rateLimitEntry)}
}

func (limiter *ipRateLimiter) allow(ip string, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if limiter.lastCleanup.IsZero() || now.Sub(limiter.lastCleanup) >= apiRateWindow {
		for key, entry := range limiter.entries {
			if now.Sub(entry.windowStart) >= apiRateWindow {
				delete(limiter.entries, key)
			}
		}
		limiter.lastCleanup = now
	}

	entry := limiter.entries[ip]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= apiRateWindow {
		limiter.entries[ip] = rateLimitEntry{count: 1, windowStart: now}
		return true
	}
	if entry.count >= apiRateLimit {
		return false
	}
	entry.count++
	limiter.entries[ip] = entry
	return true
}

func rateLimitRequests(limiter *ipRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := requestClientIP(r)
		if clientIP == "" {
			http.Error(w, "Invalid forwarded address.", http.StatusBadRequest)
			return
		}
		if !limiter.allow(clientIP, time.Now()) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestClientIP(r *http.Request) string {
	remoteIP := normalizeForwardedIP(r.RemoteAddr)
	if isTrustedProxyIP(remoteIP) {
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded == "" {
			return ""
		}
		parts := strings.Split(forwarded, ",")
		if len(parts) != 1 {
			return ""
		}
		candidate := normalizeForwardedIP(strings.TrimSpace(parts[0]))
		if net.ParseIP(candidate) == nil || isTrustedProxyIP(candidate) {
			return ""
		}
		return candidate
	}
	if remoteIP != "" {
		return remoteIP
	}
	return r.RemoteAddr
}

func isTrustedProxyIP(candidate string) bool {
	ip := net.ParseIP(candidate)
	return ip != nil && (isPrivateIP(ip) || ip.IsLoopback() || ip.IsLinkLocalUnicast())
}

func isPrivateIP(ip net.IP) bool {
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4[0] == 10 ||
			(ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31) ||
			(ipv4[0] == 192 && ipv4[1] == 168)
	}
	return len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc
}

func normalizeForwardedIP(candidate string) string {
	if net.ParseIP(candidate) != nil {
		return candidate
	}
	host, _, err := net.SplitHostPort(candidate)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	return ""
}
