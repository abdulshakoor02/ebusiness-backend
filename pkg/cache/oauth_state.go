package cache

import (
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthState binds a one-time login state to the tenant that initiated it,
// so the public Facebook callback can resolve which tenant to store the token for.
type OAuthState struct {
	State     string
	TenantID  primitive.ObjectID
	CreatedAt time.Time
	ExpiresAt time.Time
}

type OAuthStateCache struct {
	states map[string]*OAuthState
	mu     sync.RWMutex
	ttl    time.Duration
}

func NewOAuthStateCache(ttl time.Duration) *OAuthStateCache {
	c := &OAuthStateCache{
		states: make(map[string]*OAuthState),
		ttl:    ttl,
	}
	go c.startCleanup(1 * time.Minute)
	return c
}

func (c *OAuthStateCache) Set(state string, tenantID primitive.ObjectID) {
	now := time.Now()
	c.mu.Lock()
	c.states[state] = &OAuthState{
		State:     state,
		TenantID:  tenantID,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *OAuthStateCache) Get(state string) (primitive.ObjectID, bool) {
	c.mu.RLock()
	entry, ok := c.states[state]
	c.mu.RUnlock()

	if !ok {
		return primitive.ObjectID{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		c.Delete(state)
		return primitive.ObjectID{}, false
	}
	return entry.TenantID, true
}

func (c *OAuthStateCache) Delete(state string) {
	c.mu.Lock()
	delete(c.states, state)
	c.mu.Unlock()
}

func (c *OAuthStateCache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for state, entry := range c.states {
			if now.After(entry.ExpiresAt) {
				delete(c.states, state)
			}
		}
		c.mu.Unlock()
	}
}
