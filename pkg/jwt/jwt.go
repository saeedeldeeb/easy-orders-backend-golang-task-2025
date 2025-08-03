package jwt

import (
	"errors"
	"time"

	"easy-orders-backend/internal/config"
	"easy-orders-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID string          `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT token operations
type TokenManager struct {
	secretKey []byte
	issuer    string
	expiry    time.Duration
}

// NewTokenManager creates a new token manager
func NewTokenManager(cfg *config.JWTConfig) *TokenManager {
	return &TokenManager{
		secretKey: []byte(cfg.Secret),
		issuer:    "easy-orders-backend",
		expiry:    cfg.ExpireTime,
	}
}

// GenerateToken generates a new JWT token for a user
func (tm *TokenManager) GenerateToken(user *models.User) (string, error) {
	if user == nil {
		return "", errors.New("user cannot be nil")
	}

	// Create claims
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tm.expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key
	tokenString, err := token.SignedString(tm.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is required")
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return tm.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Additional validation
	if claims.UserID == "" {
		return nil, errors.New("invalid user ID in token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

// RefreshToken generates a new token with extended expiry
func (tm *TokenManager) RefreshToken(claims *Claims) (string, error) {
	if claims == nil {
		return "", errors.New("claims cannot be nil")
	}

	// Create new claims with extended expiry
	newClaims := &Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   claims.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tm.expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create and sign new token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := token.SignedString(tm.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetTokenClaims extracts claims from a token without validation (for debugging)
func (tm *TokenManager) GetTokenClaims(tokenString string) (*Claims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

// IsTokenExpired checks if a token is expired without full validation
func (tm *TokenManager) IsTokenExpired(tokenString string) bool {
	claims, err := tm.GetTokenClaims(tokenString)
	if err != nil {
		return true
	}

	if claims.ExpiresAt == nil {
		return true
	}

	return claims.ExpiresAt.Time.Before(time.Now())
}
