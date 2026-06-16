package auth

import (
	"errors"
	"testing"
	"time"
)

func TestLoginAuthenticateAndLogout(t *testing.T) {
	service := newTestService()

	token, err := service.Login("admin", "password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	username, err := service.Authenticate(token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if username != "admin" {
		t.Fatalf("expected admin username, got %s", username)
	}

	service.Logout(token)

	_, err = service.Authenticate(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected token to be invalid after logout, got %v", err)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	service := newTestService()

	_, err := service.Login("admin", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestAuthenticateRejectsExpiredToken(t *testing.T) {
	service := newTestService()
	start := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return start }

	token, err := service.Login("admin", "password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	service.now = func() time.Time { return start.Add(2 * time.Hour) }
	_, err = service.Authenticate(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected expired token, got %v", err)
	}
}

func newTestService() *Service {
	return NewService(Config{
		AdminUsername: "admin",
		AdminPassword: "password",
		TokenSecret:   "test-secret",
		TokenTTL:      time.Hour,
	})
}
