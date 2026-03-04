// Package web — Embedded HTTP dashboard for qeloxd.
// Serves a static SPA via embed.FS and exposes REST API for:
//
//	GET  /api/metrics  → snapshot de métricas
//	GET  /api/status   → estado + uptime do node
//	GET  /api/logs     → últimas N linhas de log (?tail=N, default 100)
//	POST /api/command  → start | stop | restart
package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/log"
	"github.com/zeus/qelox/internal/monitor"
	"github.com/zeus/qelox/internal/node"
)

//go:embed static
var staticFiles embed.FS

// Server is the dashboard HTTP server.
type Server struct {
	cfg    *config.Config
	node   *node.Controller
	mon    *monitor.Monitor
	logger *log.Logger
	srv    *http.Server
}

// New cria um Server.
func New(cfg *config.Config, nc *node.Controller, mon *monitor.Monitor, logger *log.Logger) *Server {
	return &Server{cfg: cfg, node: nc, mon: mon, logger: logger}
}

// Start inicia o HTTP server em goroutine.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints.
	mux.HandleFunc("/api/stats", s.handleMetrics)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/config/environment", s.handleEnvironment)
	mux.HandleFunc("/api/health", s.handleHealth)

	// SPA estática via embed.FS — serve index.html para qualquer rota não-API.
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to mount static fs: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Se o arquivo não existe, serve index.html (SPA fallback).
		if r.URL.Path != "/" {
			f, err := staticFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err != nil {
				r.URL.Path = "/"
			} else {
				f.Close()
			}
		}
		fileServer.ServeHTTP(w, r)
	}))

	handler := corsMiddleware(mux)
	if s.cfg.Web.Username != "" && s.cfg.Web.Password != "" {
		handler = s.basicAuthMiddleware(handler)
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Web.Bind, s.cfg.Web.Port)
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to open web port %s: %w", addr, err)
	}

	go func() {
		s.logger.Info("web dashboard started", "addr", "http://"+addr)
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("web server error", "error", err)
		}
	}()

	return nil
}

// Stop encerra o HTTP server com graceful shutdown.
func (s *Server) Stop() {
	if s.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.srv.Shutdown(ctx)
	s.logger.Info("web dashboard terminated")
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.mon.Snapshot())
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]interface{}{
		"state":    s.node.State().String(),
		"uptime":   s.node.Uptime().Round(time.Second).String(),
		"restarts": s.node.Restarts(),
	})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	tailN := 100
	if q := r.URL.Query().Get("tail"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 5000 {
			tailN = n
		}
	}
	lines := s.logger.Tail(tailN)
	writeJSON(w, map[string]interface{}{"lines": lines, "count": len(lines)})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok"})
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, "body inválido", http.StatusBadRequest)
		return
	}

	var err error
	switch strings.ToLower(strings.TrimSpace(body.Command)) {
	case "start":
		err = s.node.Start()
	case "stop":
		err = s.node.Stop()
	case "restart":
		err = s.node.Restart()
	default:
		httpError(w, "unknown command: "+body.Command, http.StatusBadRequest)
		return
	}

	if err != nil {
		writeJSON(w, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (s *Server) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Environment string `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, "body inválido", http.StatusBadRequest)
		return
	}

	env := strings.ToLower(strings.TrimSpace(body.Environment))
	if env == "" {
		httpError(w, "environment cannot be empty", http.StatusBadRequest)
		return
	}

	s.logger.Info("changing environment to", "env", env)

	// 1. Pausa o nó
	s.node.Stop()
	time.Sleep(1 * time.Second) // Dá tempo para o processo morrer com calma

	// 2. Atualiza config em memória
	home, _ := os.UserHomeDir()

	// Ajusta data-dir baseado no environment
	s.cfg.Node.DataDir = fmt.Sprintf("%s/.go-quai-%s", home, env)

	foundEnv := false
	for i, arg := range s.cfg.Node.ExtraArgs {
		if strings.HasPrefix(arg, "--node.environment=") {
			s.cfg.Node.ExtraArgs[i] = "--node.environment=" + env
			foundEnv = true
		}
	}
	if !foundEnv {
		s.cfg.Node.ExtraArgs = append(s.cfg.Node.ExtraArgs, "--node.environment="+env)
	}

	// 3. Salva no disco
	if err := config.Save(s.cfg); err != nil {
		s.logger.Error("failed to save config", "error", err)
		httpError(w, "failed to save", http.StatusInternalServerError)
		return
	}

	// 4. Reinicia o nó com nova config
	if err := s.node.Start(); err != nil {
		s.logger.Error("failed to start new network", "error", err)
		writeJSON(w, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{"ok": true})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// corsMiddleware adiciona headers CORS básicos.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// basicAuthMiddleware exige usuário e senha definidos no config.toml
func (s *Server) basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != s.cfg.Web.Username || pass != s.cfg.Web.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
