package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.DataDir == "" {
		t.Error("Expected non-empty DataDir")
	}

	if cfg.DatabasePath == "" {
		t.Error("Expected non-empty DatabasePath")
	}

	if cfg.KeyStorePath == "" {
		t.Error("Expected non-empty KeyStorePath")
	}

	// DatabasePath should be within DataDir
	if !filepath.IsAbs(cfg.DatabasePath) {
		t.Error("Expected DatabasePath to be absolute")
	}

	// Check file names
	if filepath.Base(cfg.DatabasePath) != "loom.db" {
		t.Errorf("Expected database file 'loom.db', got %q", filepath.Base(cfg.DatabasePath))
	}

	if filepath.Base(cfg.KeyStorePath) != "keystore.json" {
		t.Errorf("Expected keystore file 'keystore.json', got %q", filepath.Base(cfg.KeyStorePath))
	}
}

func TestDefaultWithXDGDataHome(t *testing.T) {
	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create temp directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)

	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	expectedDataDir := filepath.Join(tmpDir, "loom")

	if cfg.DataDir != expectedDataDir {
		t.Errorf("Expected DataDir %q, got %q", expectedDataDir, cfg.DataDir)
	}

	// Verify directory was created
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		t.Errorf("DataDir %q was not created", cfg.DataDir)
	}
}

func TestDefaultCreatesDataDirectory(t *testing.T) {
	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create temp directory
	tmpDir := t.TempDir()
	dataHome := filepath.Join(tmpDir, "data")
	os.Setenv("XDG_DATA_HOME", dataHome)

	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	// Verify both data home and loom directories were created
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		t.Errorf("DataDir %q was not created", cfg.DataDir)
	}

	// Check permissions (0700)
	info, err := os.Stat(cfg.DataDir)
	if err != nil {
		t.Fatalf("Failed to stat DataDir: %v", err)
	}

	expectedPerm := os.FileMode(0700)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("DataDir permissions = %o, want %o", info.Mode().Perm(), expectedPerm)
	}
}

func TestDefaultWithoutXDG(t *testing.T) {
	// Save and clear XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)
	os.Unsetenv("XDG_DATA_HOME")

	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	// Should fall back to ~/.local/share/loom
	if !filepath.IsAbs(cfg.DataDir) {
		t.Error("Expected absolute path for DataDir")
	}

	// DataDir should contain "loom"
	if filepath.Base(cfg.DataDir) != "loom" {
		t.Errorf("Expected DataDir basename 'loom', got %q", filepath.Base(cfg.DataDir))
	}

	// Should contain .local/share/loom
	if !(contains(cfg.DataDir, ".local") && contains(cfg.DataDir, "share") && contains(cfg.DataDir, "loom")) {
		t.Errorf("Expected DataDir to contain .local/share/loom, got %q", cfg.DataDir)
	}
}

func TestGetPasswordFromEnv(t *testing.T) {
	// Save original LOOM_PASSWORD
	originalPassword := os.Getenv("LOOM_PASSWORD")
	defer func() {
		if originalPassword != "" {
			os.Setenv("LOOM_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("LOOM_PASSWORD")
		}
	}()

	// Set test password
	testPassword := "test-password-123"
	os.Setenv("LOOM_PASSWORD", testPassword)

	password, err := GetPassword()
	if err != nil {
		t.Fatalf("GetPassword() error = %v", err)
	}

	if password != testPassword {
		t.Errorf("GetPassword() = %q, want %q", password, testPassword)
	}
}

func TestGetPasswordEmptyEnv(t *testing.T) {
	// Save original LOOM_PASSWORD
	originalPassword := os.Getenv("LOOM_PASSWORD")
	defer func() {
		if originalPassword != "" {
			os.Setenv("LOOM_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("LOOM_PASSWORD")
		}
	}()

	// Unset password
	os.Unsetenv("LOOM_PASSWORD")

	// This test would require mocking stdin, which is complex
	// Skip interactive test for now
	t.Skip("GetPassword without env requires stdin mocking")
}

func TestConfigStructFields(t *testing.T) {
	cfg := &Config{
		DatabasePath: "/path/to/db",
		KeyStorePath: "/path/to/keystore",
		DataDir:      "/path/to/data",
	}

	if cfg.DatabasePath != "/path/to/db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "/path/to/db")
	}

	if cfg.KeyStorePath != "/path/to/keystore" {
		t.Errorf("KeyStorePath = %q, want %q", cfg.KeyStorePath, "/path/to/keystore")
	}

	if cfg.DataDir != "/path/to/data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/path/to/data")
	}
}

func TestDefaultPathStructure(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	// DatabasePath and KeyStorePath should be in DataDir
	dbDir := filepath.Dir(cfg.DatabasePath)
	if dbDir != cfg.DataDir {
		t.Errorf("DatabasePath not in DataDir: %q vs %q", dbDir, cfg.DataDir)
	}

	keyDir := filepath.Dir(cfg.KeyStorePath)
	if keyDir != cfg.DataDir {
		t.Errorf("KeyStorePath not in DataDir: %q vs %q", keyDir, cfg.DataDir)
	}
}

