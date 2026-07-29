package cluster

import (
	"sync"
	"time"
)

type replayGuard struct {
	mu       sync.Mutex
	entries  map[string]time.Time
	capacity int
	ttl      time.Duration
}

func newReplayGuard(capacity int, ttl time.Duration) *replayGuard {
	return &replayGuard{
		entries:  make(map[string]time.Time),
		capacity: capacity,
		ttl:      ttl,
	}
}

func (g *replayGuard) Accept(controllerID, nonce string, now time.Time) error {
	key := controllerID + ":" + nonce
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pruneLocked(now)
	if _, exists := g.entries[key]; exists {
		return ErrReplay
	}
	if len(g.entries) >= g.capacity {
		return ErrRateLimited
	}
	g.entries[key] = now.Add(g.ttl)
	return nil
}

func (g *replayGuard) pruneLocked(now time.Time) {
	for key, expiresAt := range g.entries {
		if !expiresAt.After(now) {
			delete(g.entries, key)
		}
	}
}

type fixedWindowLimiter struct {
	mu          sync.Mutex
	entries     map[string]windowCount
	limit       int
	window      time.Duration
	maxSubjects int
}

type windowCount struct {
	start time.Time
	count int
}

func newFixedWindowLimiter(limit int, window time.Duration, maxSubjects int) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		entries:     make(map[string]windowCount),
		limit:       limit,
		window:      window,
		maxSubjects: maxSubjects,
	}
}

func (l *fixedWindowLimiter) Allow(subject string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	current, ok := l.entries[subject]
	if !ok || now.Sub(current.start) >= l.window {
		if !ok && len(l.entries) >= l.maxSubjects {
			l.pruneLocked(now)
			if len(l.entries) >= l.maxSubjects {
				return false
			}
		}
		l.entries[subject] = windowCount{start: now, count: 1}
		return true
	}
	if current.count >= l.limit {
		return false
	}
	current.count++
	l.entries[subject] = current
	return true
}

func (l *fixedWindowLimiter) pruneLocked(now time.Time) {
	for key, current := range l.entries {
		if now.Sub(current.start) >= l.window {
			delete(l.entries, key)
		}
	}
}
