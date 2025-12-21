package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CachedResponse struct {
	Hash      string          `json:"hash"`
	Response  json.RawMessage `json:"response"`
	CreatedAt time.Time       `json:"created_at"`
}

type Cache struct {
	cacheDir string
	ttl      time.Duration
}

func NewCache(ttl time.Duration) (*Cache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error obtaining home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".matecommit", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating cache directory: %w", err)
	}

	cache := &Cache{
		cacheDir: cacheDir,
		ttl:      ttl,
	}

	_ = cache.CleanExpired()

	return cache, nil
}

// GenerateHash generates a SHA256 hash of the content
func (c *Cache) GenerateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Get gets a response from the cache
func (c *Cache) Get(hash string) (json.RawMessage, bool, error) {
	filePath := filepath.Join(c.cacheDir, hash+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("error reading cache: %w", err)
	}

	var cached CachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false, fmt.Errorf("error unmarshaling cache: %w", err)
	}

	if time.Since(cached.CreatedAt) > c.ttl {
		_ = os.Remove(filePath)
		return nil, false, nil
	}

	return cached.Response, true, nil
}

// Set saves a response to the cache
func (c *Cache) Set(hash string, response interface{}) error {
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error marshaling response: %w", err)
	}

	cached := CachedResponse{
		Hash:      hash,
		Response:  responseData,
		CreatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cache: %w", err)
	}

	filePath := filepath.Join(c.cacheDir, hash+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error saving cache: %w", err)
	}

	return nil
}

// CleanExpired removes expired cache files
func (c *Cache) CleanExpired() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("error reading cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(c.cacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > c.ttl {
			_ = os.Remove(filePath)
		}
	}

	return nil
}

// Clean removes the entire cache
func (c *Cache) Clean() error {
	return os.RemoveAll(c.cacheDir)
}
