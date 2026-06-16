package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"pledgev2-rebackend/internal/config"
	"pledgev2-rebackend/internal/store"
)

type healthResponse struct {
	Status string `json:"status"`
	App    string `json:"app"`
}

type listResponse[T any] struct {
	Data []indexedItem[T] `json:"data"`
}

type indexedItem[T any] struct {
	Index int `json:"index"`
	Data  T   `json:"pool_data"`
}

type tokenListResponse struct {
	Data []store.TokenInfo `json:"data"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(cfg config.Config, logger *slog.Logger, repo store.Repository) *http.Server {
	mux := http.NewServeMux()
	apiPrefix := "/api/v" + strings.TrimPrefix(cfg.APIVersion, "v")

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status: "ok",
			App:    "pledgev2-rebackend",
		})
	})

	mux.HandleFunc("GET "+apiPrefix+"/poolBaseInfo", func(w http.ResponseWriter, r *http.Request) {
		chainID, ok := requireChainID(w, r)
		if !ok {
			return
		}

		pools, err := repo.ListPoolBases(r.Context(), chainID)
		if err != nil {
			logger.Error("list pool base info failed", slog.Any("error", err))
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list pool base info failed"})
			return
		}

		items := make([]indexedItem[store.PoolBase], 0, len(pools))
		for _, pool := range pools {
			items = append(items, indexedItem[store.PoolBase]{
				Index: int(pool.Key.PoolID - 1),
				Data:  pool,
			})
		}
		writeJSON(w, http.StatusOK, listResponse[store.PoolBase]{Data: items})
	})

	mux.HandleFunc("GET "+apiPrefix+"/poolDataInfo", func(w http.ResponseWriter, r *http.Request) {
		chainID, ok := requireChainID(w, r)
		if !ok {
			return
		}

		pools, err := repo.ListPoolData(r.Context(), chainID)
		if err != nil {
			logger.Error("list pool data info failed", slog.Any("error", err))
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list pool data info failed"})
			return
		}

		items := make([]indexedItem[store.PoolData], 0, len(pools))
		for _, pool := range pools {
			items = append(items, indexedItem[store.PoolData]{
				Index: int(pool.Key.PoolID - 1),
				Data:  pool,
			})
		}
		writeJSON(w, http.StatusOK, listResponse[store.PoolData]{Data: items})
	})

	mux.HandleFunc("GET "+apiPrefix+"/token", func(w http.ResponseWriter, r *http.Request) {
		chainID, ok := requireChainID(w, r)
		if !ok {
			return
		}

		tokens, err := repo.ListTokens(r.Context(), chainID)
		if err != nil {
			logger.Error("list tokens failed", slog.Any("error", err))
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list tokens failed"})
			return
		}
		writeJSON(w, http.StatusOK, tokenListResponse{Data: tokens})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "route not found"})
	})

	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: requestLogger(logger, mux),
	}
}

func requireChainID(w http.ResponseWriter, r *http.Request) (string, bool) {
	chainID := r.URL.Query().Get("chainId")
	if chainID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "chainId is required"})
		return "", false
	}
	if _, err := strconv.ParseInt(chainID, 10, 64); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "chainId must be a number"})
		return "", false
	}
	return chainID, true
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("http request", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
