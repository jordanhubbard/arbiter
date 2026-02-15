package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("Expected non-nil store")
	}
	if store.secrets == nil {
		t.Error("Expected secrets map to be initialized")
	}
	if store.key == nil {
		t.Error("Expected key to be initialized")
	}
	if len(store.key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(store.key))
	}
}

func TestStore_SetAndGet(t *testing.T) {
	store := NewStore()

	// Test setting a secret
	err := store.Set("test-key", "test-value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Test getting the secret
	value, err := store.Get("test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "test-value" {
		t.Errorf("Get() = %q, want %q", value, "test-value")
	}
}

func TestStore_GetNonExistent(t *testing.T) {
	store := NewStore()

	_, err := store.Get("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent secret")
	}
}

func TestStore_SetMultipleSecrets(t *testing.T) {
	store := NewStore()

	secrets := map[string]string{
		"api-key":      "sk-1234567890",
		"db-password":  "super-secret-password",
		"oauth-secret": "oauth-token-xyz",
	}

	// Set all secrets
	for name, value := range secrets {
		if err := store.Set(name, value); err != nil {
			t.Fatalf("Set(%q) error = %v", name, err)
		}
	}

	// Get and verify all secrets
	for name, expected := range secrets {
		value, err := store.Get(name)
		if err != nil {
			t.Fatalf("Get(%q) error = %v", name, err)
		}
		if value != expected {
			t.Errorf("Get(%q) = %q, want %q", name, value, expected)
		}
	}
}

func TestStore_SetEmptyValue(t *testing.T) {
	store := NewStore()

	err := store.Set("empty-key", "")
	if err != nil {
		t.Fatalf("Set() with empty value error = %v", err)
	}

	value, err := store.Get("empty-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "" {
		t.Errorf("Get() = %q, want empty string", value)
	}
}

func TestStore_SetSpecialCharacters(t *testing.T) {
	store := NewStore()

	specialValues := []string{
		"with\nnewlines\nand\ttabs",
		"unicode: 你好世界 🚀",
		"special chars: !@#$%^&*()_+-=[]{}|;:',.<>?",
		`{"json": "value", "nested": {"key": "value"}}`,
	}

	for i, value := range specialValues {
		key := "special-" + string(rune('a'+i))
		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set(%q) error = %v", key, err)
		}

		retrieved, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get(%q) error = %v", key, err)
		}

		if retrieved != value {
			t.Errorf("Get(%q) = %q, want %q", key, retrieved, value)
		}
	}
}

func TestStore_Encrypt(t *testing.T) {
	store := NewStore()

	plaintext := "secret-value"
	encrypted, err := store.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	if encrypted == plaintext {
		t.Error("Encrypted value should not equal plaintext")
	}

	if encrypted == "" {
		t.Error("Encrypted value should not be empty")
	}
}

func TestStore_EncryptDecrypt(t *testing.T) {
	store := NewStore()

	plaintext := "test-secret-value"

	// Encrypt
	encrypted, err := store.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	// Decrypt
	decrypted, err := store.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypt(encrypt(%q)) = %q", plaintext, decrypted)
	}
}

func TestStore_EncryptionUniqueness(t *testing.T) {
	store := NewStore()

	plaintext := "same-value"

	// Encrypt the same value twice
	encrypted1, err1 := store.encrypt(plaintext)
	encrypted2, err2 := store.encrypt(plaintext)

	if err1 != nil || err2 != nil {
		t.Fatalf("encrypt() errors: %v, %v", err1, err2)
	}

	// Due to random nonce, encrypted values should be different
	if encrypted1 == encrypted2 {
		t.Error("Encrypting the same value twice should produce different ciphertexts (different nonces)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := store.decrypt(encrypted1)
	decrypted2, _ := store.decrypt(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Errorf("Both encrypted values should decrypt to original plaintext")
	}
}

func TestStore_DecryptInvalidData(t *testing.T) {
	store := NewStore()

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"empty string", ""},
		{"invalid base64", "not-valid-base64!!!"},
		{"too short", "YQ=="}, // "a" in base64, too short for nonce
		{"garbage data", "YWJjZGVmZ2hpamtsbW5vcA=="}, // valid base64 but invalid ciphertext
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.decrypt(tt.ciphertext)
			if err == nil {
				t.Errorf("decrypt(%q) expected error, got nil", tt.ciphertext)
			}
		})
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create and populate store
	store1 := NewStore()
	store1.Set("key1", "value1")
	store1.Set("key2", "value2")

	// Save to disk
	err := store1.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	secretsPath := filepath.Join(tmpDir, secretsFileName)
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		t.Fatalf("Secrets file was not created at %s", secretsPath)
	}

	// Create new store and load
	store2 := NewStore()
	err = store2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded secrets
	value1, err1 := store2.Get("key1")
	value2, err2 := store2.Get("key2")

	if err1 != nil || err2 != nil {
		t.Fatalf("Get() errors after load: %v, %v", err1, err2)
	}

	if value1 != "value1" {
		t.Errorf("Get(key1) after load = %q, want %q", value1, "value1")
	}
	if value2 != "value2" {
		t.Errorf("Get(key2) after load = %q, want %q", value2, "value2")
	}
}

