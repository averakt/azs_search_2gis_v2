package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

type Cache struct {
	cache *cache.Cache
	group singleflight.Group
}

func New(ttl time.Duration) *Cache {
	return &Cache{
		cache: cache.New(ttl, ttl*2),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *Cache) Set(key string, value interface{}) {
	c.cache.Set(key, value, cache.DefaultExpiration)
}

func (c *Cache) GetOrLoad(key string, load func() (interface{}, error)) (interface{}, error) {
	v, err, _ := c.group.Do(key, func() (interface{}, error) {
		if val, found := c.cache.Get(key); found {
			return val, nil
		}
		val, err := load()
		if err != nil {
			return nil, err
		}
		c.cache.Set(key, val, cache.DefaultExpiration)
		return val, nil
	})
	return v, err
}

func (c *Cache) Delete(key string) {
	c.cache.Delete(key)
}
