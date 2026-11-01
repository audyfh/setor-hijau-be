package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientRecord struct {
	count       int
	windowStart time.Time
}

// RateLimiter implements an in-memory sliding window rate limiter.
type RateLimiter struct {
	mu          sync.Mutex
	records     map[string]*clientRecord
	limit       int
	window      time.Duration
	cleanupTick time.Duration
}

// NewRateLimiter creates a new thread-safe in-memory RateLimiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		records:     make(map[string]*clientRecord),
		limit:       limit,
		window:      window,
		cleanupTick: window * 2,
	}

	// Periodic cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupTick)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, rec := range rl.records {
			if now.Sub(rec.windowStart) > rl.window {
				delete(rl.records, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given key is allowed under the rate limit.
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, exists := rl.records[key]

	if !exists || now.Sub(rec.windowStart) > rl.window {
		rl.records[key] = &clientRecord{
			count:       1,
			windowStart: now,
		}
		return true, 0
	}

	if rec.count >= rl.limit {
		retryAfter := rl.window - now.Sub(rec.windowStart)
		return false, retryAfter
	}

	rec.count++
	return true, 0
}

// RateLimitMiddleware returns a Gin middleware that enforces rate limiting per client IP.
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		allowed, retryAfter := limiter.Allow(ip)
		if !allowed {
			c.Header("Retry-After", time.Duration(retryAfter.Seconds()).String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Terlalu banyak permintaan. Silakan tunggu beberapa saat sebelum mencoba kembali.",
			})
			return
		}

		c.Next()
	}
}
