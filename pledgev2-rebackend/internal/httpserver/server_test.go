package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pledgev2-rebackend/internal/auth"
	"pledgev2-rebackend/internal/chain"
	"pledgev2-rebackend/internal/config"
	"pledgev2-rebackend/internal/store"
)

func TestHealthz(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected healthy response, got %+v", body)
	}
}

func TestPoolBaseInfo(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/poolBaseInfo?chainId=97", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body listResponse[store.PoolBase]
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Data) != 1 {
		t.Fatalf("expected one pool, got %d", len(body.Data))
	}
	if body.Data[0].Data.Key.PoolID != 1 {
		t.Fatalf("expected pool id 1, got %+v", body.Data[0])
	}
}

func TestPoolDataInfoRequiresChainID(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/poolDataInfo", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTokenList(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token?chainId=97", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body tokenListResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Data) != 2 {
		t.Fatalf("expected two tokens, got %d", len(body.Data))
	}
}

func TestLoginAndProtectedSession(t *testing.T) {
	server := newTestServer(t)

	token := loginForTest(t, server)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body sessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Username != "admin" {
		t.Fatalf("expected admin username, got %s", body.Username)
	}
}

func TestProtectedSessionRejectsMissingToken(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	server := newTestServer(t)
	token := loginForTest(t, server)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected logout status %d, got %d", http.StatusOK, rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked token status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func newTestServer(t *testing.T) *http.Server {
	t.Helper()

	repo := store.NewMemoryStore()
	if err := chain.SyncPools(context.Background(), chain.NewDemoReader(), repo, "97"); err != nil {
		t.Fatalf("sync demo contract data: %v", err)
	}

	return New(
		config.Config{Env: "test", Port: "0", APIVersion: "1"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repo,
		auth.NewService(auth.Config{
			AdminUsername: "admin",
			AdminPassword: "password",
			TokenSecret:   "test-secret",
			TokenTTL:      time.Hour,
		}),
	)
}

func loginForTest(t *testing.T, server *http.Server) string {
	t.Helper()

	body := bytes.NewBufferString(`{"name":"admin","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", body)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, rec.Code)
	}

	var response loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if response.TokenID == "" {
		t.Fatal("expected login token")
	}
	return response.TokenID
}
