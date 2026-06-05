package middleware

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*attemptInfo
	limit    int
	window   time.Duration
	stop     chan struct{}
	stopOnce sync.Once
}

type attemptInfo struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string]*attemptInfo),
		limit:    limit,
		window:   window,
		stop:     make(chan struct{}),
	}

	// Cleanup goroutine. Exits on Stop() so RateLimiters created during server
	// lifetime don't leak on graceful shutdown.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanup()
			case <-rl.stop:
				return
			}
		}
	}()

	return rl
}

// Stop terminates the cleanup goroutine. Safe to call multiple times.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stop)
	})
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.attempts[key]

	if !exists || now.After(info.resetAt) {
		rl.attempts[key] = &attemptInfo{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true
	}

	if info.count >= rl.limit {
		return false
	}

	info.count++
	return true
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, info := range rl.attempts {
		if now.After(info.resetAt) {
			delete(rl.attempts, key)
		}
	}
}
