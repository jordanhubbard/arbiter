package auth

import (
	"strings"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager("test-secret")

	if m == nil {
		t.Fatal("Expected non-nil manager")
	}

	if m.jwtSecret != "test-secret" {
		t.Errorf("Expected jwtSecret 'test-secret', got %q", m.jwtSecret)
	}

	if len(m.users) == 0 {
		t.Error("Expected default admin user to be created")
	}

	if len(m.roles) == 0 {
		t.Error("Expected predefined roles to be loaded")
	}

	// Check default admin user
	adminUser := m.users["user-admin"]
	if adminUser == nil {
		t.Fatal("Expected default admin user with ID 'user-admin'")
	}

	if adminUser.Username != "admin" {
		t.Errorf("Expected admin username, got %q", adminUser.Username)
	}

	if adminUser.Role != "admin" {
		t.Errorf("Expected admin role, got %q", adminUser.Role)
	}
}

func TestNewManagerEmptySecret(t *testing.T) {
	m := NewManager("")

	if m.jwtSecret == "" {
		t.Error("Expected random secret to be generated")
	}

	if len(m.jwtSecret) == 0 {
		t.Error("Expected non-empty JWT secret")
	}
}

func TestManager_Login_Success(t *testing.T) {
	m := NewManager("test-secret")

	resp, err := m.Login("admin", "admin")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.Token == "" {
		t.Error("Expected token to be returned")
	}

	if resp.User.Username != "admin" {
		t.Errorf("Expected user admin, got %q", resp.User.Username)
	}

	if resp.ExpiresIn == 0 {
		t.Error("Expected non-zero expiration time")
	}
}

func TestManager_Login_InvalidPassword(t *testing.T) {
	m := NewManager("test-secret")

	_, err := m.Login("admin", "wrong-password")
	if err == nil {
		t.Error("Expected error for invalid password")
	}
}

func TestManager_Login_NonExistentUser(t *testing.T) {
	m := NewManager("test-secret")

	_, err := m.Login("nonexistent", "password")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestManager_GenerateToken(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	token, err := m.GenerateToken(adminUser)

	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Token should be stored
	found := false
	for _, stored := range m.tokens {
		if stored.Token == token {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected token to be stored")
	}
}

func TestManager_GenerateTokenInvalidRole(t *testing.T) {
	m := NewManager("test-secret")

	user := &User{
		ID:       "test-user",
		Username: "testuser",
		Role:     "nonexistent-role",
	}

	_, err := m.GenerateToken(user)
	if err == nil {
		t.Error("Expected error for unknown role")
	}
}

func TestManager_ValidateToken_Success(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	token, err := m.GenerateToken(adminUser)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != adminUser.ID {
		t.Errorf("Expected UserID %q, got %q", adminUser.ID, claims.UserID)
	}

	if claims.Username != adminUser.Username {
		t.Errorf("Expected Username %q, got %q", adminUser.Username, claims.Username)
	}

	if claims.Role != adminUser.Role {
		t.Errorf("Expected Role %q, got %q", adminUser.Role, claims.Role)
	}
}

func TestManager_ValidateTokenInvalid(t *testing.T) {
	m := NewManager("test-secret")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"invalid format", "invalid-token"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.ValidateToken(tt.token)
			if err == nil {
				t.Errorf("ValidateToken(%q) expected error, got nil", tt.token)
			}
		})
	}
}

func TestManager_CreateAPIKey(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	req := CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"agents:read", "beads:write"},
		ExpiresIn:   3600, // 1 hour
	}

	resp, err := m.CreateAPIKey(adminUser.ID, req)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	if resp.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if resp.Name != "test-key" {
		t.Errorf("Expected name 'test-key', got %q", resp.Name)
	}

	if resp.Key == "" {
		t.Error("Expected non-empty key")
	}

	if resp.ExpiresAt == nil {
		t.Error("Expected expiration time")
	}

	// Verify API key was stored
	found := false
	for _, apiKey := range m.apiKeys {
		if apiKey.ID == resp.ID {
			found = true
			if apiKey.Name != req.Name {
				t.Errorf("Stored name %q, want %q", apiKey.Name, req.Name)
			}
			break
		}
	}

	if !found {
		t.Error("API key was not stored")
	}
}

