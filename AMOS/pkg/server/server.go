package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	// SPA Fallback Handler - serves index.html for client-side routes
	mux.HandleFunc("/", s.handleSPA)

	klog.Infof("Starting Web UI server on port %s", s.port)
	if err := http.ListenAndServe(":"+s.port, s.enableCORS(mux)); err != nil {
		klog.Fatalf("Server failed: %v", err)
	}
}

// handleSPA serves static files if they exist, otherwise falls back to index.html
// This enables React Router to handle client-side routes like /pods, /deployments, /jobs
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// Get the path from the URL
	path := r.URL.Path

	// Build the full file path
	fullPath := filepath.Join(s.frontendPath, path)

	// Check if the requested file exists
	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		// File exists, serve it
		http.ServeFile(w, r, fullPath)
		return
	}

	// Check if it's a request for static assets (has a file extension)
	if strings.Contains(filepath.Base(path), ".") {
		// It's a file request that doesn't exist - return 404
		http.NotFound(w, r)
		return
	}

	// For all other routes (like /pods, /deployments, /jobs), serve index.html
	// This allows React Router to handle the routing
	indexPath := filepath.Join(s.frontendPath, "index.html")
	http.ServeFile(w, r, indexPath)
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
