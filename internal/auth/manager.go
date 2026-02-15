package auth

import (
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Manager handles authentication and authorization
type Manager struct {
	jwtSecret string
	users     map[string]*User   // userID -> User
	tokens    map[string]*Token  // tokenID -> Token
	apiKeys   map[string]*APIKey // keyID -> APIKey
	passwords map[string]string  // userID -> password hash
	roles     map[string]Role    // roleName -> Role
	tokenTTL  time.Duration
}

// NewManager creates a new auth manager
func NewManager(jwtSecret string) *Manager {
	if jwtSecret == "" {
		// Generate a random JWT secret if not provided
		jwtSecret = generateRandomSecret(32)
		log.Printf("Generated random JWT secret for session (not persistent)")
	}

	m := &Manager{
		jwtSecret: jwtSecret,
		users:     make(map[string]*User),
		tokens:    make(map[string]*Token),
		apiKeys:   make(map[string]*APIKey),
		passwords: make(map[string]string),
		roles:     make(map[string]Role),
		tokenTTL:  24 * time.Hour,
	}

	// Initialize predefined roles
	for roleName, role := range PreDefinedRoles {
		m.roles[roleName] = role
	}

	// Create default admin user (password: admin)
	adminUser := &User{
		ID:        "user-admin",
		Username:  "admin",
		Email:     "admin@loom.local",
		Role:      "admin",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[adminUser.ID] = adminUser

	// Hash and store default password
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	m.passwords[adminUser.ID] = string(passwordHash)

	return m
}

// Login authenticates a user and returns a token
func (m *Manager) Login(username, password string) (*LoginResponse, error) {
	// Find user by username
	var user *User
	for _, u := range m.users {
		if u.Username == username && u.IsActive {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Verify password
	passwordHash, exists := m.passwords[user.ID]
	if !exists {
		return nil, fmt.Errorf("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Generate JWT token
	token, err := m.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:     token,
		ExpiresIn: int64(m.tokenTTL.Seconds()),
		User:      *user,
	}, nil
}

// GenerateToken creates a JWT token for a user
func (m *Manager) GenerateToken(user *User) (string, error) {
	// Get user's permissions from role
	role, exists := m.roles[user.Role]
	if !exists {
		return "", fmt.Errorf("unknown role: %s", user.Role)
	}

	now := time.Now()
	expiresAt := now.Add(m.tokenTTL)

	claims := &Claims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: role.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "loom",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.jwtSecret))
	if err != nil {
		return "", err
	}

	// Store token for revocation
	tokenID := generateRandomID()
	m.tokens[tokenID] = &Token{
		ID:        tokenID,
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check expiration
	if claims.ExpiresAt != nil {
		if time.Now().After(time.Unix(claims.ExpiresAt.Unix(), 0)) {
			return nil, fmt.Errorf("token expired")
		}
	}

	return claims, nil
}

// CreateAPIKey creates a new API key for a user
func (m *Manager) CreateAPIKey(userID string, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Generate API key
	keyID := generateRandomID()
	keyValue := generateRandomSecret(32)
	keyPrefix := keyValue[:8]
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(keyValue), bcrypt.DefaultCost)

	var expiresAt *time.Time
	var expiresAtValue time.Time
	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		expiresAt = &exp
		expiresAtValue = exp
	}

	apiKey := &APIKey{
		ID:          keyID,
		Name:        req.Name,
		UserID:      userID,
		KeyPrefix:   keyPrefix,
		KeyHash:     string(keyHash),
		Permissions: req.Permissions,
		IsActive:    true,
		ExpiresAt:   expiresAtValue,
		CreatedAt:   time.Now(),
	}

	m.apiKeys[keyID] = apiKey

	log.Printf("Created API key %s for user %s", keyPrefix, user.Username)

	return &CreateAPIKeyResponse{
		ID:        keyID,
		Name:      req.Name,
		Key:       keyValue, // Only returned once!
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateAPIKey validates an API key and returns the user and permissions
func (m *Manager) ValidateAPIKey(keyValue string) (string, []string, error) {
	// Find API key by hashing the provided value
	for _, apiKey := range m.apiKeys {
		if !apiKey.IsActive {
			continue
		}

		// Check expiration
		if apiKey.ExpiresAt != *new(time.Time) && time.Now().After(apiKey.ExpiresAt) {
			continue
		}

		// Verify key hash
		if err := bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(keyValue)); err != nil {
			continue
		}

		// Update last used
		apiKey.LastUsed = time.Now()

		return apiKey.UserID, apiKey.Permissions, nil
	}

	return "", nil, fmt.Errorf("invalid API key")
}

// ChangePassword changes a user's password
func (m *Manager) ChangePassword(userID, oldPassword, newPassword string) error {
	user, exists := m.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	passwordHash, exists := m.passwords[userID]
	if !exists {
		return fmt.Errorf("password not set")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("incorrect password")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	m.passwords[userID] = string(newHash)
	user.UpdatedAt = time.Now()

	log.Printf("Password changed for user %s", user.Username)
	return nil
}

// CreateUser creates a new user
func (m *Manager) CreateUser(username, email, role, password string) (*User, error) {
	// Check if username already exists
	for _, u := range m.users {
		if u.Username == username {
			return nil, fmt.Errorf("username already exists")
		}
	}

	// Validate role
	if _, exists := m.roles[role]; !exists {
		return nil, fmt.Errorf("unknown role: %s", role)
	}

	userID := generateRandomID()
	user := &User{
		ID:        userID,
		Username:  username,
		Email:     email,
		Role:      role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Hash password
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	m.passwords[userID] = string(passwordHash)

	m.users[userID] = user

	log.Printf("Created user %s with role %s", username, role)
	return user, nil
}

// GetUser retrieves a user by ID
func (m *Manager) GetUser(userID string) (*User, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// ListUsers lists all users
func (m *Manager) ListUsers() []*User {
	var users []*User
	for _, u := range m.users {
		users = append(users, u)
	}
	return users
}

// HasPermission checks if a user has a permission
func (m *Manager) HasPermission(claims *Claims, permission string) bool {
	for _, p := range claims.Permissions {
		// Check for exact match
		if p == permission {
			return true
		}
		// Check for wildcard
		if p == "*:*" {
			return true
		}
		// Check for resource wildcard (e.g., "agents:*")
		// Extract resource part (before :) and check if permission matches with wildcard
		parts := strings.Split(permission, ":")
		if len(parts) == 2 {
			resourceWildcard := parts[0] + ":*"
			if p == resourceWildcard {
				return true
			}
		}
	}
	return false
}

// generateRandomID generates a random ID
func generateRandomID() string {
	return fmt.Sprintf("id-%s", generateRandomSecret(12))
}

// generateRandomSecret generates a random secret string
func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	// Convert to hex string
	return fmt.Sprintf("%x", bytes)
}