func TestManager_CreateAPIKeyNonExistentUser(t *testing.T) {
	m := NewManager("test-secret")

	req := CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"agents:read"},
	}

	_, err := m.CreateAPIKey("nonexistent-user", req)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestManager_CreateAPIKeyNoExpiration(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	req := CreateAPIKeyRequest{
		Name:        "no-expiry-key",
		Permissions: []string{"agents:read"},
		ExpiresIn:   0, // No expiration
	}

	resp, err := m.CreateAPIKey(adminUser.ID, req)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	// Bug fixed - should not panic now
	if resp.ExpiresAt != nil {
		t.Error("Expected nil expiration for key with ExpiresIn=0")
	}

	// Verify API key was stored with zero time
	apiKey := m.apiKeys[resp.ID]
	if !apiKey.ExpiresAt.IsZero() {
		t.Error("Expected zero time for ExpiresAt when no expiration")
	}
}

func TestManager_ValidateAPIKey_Success(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	req := CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"agents:read", "beads:write"},
		ExpiresIn:   3600, // 1 hour - avoid bug with ExpiresIn=0
	}

	resp, err := m.CreateAPIKey(adminUser.ID, req)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	// Validate the API key
	userID, permissions, err := m.ValidateAPIKey(resp.Key)
	if err != nil {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}

	if userID != adminUser.ID {
		t.Errorf("Expected userID %q, got %q", adminUser.ID, userID)
	}

	if len(permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(permissions))
	}
}

func TestManager_ValidateAPIKeyInvalid(t *testing.T) {
	m := NewManager("test-secret")

	_, _, err := m.ValidateAPIKey("invalid-key")
	if err == nil {
		t.Error("Expected error for invalid API key")
	}
}

func TestManager_ChangePassword_Success(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]

	err := m.ChangePassword(adminUser.ID, "admin", "new-password")
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	// Try logging in with new password
	_, err = m.Login("admin", "new-password")
	if err != nil {
		t.Errorf("Login with new password failed: %v", err)
	}

	// Old password should not work
	_, err = m.Login("admin", "admin")
	if err == nil {
		t.Error("Old password should not work after change")
	}
}

func TestManager_ChangePasswordWrongOldPassword(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]

	err := m.ChangePassword(adminUser.ID, "wrong-password", "new-password")
	if err == nil {
		t.Error("Expected error for wrong old password")
	}
}

func TestManager_ChangePasswordNonExistentUser(t *testing.T) {
	m := NewManager("test-secret")

	err := m.ChangePassword("nonexistent", "old", "new")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestManager_CreateUser_Success(t *testing.T) {
	m := NewManager("test-secret")

	user, err := m.CreateUser("testuser", "test@example.com", "user", "password123")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %q", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %q", user.Email)
	}

	if user.Role != "user" {
		t.Errorf("Expected role 'user', got %q", user.Role)
	}

	if !user.IsActive {
		t.Error("Expected new user to be active")
	}

	// Verify user was stored
	if _, exists := m.users[user.ID]; !exists {
		t.Error("User was not stored")
	}

	// Verify password was stored
	if _, exists := m.passwords[user.ID]; !exists {
		t.Error("Password was not stored")
	}

	// Try logging in
	_, err = m.Login("testuser", "password123")
	if err != nil {
		t.Errorf("Login with new user failed: %v", err)
	}
}

func TestManager_CreateUserDuplicateUsername(t *testing.T) {
	m := NewManager("test-secret")

	_, err := m.CreateUser("admin", "admin2@example.com", "user", "password")
	if err == nil {
		t.Error("Expected error for duplicate username")
	}
}

