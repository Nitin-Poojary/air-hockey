package handlers

import (
	"sync"

	"golang.org/x/time/rate"
)

// KeyLimiter implements a thread-safe map of rate limiters keyed by an arbitrary string (IP, player ID, etc.).
type KeyLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

// NewKeyLimiter creates a new KeyLimiter with the specified rate and burst parameters.
func NewKeyLimiter(r rate.Limit, b int) *KeyLimiter {
	return &KeyLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

// GetLimiter retrieves the rate limiter for a specific key, creating one if it does not exist yet.
func (kl *KeyLimiter) GetLimiter(key string) *rate.Limiter {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	limiter, exists := kl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(kl.r, kl.b)
		kl.limiters[key] = limiter
	}
	return limiter
}
