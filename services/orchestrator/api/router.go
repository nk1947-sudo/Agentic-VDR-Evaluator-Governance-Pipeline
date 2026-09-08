package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func SetupRouter(dealHandler *DealHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "Go Governance Orchestrator",
		})
	})

	mux.HandleFunc("/api/v1/deals/ingest", dealHandler.HandleIngest)

	// Dynamic route dispatcher for /api/v1/deals/{id}/...
	mux.HandleFunc("/api/v1/deals/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")

		if len(parts) >= 4 {
			action := parts[len(parts)-1]
			switch action {
			case "review":
				dealHandler.HandleReview(w, r)
				return
			case "decision":
				dealHandler.HandleDecision(w, r)
				return
			}
		}

		http.NotFound(w, r)
	})

	// Wrap with basic CORS middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
