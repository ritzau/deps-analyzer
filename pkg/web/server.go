package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ritzau/deps-analyzer/pkg/analysis"
)

//go:embed static/*
var staticFiles embed.FS

// AnalysisData holds all the analysis results to send to the frontend
type AnalysisData struct {
	Workspace      string                    `json:"workspace"`
	TotalFiles     int                       `json:"totalFiles"`
	CoveredFiles   int                       `json:"coveredFiles"`
	UncoveredFiles []analysis.UncoveredFile  `json:"uncoveredFiles"`
	CoveragePercent float64                  `json:"coveragePercent"`
}

// Server represents the web server
type Server struct {
	router       *mux.Router
	analysisData *AnalysisData
}

// NewServer creates a new web server
func NewServer() *Server {
	s := &Server{
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// SetAnalysisData updates the analysis data
func (s *Server) SetAnalysisData(data *AnalysisData) {
	s.analysisData = data
}

func (s *Server) setupRoutes() {
	// API routes
	s.router.HandleFunc("/api/analysis", s.handleAnalysis).Methods("GET")

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	s.router.PathPrefix("/").Handler(http.FileServer(http.FS(staticFS)))
}

func (s *Server) handleAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.analysisData == nil {
		http.Error(w, "No analysis data available", http.StatusServiceUnavailable)
		return
	}

	json.NewEncoder(w).Encode(s.analysisData)
}

// Start starts the web server on the specified port
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.router)
}
