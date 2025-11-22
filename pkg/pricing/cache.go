package pricing

import (
	"sync"
	"time"

	"github.com/opscart/k8s-cost-optimizer/pkg/models"
)

// PriceCache caches pricing data to reduce API calls
type PriceCache struct {
	data  map[string]*cacheEntry
	ttl   time.Duration
	mutex sync.RWMutex
}

type cacheEntry struct {
	costInfo  *models.CostInfo
	expiresAt time.Time
}

func NewPriceCache(ttl time.Duration) *PriceCache {
	return &PriceCache{
		data: make(map[string]*cacheEntry),
		ttl:  ttl,
	}
}

func (c *PriceCache) Get(key string) *models.CostInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		// Expired
		delete(c.data, key)
		return nil
	}

	return entry.costInfo
}

func (c *PriceCache) Set(key string, costInfo *models.CostInfo) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = &cacheEntry{
		costInfo:  costInfo,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *PriceCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*cacheEntry)
}