func TestGetPasswordWithEmptyString(t *testing.T) {
	// Save and set empty password env
	originalPassword := os.Getenv("LOOM_PASSWORD")
	defer func() {
		if originalPassword != "" {
			os.Setenv("LOOM_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("LOOM_PASSWORD")
		}
	}()

	os.Setenv("LOOM_PASSWORD", "")

	// With empty env, it will try to prompt (which will fail in test)
	// but we're testing the env check path
	t.Skip("GetPassword with empty env requires stdin mocking")
}

// Helper function
func contains(s, substr string) bool {
	return filepath.ToSlash(s) != "" && filepath.ToSlash(s) != "/" &&
		(filepath.ToSlash(s) == substr ||
		 filepath.Base(s) == substr ||
		 filepath.Dir(s) != s && contains(filepath.Dir(s), substr))
}

// TestConfig_StructInitialization tests Config struct initialization
func TestConfig_StructInitialization(t *testing.T) {
	cfg := &Config{
		DatabasePath: "/path/to/db.db",
		KeyStorePath: "/path/to/keystore.json",
		DataDir:      "/path/to/data",
	}

	if cfg.DatabasePath != "/path/to/db.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "/path/to/db.db")
	}

	if cfg.KeyStorePath != "/path/to/keystore.json" {
		t.Errorf("KeyStorePath = %q, want %q", cfg.KeyStorePath, "/path/to/keystore.json")
	}

	if cfg.DataDir != "/path/to/data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/path/to/data")
	}
}

// TestConfig_ZeroValue tests Config with zero values
func TestConfig_ZeroValue(t *testing.T) {
	var cfg Config

	if cfg.DatabasePath != "" {
		t.Errorf("Zero-value DatabasePath = %q, want empty string", cfg.DatabasePath)
	}

	if cfg.KeyStorePath != "" {
		t.Errorf("Zero-value KeyStorePath = %q, want empty string", cfg.KeyStorePath)
	}

	if cfg.DataDir != "" {
		t.Errorf("Zero-value DataDir = %q, want empty string", cfg.DataDir)
	}
}

// TestDefaultCreatesDirectoryWithCorrectPermissions tests directory creation with specific permissions
func TestDefaultCreatesDirectoryWithCorrectPermissions(t *testing.T) {
	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create temp directory
	tmpDir := t.TempDir()
	customDataHome := filepath.Join(tmpDir, "custom_data")
	os.Setenv("XDG_DATA_HOME", customDataHome)

	cfg, err := Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(cfg.DataDir)
	if err != nil {
		t.Fatalf("Failed to stat DataDir: %v", err)
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Error("DataDir should be a directory")
	}

	// Verify permissions are 0700
	expectedPerm := os.FileMode(0700)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("DataDir permissions = %o, want %o", info.Mode().Perm(), expectedPerm)
	}
}

// TestDefaultConsistency tests that multiple calls to Default() return consistent paths
func TestDefaultConsistency(t *testing.T) {
	// Save original XDG_DATA_HOME
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)

	cfg1, err1 := Default()
	if err1 != nil {
		t.Fatalf("First Default() error = %v", err1)
	}

	cfg2, err2 := Default()
	if err2 != nil {
		t.Fatalf("Second Default() error = %v", err2)
	}

	if cfg1.DataDir != cfg2.DataDir {
		t.Errorf("DataDir not consistent: %q vs %q", cfg1.DataDir, cfg2.DataDir)
	}

	if cfg1.DatabasePath != cfg2.DatabasePath {
		t.Errorf("DatabasePath not consistent: %q vs %q", cfg1.DatabasePath, cfg2.DatabasePath)
	}

	if cfg1.KeyStorePath != cfg2.KeyStorePath {
		t.Errorf("KeyStorePath not consistent: %q vs %q", cfg1.KeyStorePath, cfg2.KeyStorePath)
	}
}

// TestGetPasswordWithValidEnvironmentVariable tests GetPassword with LOOM_PASSWORD set
func TestGetPasswordWithValidEnvironmentVariable(t *testing.T) {
	originalPassword := os.Getenv("LOOM_PASSWORD")
	defer func() {
		if originalPassword != "" {
			os.Setenv("LOOM_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("LOOM_PASSWORD")
		}
	}()

	testPassword := "my-secure-password-123"
	os.Setenv("LOOM_PASSWORD", testPassword)

	password, err := GetPassword()
	if err != nil {
		t.Fatalf("GetPassword() error = %v", err)
	}

	if password != testPassword {
		t.Errorf("GetPassword() = %q, want %q", password, testPassword)
	}
}
