package utils

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/llm-aware-gateway/pkg/interfaces"
)

// cacheItem 缓存项
type cacheItem struct {
	value     interface{}
	expiredAt time.Time
}

// cache LRU缓存实现
type cache struct {
	lru   *lru.Cache[string, *cacheItem]
	mutex sync.RWMutex
}

// NewCache 创建缓存
func NewCache(size int) interfaces.Cache {
	lruCache, _ := lru.New[string, *cacheItem](size)

	return &cache{
		lru: lruCache,
	}
}

// Get 获取缓存值
func (c *cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.lru.Get(key)
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.expiredAt) {
		c.lru.Remove(key)
		return nil, false
	}

	return item.value, true
}

// Set 设置缓存值
func (c *cache) Set(key string, value interface{}, ttl int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var expiredAt time.Time
	if ttl > 0 {
		expiredAt = time.Now().Add(time.Duration(ttl) * time.Second)
	} else {
		expiredAt = time.Time{} // 永不过期
	}

	item := &cacheItem{
		value:     value,
		expiredAt: expiredAt,
	}

	c.lru.Add(key, item)
	return nil
}

// Delete 删除缓存值
func (c *cache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.lru.Remove(key)
	return nil
}

// Clear 清空缓存
func (c *cache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.lru.Purge()
	return nil
}

// Size 获取缓存大小
func (c *cache) Size() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return int64(c.lru.Len())
}

// cleanupExpired 清理过期项
func (c *cache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	keys := c.lru.Keys()

	for _, key := range keys {
		if item, exists := c.lru.Get(key); exists {
			if !item.expiredAt.IsZero() && now.After(item.expiredAt) {
				c.lru.Remove(key)
			}
		}
	}
}

// StartCleanup 启动定期清理
func (c *cache) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.cleanupExpired()
		}
	}()
}
