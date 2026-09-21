// Agente_GCP — Go Backend
// API REST + WebSocket para Chrome Extension
// Conexión: BigQuery, Google Workspace, ML (BQML / Gemma)
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Jerewergez/Agente_Workspace/internal/auth"
	"github.com/Jerewergez/Agente_Workspace/internal/bq"
	mw "github.com/Jerewergez/Agente_Workspace/internal/middleware"
	"github.com/Jerewergez/Agente_Workspace/internal/ml"
	"github.com/Jerewergez/Agente_Workspace/internal/nl"
	"github.com/Jerewergez/Agente_Workspace/internal/query"
	"github.com/Jerewergez/Agente_Workspace/internal/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// --- Initialise WebSocket Hub ---
	hub := ws.NewHub()
	go hub.Run()

	// --- Initialise BigQuery Client (if SA key available) ---
	var bqClient *bq.Client
	saKeyPath := os.Getenv("GCP_SA_KEY_PATH")
	if saKeyPath != "" {
		var err error
		bqClient, err = bq.NewClient(context.Background())
		if err != nil {
			log.Printf("main: bigquery client init skipped: %v", err)
		} else {
			defer bqClient.Close()
			log.Printf("main: bigquery client connected")
		}
	} else {
		log.Println("main: GCP_SA_KEY_PATH not set, BQ client not initialised")
	}

	// --- Initialise Schema Cache (backed by BQ) ---
	var schemaCache *nl.SchemaCache
	if bqClient != nil {
		schemaCache = nl.NewSchemaCache(bqClient, []string{
			os.Getenv("BQ_DATASET_SILVER"),
			os.Getenv("BQ_DATASET_GOLD"),
		})
		if err := schemaCache.Start(context.Background()); err != nil {
			log.Printf("main: schema cache init error: %v", err)
		} else {
			log.Printf("main: schema cache loaded")
		}
	}

	// --- Initialise Gemma Client ---
	gemmaClient := ml.NewGemmaClient()
	if gemmaClient.IsConfigured() {
		log.Printf("main: gemma client configured (GEMMA_URL=%s)", os.Getenv("GEMMA_URL"))
	} else {
		log.Println("main: GEMMA_URL not set, gemma fallback disabled")
	}

	// --- Initialise NL→SQL Query Service ---
	var queryService *query.Service
	if bqClient != nil && schemaCache != nil {
		queryService = query.NewService(bqClient, schemaCache, gemmaClient, query.ServiceConfig{
			MaxRows: 500,
		})
		hub.QueryHandler = ws.QueryRunnerFunc(func(ctx context.Context, rawQuery string) (interface{}, error) {
			return queryService.Query(ctx, rawQuery)
		})
		log.Printf("main: query service ready")
	} else {
		log.Println("main: query service not available (missing BQ client or schema cache)")
	}

	// --- Initialise Query History Store ---
	historyStore := nl.NewHistoryStore(bqClient, nl.HistoryConfig{
		MaxEntriesPerUser: 100,
		BQTable:           os.Getenv("BQ_HISTORY_TABLE"),
		FlushInterval:     30 * time.Second,
	})
	if bqClient != nil {
		log.Printf("main: query history store active")
	}

	// --- Middleware ---
	corsMW := mw.CORS(mw.DefaultCORSOptions())
	rlMW := mw.RateLimitMiddleware

	// --- Routes ---
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)

	// Auth info — requires auth middleware
	mux.Handle("/auth/me", rlMW(corsMW(auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth.WriteUserInfoJSON(w, r)
	})))))

	// WebSocket endpoint
	mux.Handle("/ws", rlMW(corsMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleConnection(w, r)
	}))))

	// --- Query API endpoints ---

	// POST /api/query — execute an NL query
	// Body: {"query": "show me FCR for agent John"}
	mux.Handle("/api/query", rlMW(corsMW(auth.Protected(func(w http.ResponseWriter, r *http.Request, u *auth.UserInfo) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		if queryService == nil {
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unavailable",
				"error":  "query service not configured",
			})
			return
		}

		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Query == "" {
			http.Error(w, `{"error":"query field is required"}`, http.StatusBadRequest)
			return
		}

		start := time.Now()
		result, err := queryService.Query(r.Context(), req.Query)
		latencyMs := time.Since(start).Milliseconds()

		// Record to history (async, non-blocking)
		if historyStore != nil {
			success := err == nil && result != nil && result.Error == ""
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			} else if result != nil && result.Error != "" {
				errMsg = result.Error
				success = false
			}
			source := "template"
			if result != nil {
				source = result.Source
			}

			historyStore.Record(nl.NewHistoryEntry(
				u.Email,
				req.Query,
				resultSQL(result),
				source,
				success,
				errMsg,
				latencyMs,
			))
		}

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "error",
				"error":      err.Error(),
				"latency_ms": latencyMs,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"result": result,
		})
	}))))

	// GET /api/query/history?user_id=X&offset=0&limit=50 — query history
	mux.Handle("/api/query/history", rlMW(corsMW(auth.Protected(func(w http.ResponseWriter, r *http.Request, u *auth.UserInfo) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		if historyStore == nil {
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unavailable",
				"error":  "history store not configured",
			})
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = u.Email
		}
		offsetStr := r.URL.Query().Get("offset")
		limitStr := r.URL.Query().Get("limit")

		offset := 0
		limit := 50
		if offsetStr != "" {
			if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
				offset = v
			}
		}
		if limitStr != "" {
			if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 200 {
				limit = v
			}
		}

		entries := historyStore.GetHistory(userID, offset, limit)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"entries": entries,
			"total":   len(entries),
			"offset":  offset,
			"limit":   limit,
			"user_id": userID,
		})
	}))))

	// Existing API stubs
	mux.Handle("/api/sheets", rlMW(corsMW(auth.Protected(func(w http.ResponseWriter, r *http.Request, u *auth.UserInfo) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_implemented",
			"user":   u.Email,
		})
	}))))

	mux.Handle("/api/gmail", rlMW(corsMW(auth.Protected(func(w http.ResponseWriter, r *http.Request, u *auth.UserInfo) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_implemented",
			"user":   u.Email,
		})
	}))))

	mux.Handle("/api/ml/predict", rlMW(corsMW(auth.Protected(func(w http.ResponseWriter, r *http.Request, u *auth.UserInfo) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_implemented",
			"user":   u.Email,
		})
	}))))

	// Wrap the entire mux with CORS for preflight handling on all routes
	handler := corsMW(mux)

	// --- Server ---
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second, // longer for query execution
		IdleTimeout:  60 * time.Second,
	}

	// --- Graceful Shutdown ---
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("main: shutting down…")

		if schemaCache != nil {
			schemaCache.Stop()
		}
		if historyStore != nil {
			historyStore.Stop()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("main: forced shutdown: %v", err)
		}
	}()

	log.Printf("Agente_GCP backend starting on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

// --- Handlers ---

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// resultSQL extracts the SQL string from a QueryResult.
func resultSQL(result *query.QueryResult) string {
	if result == nil {
		return ""
	}
	return result.SQL
}
