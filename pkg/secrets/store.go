package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	secretsFileName = ".loom_secrets"
)

// Store manages encrypted secrets
type Store struct {
	secrets map[string]string
	key     []byte
}

// NewStore creates a new secret store
func NewStore() *Store {
	return &Store{
		secrets: make(map[string]string),
		key:     deriveKey(),
	}
}

// Set stores a secret with the given name
func (s *Store) Set(name, value string) error {
	encrypted, err := s.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}
	s.secrets[name] = encrypted
	return nil
}

// Get retrieves a secret by name
func (s *Store) Get(name string) (string, error) {
	encrypted, ok := s.secrets[name]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", name)
	}

	decrypted, err := s.decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}
	return decrypted, nil
}

// Load loads secrets from disk
func (s *Store) Load() error {
	path, err := getSecretsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No secrets file yet
		}
		return err
	}

	return json.Unmarshal(data, &s.secrets)
}

// Save saves secrets to disk
func (s *Store) Save() error {
	path, err := getSecretsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.secrets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// encrypt encrypts a value using AES-GCM
func (s *Store) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a value using AES-GCM
func (s *Store) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// deriveKey derives an encryption key from machine-specific data
func deriveKey() []byte {
	// Use hostname and user info to derive a key
	// This is a simple approach; in production, consider using OS keychain
	hostname, err := os.Hostname()
	if err != nil {
		// Fall back to a constant if hostname cannot be determined
		hostname = "unknown-host"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fall back to current working directory if home cannot be determined
		homeDir, _ = os.Getwd()
		if homeDir == "" {
			homeDir = "unknown-home"
		}
	}

	data := hostname + homeDir
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// getSecretsPath returns the path to the secrets file
func getSecretsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, secretsFileName), nil
}
