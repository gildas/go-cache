package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gildas/go-core"
	"github.com/gildas/go-errors"
	"github.com/google/uuid"
)

// Cache is a cache
type Cache[T interface{}] struct {
	Name          string
	Items         sync.Map
	Expiration    time.Duration
	persistent    bool
	folder        string
	encryptionKey []byte
}

type CacheOption int

const (
	// CacheOptionNone is the default option
	CacheOptionNone CacheOption = iota
	// CacheOptionPersistent tells the cache to persist the data
	CacheOptionPersistent
)

type record[T interface{}] struct {
	Item       T
	Expiration uint64
}

// New creates a new Cache
func New[T any](name string, option ...CacheOption) *Cache[T] {
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

// WithEncryptionKey sets the encryption key for the cache
func (cache *Cache[T]) WithEncryptionKey(key []byte) *Cache[T] {
	cache.encryptionKey = key
	cache.persistent = true
	cache.folder, _ = os.UserCacheDir()
	cache.folder = filepath.Join(cache.folder, cache.Name)
	return cache
}

// Set sets an item in the cache
func (cache *Cache[T]) Set(item T, key ...string) (err error) {
	return cache.SetWithExpiration(item, cache.Expiration, key...)
}

// SetWithExpiration sets an item in the cache with a custom expiration
func (cache *Cache[T]) SetWithExpiration(item T, expiration time.Duration, key ...string) (err error) {
	var r record[T]

	if identifiable, ok := any(item).(core.Identifiable); ok {
		key = append(key, identifiable.GetID().String())
	}
	if identifiable, ok := any(item).(core.StringIdentifiable); ok {
		key = append(key, identifiable.GetID())
	}
	if named, ok := any(item).(core.Named); ok {
		key = append(key, named.GetName())
	}
	if len(key)	== 0 {
		return errors.ArgumentMissing.With("key")
	}

	if expiration == 0 {
		r = record[T]{Item: item} // The Record does not expire
	} else {
		r = record[T]{Item: item, Expiration: uint64(time.Now().Add(expiration).UnixNano())}
	}
	for _, k := range key {
		cache.Items.Store(k, r)
		if cache.persistent {
			var data []byte

			if data, err = json.Marshal(r); err == nil {
				if err = os.MkdirAll(cache.folder, 0700); err == nil {
					k = uuid.NewSHA1(uuid.Nil, []byte(k)).String()
					filename := filepath.Join(cache.folder, k)
					if len(cache.encryptionKey) > 0 {
						if data, err = cache.encrypt(data); err != nil {
							return
						}
					}
					err = os.WriteFile(filename, data, 0600)
				}
			}
		}
	}
	return
}

// Get gets an item from the cache
func (cache *Cache[T]) Get(key string) (*T, error) {
	item, found := cache.Items.Load(key)
	if !found {
		if cache.persistent {
			filekey := uuid.NewSHA1(uuid.Nil, []byte(key)).String()
			filename := filepath.Join(cache.folder, filekey)
			if data, err := os.ReadFile(filename); err == nil {
				var record record[T]

				if len(cache.encryptionKey) > 0 {
					if data, err = cache.decrypt(data); err != nil {
						return nil, err
					}
				}
				if err = json.Unmarshal(data, &record); err == nil {
					cache.Items.Store(key, record)
					return &record.Item, nil
				}
			}
		}
		return nil, errors.NotFound.With("key", key)
	}
	record := item.(record[T])
	if record.Expiration > 0 && time.Now().UnixNano() > int64(record.Expiration) {
		cache.Items.Delete(key)
		if cache.persistent {
			filename := filepath.Join(cache.folder, key)
			_ = os.Remove(filename)
		}
		return nil, errors.NotFound.With("key", key)
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

// encrypt encrypts data using AES
func (cache *Cache[T]) encrypt(data []byte) (encrypted []byte, err error) {
	var block cipher.Block

	if block, err = aes.NewCipher(cache.encryptionKey); err == nil {
		var gcm cipher.AEAD

		if gcm, err = cipher.NewGCM(block); err == nil {
			nonce := make([]byte, gcm.NonceSize())
			if _, err = io.ReadFull(rand.Reader, nonce); err == nil {
				return gcm.Seal(nonce, nonce, data, nil), nil
			}
		}
	}
	return nil, err
}

// decrypt decrypts data using AES
func (cache *Cache[T]) decrypt(data []byte) (decrypted []byte, err error) {
	var block cipher.Block

	if block, err = aes.NewCipher(cache.encryptionKey); err == nil {
		var gcm cipher.AEAD

		if gcm, err = cipher.NewGCM(block); err == nil {
			nonceSize := gcm.NonceSize()
			if len(data) >= nonceSize {
				nonce, ciphertext := data[:nonceSize], data[nonceSize:]
				return gcm.Open(nil, nonce, ciphertext, nil)
			}
			return nil, errors.New("ciphertext too short")
		}
	}
	return nil, err
}
