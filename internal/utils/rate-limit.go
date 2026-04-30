package utils

import (
	"sync"
	"time"
)

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

type visitor struct {
	lastSeen time.Time
	tokens   int
}

func IsAllowed(ip string) bool {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		visitors[ip] = &visitor{lastSeen: time.Now(), tokens: 10}
		return true
	}

	refill := int(time.Since(v.lastSeen).Minutes())
	if refill > 0 {
		v.tokens += refill
		if v.tokens > 10 {
			v.tokens = 10
		}
		v.lastSeen = time.Now()
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}