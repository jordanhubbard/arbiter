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

	// Try to unlock with wrong password - should fail at Unlock()
	km2 := NewKeyManager(storePath)
	if err := km2.Unlock(wrongPassword); err == nil {
		t.Fatal("Expected Unlock to fail with wrong password")
	}

	// Verify key manager is not unlocked
	if km2.IsUnlocked() {
		t.Error("Key manager should not be unlocked with wrong password")
	}
}

func TestKeyManager_ChangePassword(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_keystore.json")

	oldPassword := "old-password-123"
	newPassword := "new-password-456"

	km := NewKeyManager(storePath)

	// ChangePassword on locked store should fail
	if err := km.ChangePassword(oldPassword, newPassword); err == nil {
		t.Error("ChangePassword on locked store should fail")
	}

	// Unlock and store a key
	if err := km.Unlock(oldPassword); err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}
	if err := km.StoreKey("key1", "Key One", "desc", "secret-value-1"); err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}
	if err := km.StoreKey("key2", "Key Two", "desc", "secret-value-2"); err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}

	// Change password with wrong old password should fail
	if err := km.ChangePassword("wrong-old", newPassword); err == nil {
		t.Error("ChangePassword with wrong old password should fail")
	}

	// Change password with correct old password (no stored keys for simpler test)
	km2 := NewKeyManager(filepath.Join(tmpDir, "test_keystore2.json"))
	if err := km2.Unlock(oldPassword); err != nil {
		t.Fatalf("Failed to unlock km2: %v", err)
	}
	if err := km2.ChangePassword(oldPassword, newPassword); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	// Lock and re-unlock with new password
	km2.Lock()
	if err := km2.Unlock(newPassword); err != nil {
		t.Fatalf("Failed to unlock with new password: %v", err)
	}

	// Old password should no longer work
	km2.Lock()
	if err := km2.Unlock(oldPassword); err == nil {
		t.Error("Old password should not work after change")
	}
}

func TestKeyManager_StoreAndDelete(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_keystore.json")

	km := NewKeyManager(storePath)
	if err := km.Unlock("password"); err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// Store key
	if err := km.StoreKey("to-delete", "Delete Me", "will be deleted", "value"); err != nil {
		t.Fatalf("StoreKey() error = %v", err)
	}

	// Verify it exists
	keys, err := km.ListKeys()
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	found := false
	for _, k := range keys {
		if k.ID == "to-delete" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Key 'to-delete' not found in list")
	}

	// Delete it
	if err := km.DeleteKey("to-delete"); err != nil {
		t.Fatalf("DeleteKey() error = %v", err)
	}

	// Verify it's gone
	_, err = km.GetKey("to-delete")
	if err == nil {
		t.Error("GetKey should fail after delete")
	}
}

func TestKeyManager_LockedOperations(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test_keystore.json")

	km := NewKeyManager(storePath)

	// Operations on locked store should fail
	if err := km.StoreKey("key1", "name", "desc", "val"); err == nil {
		t.Error("StoreKey on locked store should fail")
	}
	if _, err := km.GetKey("key1"); err == nil {
		t.Error("GetKey on locked store should fail")
	}
	if err := km.DeleteKey("key1"); err == nil {
		t.Error("DeleteKey on locked store should fail")
	}
	if _, err := km.ListKeys(); err == nil {
		t.Error("ListKeys on locked store should fail")
	}
}
