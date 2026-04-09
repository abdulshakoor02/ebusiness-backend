package cache

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type ImportSession struct {
	ID        string
	FileData  []byte
	Extension string
	Headers   []string
	Rows      [][]string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type ImportSessionCache struct {
	sessions map[string]*ImportSession
	mu       sync.RWMutex
	ttl      time.Duration
}

func NewImportSessionCache(ttl time.Duration) *ImportSessionCache {
	c := &ImportSessionCache{
		sessions: make(map[string]*ImportSession),
		ttl:      ttl,
	}
	go c.startCleanup(1 * time.Minute)
	return c
}

func (c *ImportSessionCache) Set(fileData []byte, ext string, headers []string, rows [][]string) *ImportSession {
	now := time.Now()
	session := &ImportSession{
		ID:        uuid.New().String(),
		FileData:  fileData,
		Extension: ext,
		Headers:   headers,
		Rows:      rows,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
	}

	c.mu.Lock()
	c.sessions[session.ID] = session
	c.mu.Unlock()

	return session
}

func (c *ImportSessionCache) Get(id string) (*ImportSession, bool) {
	c.mu.RLock()
	session, ok := c.sessions[id]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		c.Delete(id)
		return nil, false
	}

	return session, true
}

func (c *ImportSessionCache) Delete(id string) {
	c.mu.Lock()
	delete(c.sessions, id)
	c.mu.Unlock()
}

func (c *ImportSessionCache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for id, session := range c.sessions {
			if now.After(session.ExpiresAt) {
				delete(c.sessions, id)
			}
		}
		c.mu.Unlock()
	}
}
