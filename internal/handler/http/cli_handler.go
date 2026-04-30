package handler

import (
	"sync"
	"time"
)


type CLISession struct {
    CodeChallenge string
    CodeVerifier string
    RedirectURI   string
    CreatedAt     time.Time
}

type CLIStore struct {
    mu       sync.Mutex
    sessions map[string]CLISession
}

func NewCLIStore() *CLIStore {
    return &CLIStore{sessions: make(map[string]CLISession)}
}

func (s *CLIStore) Set(state string, session CLISession) {
    s.mu.Lock()
    defer s.mu.Unlock()
    session.CreatedAt = time.Now()
    s.sessions[state] = session
}

func (s *CLIStore) Get(state string) (CLISession, bool) {
    s.mu.Lock()
    defer s.mu.Unlock()

    sess, ok := s.sessions[state]
    if !ok {
        return CLISession{}, false
    }

    // Expire after 10 minutes
    if time.Since(sess.CreatedAt) > 10*time.Minute {
        delete(s.sessions, state)
        return CLISession{}, false
    }
    return sess, true
}

func (s *CLIStore) Delete(state string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.sessions, state)
}