func TestManager_CreateUserInvalidRole(t *testing.T) {
	m := NewManager("test-secret")

	_, err := m.CreateUser("testuser", "test@example.com", "invalid-role", "password")
	if err == nil {
		t.Error("Expected error for invalid role")
	}
}

func TestManager_GetUser_Success(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	user, err := m.GetUser(adminUser.ID)

	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}

	if user.ID != adminUser.ID {
		t.Errorf("Expected ID %q, got %q", adminUser.ID, user.ID)
	}
}

func TestManager_GetUserNotFound(t *testing.T) {
	m := NewManager("test-secret")

	_, err := m.GetUser("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestManager_ListUsers(t *testing.T) {
	m := NewManager("test-secret")

	// Create additional users
	m.CreateUser("user1", "user1@example.com", "user", "password")
	m.CreateUser("user2", "user2@example.com", "viewer", "password")

	users := m.ListUsers()

	if len(users) < 3 {
		t.Errorf("Expected at least 3 users, got %d", len(users))
	}

	// Verify admin user is in the list
	foundAdmin := false
	for _, u := range users {
		if u.Username == "admin" {
			foundAdmin = true
			break
		}
	}

	if !foundAdmin {
		t.Error("Expected admin user in list")
	}
}

func TestManager_HasPermission(t *testing.T) {
	m := NewManager("test-secret")

	tests := []struct {
		name       string
		role       string
		permission string
		want       bool
	}{
		{"admin has all", "admin", "agents:read", true},
		{"admin wildcard", "admin", "anything:anything", true},
		{"user has agents:read", "user", "agents:read", true},
		{"user has beads:write", "user", "beads:write", true},
		{"viewer has agents:read", "viewer", "agents:read", true},
		{"viewer cannot write", "viewer", "agents:write", false},
		{"viewer cannot delete", "viewer", "agents:delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, exists := m.roles[tt.role]
			if !exists {
				t.Fatalf("Role %q not found", tt.role)
			}

			claims := &Claims{
				UserID:      "test-user",
				Username:    "testuser",
				Role:        tt.role,
				Permissions: role.Permissions,
			}

			got := m.HasPermission(claims, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.permission, got, tt.want)
			}
		})
	}
}

func TestGenerateRandomID(t *testing.T) {
	id := generateRandomID()

	if id == "" {
		t.Error("Expected non-empty ID")
	}

	if !strings.HasPrefix(id, "id-") {
		t.Errorf("Expected ID to start with 'id-', got %q", id)
	}

	// Generate another ID - should be different
	id2 := generateRandomID()
	if id == id2 {
		t.Error("Expected different random IDs")
	}
}

func TestGenerateRandomSecret(t *testing.T) {
	secret := generateRandomSecret(16)

	if secret == "" {
		t.Error("Expected non-empty secret")
	}

	if len(secret) != 32 { // hex encoding doubles the length
		t.Errorf("Expected length 32 for 16 bytes hex, got %d", len(secret))
	}

	// Generate another secret - should be different
	secret2 := generateRandomSecret(16)
	if secret == secret2 {
		t.Error("Expected different random secrets")
	}
}

func TestManager_TokenTTL(t *testing.T) {
	m := NewManager("test-secret")

	if m.tokenTTL != 24*time.Hour {
		t.Errorf("Expected default TTL 24h, got %v", m.tokenTTL)
	}
}

func TestManager_PreDefinedRoles(t *testing.T) {
	m := NewManager("test-secret")

	expectedRoles := []string{"admin", "user", "viewer", "service"}

	for _, roleName := range expectedRoles {
		role, exists := m.roles[roleName]
		if !exists {
			t.Errorf("Expected role %q to exist", roleName)
			continue
		}

		if role.Name != roleName {
			t.Errorf("Role name mismatch: expected %q, got %q", roleName, role.Name)
		}

		if role.Description == "" {
			t.Errorf("Role %q should have a description", roleName)
		}
	}

	// Admin should have wildcard permission
	adminRole := m.roles["admin"]
	hasWildcard := false
	for _, perm := range adminRole.Permissions {
		if perm == "*:*" {
			hasWildcard = true
			break
		}
	}

	if !hasWildcard {
		t.Error("Admin role should have *:* permission")
	}
}