func TestStore_LoadNonExistentFile(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	store := NewStore()

	// Load should not error on non-existent file
	err := store.Load()
	if err != nil {
		t.Errorf("Load() with non-existent file error = %v, want nil", err)
	}
}

func TestStore_SaveEmptyStore(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	store := NewStore()

	err := store.Save()
	if err != nil {
		t.Fatalf("Save() empty store error = %v", err)
	}

	// Verify file was created
	secretsPath := filepath.Join(tmpDir, secretsFileName)
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		t.Fatalf("Secrets file was not created")
	}
}

func TestStore_OverwriteExistingSecret(t *testing.T) {
	store := NewStore()

	// Set initial value
	store.Set("key", "value1")

	// Overwrite with new value
	store.Set("key", "value2")

	// Get should return latest value
	value, err := store.Get("key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "value2" {
		t.Errorf("Get() after overwrite = %q, want %q", value, "value2")
	}
}

func TestDeriveKey(t *testing.T) {
	key := deriveKey()

	if key == nil {
		t.Fatal("deriveKey() returned nil")
	}

	if len(key) != 32 {
		t.Errorf("deriveKey() length = %d, want 32 (SHA-256)", len(key))
	}

	// Derive key again - should be deterministic
	key2 := deriveKey()

	if len(key) != len(key2) {
		t.Error("deriveKey() should return same length consistently")
	}

	// Keys should be the same for same machine
	equal := true
	for i := range key {
		if key[i] != key2[i] {
			equal = false
			break
		}
	}

	if !equal {
		t.Error("deriveKey() should be deterministic for same machine")
	}
}

func TestGetSecretsPath(t *testing.T) {
	path, err := getSecretsPath()
	if err != nil {
		t.Fatalf("getSecretsPath() error = %v", err)
	}

	if path == "" {
		t.Error("getSecretsPath() returned empty path")
	}

	// Should end with secrets file name
	if filepath.Base(path) != secretsFileName {
		t.Errorf("getSecretsPath() basename = %q, want %q", filepath.Base(path), secretsFileName)
	}

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("getSecretsPath() = %q, expected absolute path", path)
	}
}

func TestStore_LongValue(t *testing.T) {
	store := NewStore()

	// Test with a very long value
	longValue := string(make([]byte, 10000))
	for i := range longValue {
		longValue = longValue[:i] + "a"
	}

	err := store.Set("long-key", longValue)
	if err != nil {
		t.Fatalf("Set() with long value error = %v", err)
	}

	retrieved, err := store.Get("long-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved) != len(longValue) {
		t.Errorf("Get() length = %d, want %d", len(retrieved), len(longValue))
	}
}

func TestStore_DifferentStoresUseSameKey(t *testing.T) {
	// Two stores on the same machine should derive the same key
	store1 := NewStore()
	store2 := NewStore()

	// Set in store1
	store1.Set("key", "value")
	encrypted, _ := store1.encrypt("test")

	// Decrypt in store2 should work
	decrypted, err := store2.decrypt(encrypted)
	if err != nil {
		t.Errorf("Different stores should use same derived key, decrypt error = %v", err)
	}

	if decrypted != "test" {
		t.Errorf("decrypt() across stores = %q, want %q", decrypted, "test")
	}
}
