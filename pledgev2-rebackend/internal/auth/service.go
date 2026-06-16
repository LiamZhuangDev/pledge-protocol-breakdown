package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type Config struct {
	AdminUsername string
	AdminPassword string
	TokenSecret   string
	TokenTTL      time.Duration
}

type Service struct {
	adminUsername string
	adminPassword string
	tokenSecret   []byte
	tokenTTL      time.Duration
	now           func() time.Time

	mu       sync.RWMutex
	sessions map[string]session
}

type session struct {
	Username  string
	ExpiresAt time.Time
}

type tokenPayload struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
	Nonce     string `json:"nonce"`
}

func NewService(cfg Config) *Service {
	return &Service{
		adminUsername: cfg.AdminUsername,
		adminPassword: cfg.AdminPassword,
		tokenSecret:   []byte(cfg.TokenSecret),
		tokenTTL:      cfg.TokenTTL,
		now:           time.Now,
		sessions:      make(map[string]session),
	}
}

func (s *Service) Login(username string, password string) (string, error) {
	if username != s.adminUsername || password != s.adminPassword {
		return "", ErrInvalidCredentials
	}

	expiresAt := s.now().UTC().Add(s.tokenTTL)
	payload := tokenPayload{
		Username:  username,
		ExpiresAt: expiresAt.Unix(),
		Nonce:     randomNonce(),
	}

	token, err := s.sign(payload)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = session{
		Username:  username,
		ExpiresAt: expiresAt,
	}
	return token, nil
}

func (s *Service) Authenticate(rawToken string) (string, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return "", ErrInvalidToken
	}

	payload, err := s.verify(token)
	if err != nil {
		return "", err
	}

	now := s.now().UTC()
	if now.Unix() >= payload.ExpiresAt {
		return "", ErrInvalidToken
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	active, ok := s.sessions[token]
	if !ok || active.Username != payload.Username || !active.ExpiresAt.After(now) {
		return "", ErrInvalidToken
	}

	return payload.Username, nil
}

func (s *Service) Logout(rawToken string) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *Service) sign(payload tokenPayload) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := signValue(encodedPayload, s.tokenSecret)
	return encodedPayload + "." + signature, nil
}

func (s *Service) verify(token string) (tokenPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return tokenPayload{}, ErrInvalidToken
	}

	expectedSignature := signValue(parts[0], s.tokenSecret)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[1])) {
		return tokenPayload{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tokenPayload{}, ErrInvalidToken
	}

	payload := tokenPayload{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return tokenPayload{}, ErrInvalidToken
	}

	return payload, nil
}

func signValue(value string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomNonce() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