// TestManager_ValidateToken_ExpiredToken tests expired token validation
func TestManager_ValidateToken_ExpiredToken(t *testing.T) {
	m := NewManager("test-secret")

	// Set very short token TTL
	m.tokenTTL = 1 * time.Nanosecond

	resp, err := m.Login("admin", "admin")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Wait a bit to ensure token expires
	time.Sleep(10 * time.Millisecond)

	_, err = m.ValidateToken(resp.Token)
	if err == nil {
		t.Error("Expected error for expired token")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("Expected 'expired' error, got: %v", err)
	}

	// Reset TTL for other tests
	m.tokenTTL = 24 * time.Hour
}

// TestManager_CreateAPIKey_NoExpiration tests API key without expiration
func TestManager_CreateAPIKey_NoExpiration(t *testing.T) {
	m := NewManager("test-secret")

	req := CreateAPIKeyRequest{
		Name:        "No Expiry Key",
		Permissions: []string{"read:*"},
		ExpiresIn:   0, // No expiration
	}

	resp, err := m.CreateAPIKey("user-admin", req)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	if resp.ExpiresAt != nil {
		t.Error("Expected nil ExpiresAt for key without expiration")
	}
}

// TestManager_ValidateAPIKey_Expiration tests API key expiration
func TestManager_ValidateAPIKey_Expiration(t *testing.T) {
	m := NewManager("test-secret")

	req := CreateAPIKeyRequest{
		Name:        "Expiring Key",
		Permissions: []string{"read:*"},
		ExpiresIn:   1, // 1 second
	}

	resp, err := m.CreateAPIKey("user-admin", req)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	// Validate immediately - should work
	userID, perms, err := m.ValidateAPIKey(resp.Key)
	if err != nil {
		t.Errorf("ValidateAPIKey() immediate error = %v", err)
	}

	if userID != "user-admin" {
		t.Errorf("UserID = %q, want %q", userID, "user-admin")
	}

	if len(perms) != 1 {
		t.Errorf("Permissions length = %d, want 1", len(perms))
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Validate after expiration - should fail
	_, _, err = m.ValidateAPIKey(resp.Key)
	if err == nil {
		t.Error("Expected error for expired API key")
	}
}

func TestManager_LoginInactiveUser(t *testing.T) {
	m := NewManager("test-secret")

	// Create a user and deactivate
	user, _ := m.CreateUser("inactive", "inactive@example.com", "user", "password")
	user.IsActive = false

	_, err := m.Login("inactive", "password")
	if err == nil {
		t.Error("Expected error for inactive user")
	}
}

func TestManager_RevokeAPIKey(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	req := CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"agents:read"},
		ExpiresIn:   3600,
	}

	resp, _ := m.CreateAPIKey(adminUser.ID, req)

	// Deactivate the key
	apiKey := m.apiKeys[resp.ID]
	apiKey.IsActive = false

	// Should fail validation
	_, _, err := m.ValidateAPIKey(resp.Key)
	if err == nil {
		t.Error("Expected error for inactive API key")
	}
}

func TestManager_APIKeyExpiration(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	req := CreateAPIKeyRequest{
		Name:        "expired-key",
		Permissions: []string{"agents:read"},
		ExpiresIn:   1, // 1 second
	}

	resp, _ := m.CreateAPIKey(adminUser.ID, req)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should fail validation
	_, _, err := m.ValidateAPIKey(resp.Key)
	if err == nil {
		t.Error("Expected error for expired API key")
	}
}

