package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestCache(t *testing.T, ttl time.Duration) (*Cache, string) {
	tempDir, err := os.MkdirTemp("", "matecommit-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	c := &Cache{
		cacheDir: tempDir,
		ttl:      ttl,
	}

	return c, tempDir
}

func TestNewCache(t *testing.T) {
	// Act
	c, err := NewCache(1 * time.Hour)

	// Assert
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewCache() returned nil")
	}
	if _, err := os.Stat(c.cacheDir); os.IsNotExist(err) {
		t.Errorf("cache directory %s was not created", c.cacheDir)
	}
}

func TestCache_GenerateHash(t *testing.T) {
	// Arrange
	c := &Cache{}
	content := "test content"

	// Act
	hash1 := c.GenerateHash(content)
	hash2 := c.GenerateHash(content)
	hash3 := c.GenerateHash("different content")

	// Assert
	if hash1 != hash2 {
		t.Errorf("GenerateHash() returned different results for same content")
	}
	if hash1 == hash3 {
		t.Errorf("GenerateHash() returned same result for different content")
	}
	if len(hash1) != 64 {
		t.Errorf("GenerateHash() length = %d, want 64", len(hash1))
	}
}

func TestCache_SetAndGet(t *testing.T) {
	// Arrange
	c, tempDir := setupTestCache(t, 1*time.Hour)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	type testData struct {
		Name string `json:"name"`
	}
	data := testData{Name: "MateCommit"}
	hash := c.GenerateHash("matecommit-key")

	// Act - Set
	err := c.Set(hash, data)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Act - Get
	resp, found, err := c.Get(hash)

	// Assert
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() returned found = false, want true")
	}

	var got testData
	_ = json.Unmarshal(resp, &got)
	if got.Name != data.Name {
		t.Errorf("Get() data = %v, want %v", got.Name, data.Name)
	}
}

func TestCache_Get_NotFound(t *testing.T) {
	// Arrange
	c, tempDir := setupTestCache(t, 1*time.Hour)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	// Act
	_, found, err := c.Get("non-existent-hash")

	// Assert
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if found {
		t.Errorf("Get() found = true, want false")
	}
}

func TestCache_Get_Expired(t *testing.T) {
	// Arrange
	c, tempDir := setupTestCache(t, 10*time.Millisecond)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	hash := "expired-hash"
	_ = c.Set(hash, "some data")

	time.Sleep(20 * time.Millisecond)

	// Act
	_, found, err := c.Get(hash)

	// Assert
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if found {
		t.Errorf("Get() found = true, want false for expired cache")
	}

	filePath := filepath.Join(tempDir, hash+".json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expired cache file was not deleted")
	}
}

func TestCache_CleanExpired(t *testing.T) {
	// Arrange
	c, tempDir := setupTestCache(t, 1*time.Hour)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	_ = c.Set("fresh", "data")

	oldHash := "old"
	_ = c.Set(oldHash, "data")
	oldFilePath := filepath.Join(tempDir, oldHash+".json")
	oldTime := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(oldFilePath, oldTime, oldTime)

	// Act
	err := c.CleanExpired()

	// Assert
	if err != nil {
		t.Errorf("CleanExpired() error = %v", err)
	}

	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Errorf("old file was not cleaned up")
	}

	freshFilePath := filepath.Join(tempDir, "fresh.json")
	if _, err := os.Stat(freshFilePath); os.IsNotExist(err) {
		t.Errorf("fresh file was incorrectly cleaned up")
	}
}

func TestCache_Clean(t *testing.T) {
	// Arrange
	c, tempDir := setupTestCache(t, 1*time.Hour)

	_ = c.Set("hash1", "data")
	_ = c.Set("hash2", "data")

	// Act
	err := c.Clean()

	// Assert
	if err != nil {
		t.Errorf("Clean() error = %v", err)
	}

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("cache directory was not removed by Clean()")
	}
}

func TestCache_Get_UnmarshalError(t *testing.T) {
	// Arrange
	c, tempDir := setupTestCache(t, 1*time.Hour)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	hash := "corrupt-hash"
	filePath := filepath.Join(tempDir, hash+".json")
	_ = os.WriteFile(filePath, []byte("invalid json{"), 0644)

	// Act
	_, found, err := c.Get(hash)

	// Assert
	if err == nil {
		t.Error("Get() error = nil, want error for invalid JSON")
	}
	if found {
		t.Error("Get() found = true, want false for invalid JSON")
	}
}
