package mocks

import (
	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/jwt"

	"github.com/stretchr/testify/mock"
)

// MockTokenManager is a mock implementation of jwt.TokenManager
type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) GenerateToken(user *models.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) ValidateToken(tokenString string) (*jwt.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.Claims), args.Error(1)
}

func (m *MockTokenManager) RefreshToken(claims *jwt.Claims) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) GetTokenClaims(tokenString string) (*jwt.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.Claims), args.Error(1)
}

func (m *MockTokenManager) IsTokenExpired(tokenString string) bool {
	args := m.Called(tokenString)
	return args.Bool(0)
}