func TestManager_MultipleUsers(t *testing.T) {
	m := NewManager("test-secret")

	// Create multiple users with different roles
	users := []struct {
		username string
		role     string
	}{
		{"user1", "user"},
		{"user2", "viewer"},
		{"admin2", "admin"},
	}

	for _, u := range users {
		_, err := m.CreateUser(u.username, u.username+"@example.com", u.role, "password")
		if err != nil {
			t.Errorf("CreateUser(%q) error = %v", u.username, err)
		}
	}

	// List all users
	allUsers := m.ListUsers()
	if len(allUsers) < 4 { // 3 created + 1 default admin
		t.Errorf("Expected at least 4 users, got %d", len(allUsers))
	}

	// Login with each user
	for _, u := range users {
		_, err := m.Login(u.username, "password")
		if err != nil {
			t.Errorf("Login(%q) error = %v", u.username, err)
		}
	}
}

func TestManager_UpdateUserTimestamp(t *testing.T) {
	m := NewManager("test-secret")

	user, _ := m.CreateUser("testuser", "test@example.com", "user", "password")
	originalTime := user.UpdatedAt

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Change password updates timestamp
	m.ChangePassword(user.ID, "password", "newpassword")

	if !user.UpdatedAt.After(originalTime) {
		t.Error("Expected UpdatedAt to be updated after password change")
	}
}

func TestManager_HasPermissionWithWildcard(t *testing.T) {
	m := NewManager("test-secret")

	// Test with custom permissions including wildcards
	claims := &Claims{
		UserID:      "test",
		Username:    "test",
		Role:        "custom",
		Permissions: []string{"agents:*", "beads:read"},
	}

	tests := []struct {
		permission string
		want       bool
	}{
		{"agents:read", true},    // matches agents:*
		{"agents:write", true},   // matches agents:*
		{"agents:delete", true},  // matches agents:*
		{"beads:read", true},     // exact match
		{"beads:write", false},   // no match
		{"projects:read", false}, // no match
	}

	for _, tt := range tests {
		t.Run(tt.permission, func(t *testing.T) {
			got := m.HasPermission(claims, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission(%q) = %v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestManager_TokenTTLConfiguration(t *testing.T) {
	m := NewManager("test-secret")

	// Default TTL
	if m.tokenTTL != 24*time.Hour {
		t.Errorf("Expected default TTL 24h, got %v", m.tokenTTL)
	}

	// Can be modified
	m.tokenTTL = 1 * time.Hour

	adminUser := m.users["user-admin"]
	token, _ := m.GenerateToken(adminUser)

	claims, _ := m.ValidateToken(token)

	// Token should expire in ~1 hour
	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn > 70*time.Minute || expiresIn < 50*time.Minute {
		t.Errorf("Expected token to expire in ~1h, got %v", expiresIn)
	}
}

func TestManager_ValidateTokenExpiredToken(t *testing.T) {
	m := NewManager("test-secret")
	m.tokenTTL = 1 * time.Second // Very short TTL

	adminUser := m.users["user-admin"]
	token, _ := m.GenerateToken(adminUser)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	_, err := m.ValidateToken(token)
	if err == nil {
		t.Error("Expected error for expired token")
	}
	if err != nil && !strings.Contains(err.Error(), "expired") {
		t.Errorf("Expected 'expired' error, got: %v", err)
	}
}

func TestManager_CreateAPIKeyWithCustomPermissions(t *testing.T) {
	m := NewManager("test-secret")

	adminUser := m.users["user-admin"]
	customPerms := []string{"custom:read", "custom:write", "special:admin"}

	req := CreateAPIKeyRequest{
		Name:        "custom-perms-key",
		Permissions: customPerms,
		ExpiresIn:   3600,
	}

	resp, err := m.CreateAPIKey(adminUser.ID, req)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	// Validate and check permissions
	_, perms, err := m.ValidateAPIKey(resp.Key)
	if err != nil {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}

	if len(perms) != len(customPerms) {
		t.Errorf("Expected %d permissions, got %d", len(customPerms), len(perms))
	}

	for i, perm := range customPerms {
		if perms[i] != perm {
			t.Errorf("Permission[%d] = %q, want %q", i, perms[i], perm)
		}
	}
}
