package server

import (
	"encoding/json"
	"net/http"

	"github.com/HarshD0011/AMOS/AMOS/pkg/state"
	"k8s.io/klog/v2"
)

type Server struct {
	stateManager *state.StateManager
	frontendPath string
	port         string
}

func NewServer(sm *state.StateManager, frontendPath, port string) *Server {
	if port == "" {
		port = "8080"
	}
	return &Server{
		stateManager: sm,
		frontendPath: frontendPath,
		port:         port,
	}
}

func (s *Server) Run() {
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/issues", s.handleGetIssues)

	// Static Files (Frontend)
	fileServer := http.FileServer(http.Dir(s.frontendPath))
	mux.Handle("/", http.StripPrefix("/", fileServer))

	klog.Infof("Starting Web UI server on port %s", s.port)
	if err := http.ListenAndServe(":"+s.port, s.enableCORS(mux)); err != nil {
		klog.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) handleGetIssues(w http.ResponseWriter, r *http.Request) {
	issues := s.stateManager.GetAllIssues()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(issues); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow any origin for development convenience
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
