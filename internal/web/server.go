// Package web — Embedded HTTP dashboard for qeloxd.
// Serves a static SPA via embed.FS and exposes REST API.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	stdlog "log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/explorer"
	"github.com/zeus/qelox/internal/log"
	"github.com/zeus/qelox/internal/monitor"
	"github.com/zeus/qelox/internal/node"
)

//go:embed static
var staticFiles embed.FS

// Server is the dashboard HTTP server.
type Server struct {
	cfg      *config.Config
	node     *node.Controller
	mon      *monitor.Monitor
	logger   *log.Logger
	explorer *explorer.Explorer
	srv      *http.Server
}

// New cria um Server.
func New(cfg *config.Config, nc *node.Controller, mon *monitor.Monitor, logger *log.Logger) *Server {
	return &Server{cfg: cfg, node: nc, mon: mon, logger: logger, explorer: explorer.New(cfg, nc)}
}

// Start inicia o HTTP server em goroutine.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints.
	mux.HandleFunc("/api/stats", s.handleMetrics)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/config/environment", s.handleEnvironment)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/command", s.handleCommand)
	mux.HandleFunc("/api/config", s.handleSaveConfig)

	// Explorer endpoints
	mux.HandleFunc("/api/explorer/search", s.handleExplorerSearch)
	mux.HandleFunc("/api/explorer/block", s.handleExplorerBlock)
	mux.HandleFunc("/api/explorer/tx", s.handleExplorerTx)
	mux.HandleFunc("/api/explorer/address", s.handleExplorerAddress)

	// SPA estática via embed.FS
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to mount static fs: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// Wrap handler with diagnostics
	handler := requestLogger(http.Handler(mux))
	if s.cfg.Web.Username != "" && s.cfg.Web.Password != "" {
		handler = s.basicAuthMiddleware(handler)
	}
	handler = corsMiddleware(handler)

	addr := fmt.Sprintf("%s:%d", s.cfg.Web.Bind, s.cfg.Web.Port)
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		s.logger.Info("web dashboard started", "addr", "http://"+addr)
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("web server error", "error", err)
		}
	}()

	return nil
}

// Stop encerra o HTTP server.
func (s *Server) Stop() {
	if s.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.srv.Shutdown(ctx)
	s.logger.Info("web dashboard terminated")
}

// requestLogger loga todas as requisições para o terminal para debug
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stdlog.Printf("WEB: %s %s from %s (Origin: %s)", r.Method, r.URL.Path, r.RemoteAddr, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adiciona headers CORS permitindo acesso local e do tauri
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") || strings.HasPrefix(origin, "tauri://")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

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
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != s.cfg.Web.Username || pass != s.cfg.Web.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.mon.Snapshot())
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status": s.node.State(),
		"uptime": s.node.Uptime().String(),
	})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	lines := 100
	if tail := r.URL.Query().Get("tail"); tail != "" {
		if l, err := strconv.Atoi(tail); err == nil {
			lines = l
		}
	}
	writeJSON(w, map[string]interface{}{
		"lines": s.logger.Tail(lines),
	})
}

func (s *Server) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]interface{}{
			"environment": s.cfg.Node.ExtraArgs,
			"config":      s.cfg,
		})
	case http.MethodPost:
		var req struct {
			Environment string `json:"environment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request", http.StatusBadRequest)
			return
		}
		req.Environment = strings.TrimSpace(req.Environment)
		if req.Environment == "" {
			httpError(w, "environment cannot be empty", http.StatusBadRequest)
			return
		}

		s.cfg.Node.ExtraArgs = replaceOrAppendArg(s.cfg.Node.ExtraArgs, "--node.environment=", "--node.environment="+req.Environment)
		if err := config.Save(s.cfg); err != nil {
			httpError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if s.node.IsRunning() {
			if err := s.node.Restart(); err != nil {
				httpError(w, "config saved, but restart failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		writeJSON(w, map[string]interface{}{
			"ok":          true,
			"environment": req.Environment,
		})
	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		httpError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Persiste no arquivo.
	if err := config.Save(&newCfg); err != nil {
		httpError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Atualiza em memória (atencao: isso não reinicia componentes automaticamente)
	*s.cfg = newCfg
	s.logger.Info("configuration updated and persisted")

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd struct {
		Action  string `json:"action"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		httpError(w, "invalid request", http.StatusBadRequest)
		return
	}

	action := strings.TrimSpace(cmd.Action)
	if action == "" {
		action = strings.TrimSpace(cmd.Command)
	}
	s.logger.Info("received remote command", "action", action)

	var err error
	switch action {
	case "start":
		err = s.node.Start()
	case "stop":
		err = s.node.Stop()
	case "restart":
		err = s.node.Restart()
	default:
		httpError(w, "unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleExplorerSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		httpError(w, "query required", http.StatusBadRequest)
		return
	}
	typ, res, err := s.explorer.Search(query)
	if err != nil {
		httpError(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]interface{}{"type": typ, "data": res})
}

func (s *Server) handleExplorerBlock(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	num := r.URL.Query().Get("number")
	var res interface{}
	var err error
	if hash != "" {
		res, err = s.explorer.GetBlockByHash(hash)
	} else if num != "" {
		res, err = s.explorer.GetBlockByNumber(num)
	} else {
		httpError(w, "hash or number required", http.StatusBadRequest)
		return
	}
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleExplorerTx(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		httpError(w, "hash required", http.StatusBadRequest)
		return
	}
	res, err := s.explorer.GetTransaction(hash)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleExplorerAddress(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("address")
	if addr == "" {
		httpError(w, "address required", http.StatusBadRequest)
		return
	}
	res, err := s.explorer.GetAddressInfo(addr)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, res)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func replaceOrAppendArg(args []string, prefix, replacement string) []string {
	updated := make([]string, 0, len(args)+1)
	replaced := false
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			if !replaced {
				updated = append(updated, replacement)
				replaced = true
			}
			continue
		}
		updated = append(updated, arg)
	}
	if !replaced {
		updated = append(updated, replacement)
	}
	return updated
}
