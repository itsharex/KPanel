package browsercore

import "sync"

type LimiterSnapshot struct {
	ActiveGlobal    int
	ActiveBySession int
}

type Limiter struct {
	mu            sync.Mutex
	maxGlobal     int
	maxPerSession int
	activeGlobal  int
	active        map[string]int
}

func NewLimiter(maxGlobal, maxPerSession int) *Limiter {
	if maxGlobal < 1 {
		maxGlobal = 1
	}
	if maxPerSession < 1 || maxPerSession > maxGlobal {
		maxPerSession = maxGlobal
	}
	return &Limiter{
		maxGlobal:     maxGlobal,
		maxPerSession: maxPerSession,
		active:        make(map[string]int),
	}
}

func (l *Limiter) Acquire(sessionID string) (func(), bool) {
	l.mu.Lock()
	if sessionID == "" || l.activeGlobal >= l.maxGlobal || l.active[sessionID] >= l.maxPerSession {
		l.mu.Unlock()
		return nil, false
	}
	l.activeGlobal++
	l.active[sessionID]++
	l.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			l.activeGlobal--
			l.active[sessionID]--
			if l.active[sessionID] == 0 {
				delete(l.active, sessionID)
			}
			l.mu.Unlock()
		})
	}, true
}

func (l *Limiter) Snapshot(sessionID string) LimiterSnapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return LimiterSnapshot{ActiveGlobal: l.activeGlobal, ActiveBySession: l.active[sessionID]}
}
