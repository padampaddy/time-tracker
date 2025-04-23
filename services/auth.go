package services

import (
	"github.com/time-tracker/v2/internal/auth"
	"github.com/time-tracker/v2/internal/config"
)

// AuthService implements auth.Service interface
type AuthService struct {
	apiClient *ApiClient
}

// NewAuthService creates a new instance of AuthService
func NewAuthService() auth.Service {
	// Provide a default BaseURL for the ApiClient
	// TODO: Make this configurable
	return &AuthService{
		apiClient: NewApiClient(config.API_URL),
	}
}

// Login authenticates a user with their email and password
func (s *AuthService) Login(email, password string) (*auth.User, error) {
	if email == "" || password == "" {
		return nil, nil
	}

	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	response, err := s.apiClient.Login(payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to User
	user := &auth.User{
		ID:       int(response["id"].(float64)),
		Username: response["username"].(string),
		Email:    response["email"].(string),
		Role:     response["role"].(string),
		Token:    response["token"].(string),
	}

	return user, nil
}
