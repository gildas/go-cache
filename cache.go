package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gildas/go-core"
	"github.com/gildas/go-errors"
	"github.com/google/uuid"
)

// Cache is a cache
type Cache[T core.Identifiable] struct {
	Name       string
	Items      sync.Map
	Expiration time.Duration
	persistent bool
	folder     string
}

type CacheOption int

const (
	// CacheOptionNone is the default option
	CacheOptionNone CacheOption = iota
	// CacheOptionPersistent tells the cache to persist the data
	CacheOptionPersistent
)

type record[T core.Identifiable] struct {
	Item       T
	Expiration uint64
}

// New creates a new Cache
func New[T core.Identifiable](name string, option ...CacheOption) *Cache[T] {
	cache := &Cache[T]{Name: name}
	for _, opt := range option {
		switch opt {
		case CacheOptionPersistent:
			cache.persistent = true
			cache.folder, _ = os.UserCacheDir()
			cache.folder = filepath.Join(cache.folder, cache.Name)
		}
	}
	return cache
}

// WithExpiration sets the expiration time for the cache
func (cache *Cache[T]) WithExpiration(expiration time.Duration) *Cache[T] {
	cache.Expiration = expiration
	return cache
}

// Set sets an item in the cache
func (cache *Cache[T]) Set(item T) (err error) {
	return cache.SetWithExpiration(item, cache.Expiration)
}

// SetWithExpiration sets an item in the cache with a custom expiration
func (cache *Cache[T]) SetWithExpiration(item T, expiration time.Duration) (err error) {
	var r record[T]

	if expiration == 0 {
		r = record[T]{Item: item} // The Record does not expire
	} else {
		r = record[T]{Item: item, Expiration: uint64(time.Now().Add(expiration).UnixNano())}
	}
	cache.Items.Store(item.GetID(), r)
	if cache.persistent {
		var data []byte

		if data, err = json.Marshal(r); err == nil {
			if err = os.MkdirAll(cache.folder, 0700); err == nil {
				filename := filepath.Join(cache.folder, item.GetID().String())
				err = os.WriteFile(filename, data, 0600)
			}
		}
	}
	return
}

// Get gets an item from the cache
func (cache *Cache[T]) Get(id uuid.UUID) (*T, error) {
	item, found := cache.Items.Load(id)
	if !found {
		if cache.persistent {
			filename := filepath.Join(cache.folder, id.String())
			if data, err := os.ReadFile(filename); err == nil {
				var record record[T]

				if err = json.Unmarshal(data, &record); err == nil {
					cache.Items.Store(id, record)
					return &record.Item, nil
				}
			}
		}
		return nil, errors.NotFound.With("id", id)
	}
	record := item.(record[T])
	if record.Expiration > 0 && time.Now().UnixNano() > int64(record.Expiration) {
		cache.Items.Delete(id)
		return nil, errors.NotFound.With("id", id)
	}
	return &record.Item, nil
}

// Clear clears the cache
func (cache *Cache[T]) Clear() error {
	cache.Items.Range(func(key, value interface{}) bool {
		cache.Items.Delete(key)
		return true
	})
	if cache.persistent {
		return os.RemoveAll(cache.folder)
	}
	return nil
}