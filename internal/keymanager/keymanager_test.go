package keymanager

import (
	"path/filepath"
	"testing"
)

func TestKeyManager(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_keystore.json")

	km := NewKeyManager(storePath)

	// Test unlock
	password := "test-password-123"
	if err := km.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock key manager: %v", err)
	}

	if !km.IsUnlocked() {
		t.Fatal("Key manager should be unlocked")
	}

	// Test storing a key
	keyID := "test-key-1"
	keyName := "Test Key"
	keyDesc := "A test key"
	keyValue := "secret-api-key-12345"

	if err := km.StoreKey(keyID, keyName, keyDesc, keyValue); err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}

	// Test retrieving the key
	retrievedKey, err := km.GetKey(keyID)
	if err != nil {
		t.Fatalf("Failed to retrieve key: %v", err)
	}

	if retrievedKey != keyValue {
		t.Errorf("Retrieved key mismatch: got %s, want %s", retrievedKey, keyValue)
	}

	// Test listing keys
	keys, err := km.ListKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}

	if keys[0].ID != keyID {
		t.Errorf("Key ID mismatch: got %s, want %s", keys[0].ID, keyID)
	}

	// Test deleting a key
	if err := km.DeleteKey(keyID); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Verify key is deleted
	_, err = km.GetKey(keyID)
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}

	// Test locking
	km.Lock()
	if km.IsUnlocked() {
		t.Error("Key manager should be locked")
	}

	// Verify operations fail when locked
	if err := km.StoreKey("test", "test", "test", "test"); err == nil {
		t.Error("Expected error when storing key in locked state")
	}
}

func TestKeyManagerPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_keystore.json")

	password := "test-password-123"
	keyID := "persistent-key"
	keyValue := "persistent-value"

	// Create and store a key
	km1 := NewKeyManager(storePath)
	if err := km1.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock key manager: %v", err)
	}

	if err := km1.StoreKey(keyID, "Test", "Test", keyValue); err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}

	km1.Lock()

	// Create a new key manager and verify the key persisted
	km2 := NewKeyManager(storePath)
	if err := km2.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock key manager: %v", err)
	}

	retrievedKey, err := km2.GetKey(keyID)
	if err != nil {
		t.Fatalf("Failed to retrieve key: %v", err)
	}

	if retrievedKey != keyValue {
		t.Errorf("Retrieved key mismatch: got %s, want %s", retrievedKey, keyValue)
	}
}

func TestKeyManagerWrongPassword(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_keystore.json")

	password := "correct-password"
	wrongPassword := "wrong-password"
	keyID := "test-key"
	keyValue := "test-value"

	// Create and store a key with correct password
	km1 := NewKeyManager(storePath)
	if err := km1.Unlock(password); err != nil {
		t.Fatalf("Failed to unlock key manager: %v", err)
	}

	if err := km1.StoreKey(keyID, "Test", "Test", keyValue); err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}

	km1.Lock()

	// Try to unlock with wrong password
	km2 := NewKeyManager(storePath)
	if err := km2.Unlock(wrongPassword); err != nil {
		t.Fatalf("Failed to unlock (unlock always succeeds): %v", err)
	}

	// Try to retrieve key - should fail due to wrong password
	_, err := km2.GetKey(keyID)
	if err == nil {
		t.Error("Expected error when decrypting with wrong password")
	}
}
