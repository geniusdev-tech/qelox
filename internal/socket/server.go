// Package socket — servidor UNIX domain socket do qeloxd.
// FIX: permissão do socket alterada de 0666 para 0660.
package socket

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/log"
	"github.com/zeus/qelox/internal/monitor"
	"github.com/zeus/qelox/internal/node"
)

// FIX: 0666 → 0660 — apenas usuário+grupo podem conectar.
const socketPerm = 0660

// Request é o comando enviado pelo client.
type Request struct {
	Command string `json:"command"`
}

// Response é a resposta padronizada do daemon.
type Response struct {
	OK      bool        `json:"ok"`
	Error   string      `json:"error,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// Server mantém o listener UNIX socket.
type Server struct {
	cfg      *config.Config
	node     *node.Controller
	mon      *monitor.Monitor
	log      *log.Logger
	listener net.Listener
	quit     chan struct{}
}

// NewServer cria um Server.
func NewServer(cfg *config.Config, nc *node.Controller, mon *monitor.Monitor, logger *log.Logger) *Server {
	return &Server{cfg: cfg, node: nc, mon: mon, log: logger, quit: make(chan struct{})}
}

// Start abre o socket e inicia loop de aceitação.
func (s *Server) Start() error {
	os.Remove(s.cfg.Daemon.SocketPath)

	ln, err := net.Listen("unix", s.cfg.Daemon.SocketPath)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.cfg.Daemon.SocketPath, socketPerm); err != nil {
		ln.Close()
		return err
	}

	s.listener = ln
	s.log.Info("socket UNIX aberto", "path", s.cfg.Daemon.SocketPath, "perm", "0660")
	go s.acceptLoop()
	return nil
}

// Stop fecha o listener.
func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				s.log.Error("erro ao aceitar conexão", "error", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	var req Request
	// FIX: limit input size to 4KB to prevent unbounded payload DoS
	limitReader := io.LimitReader(conn, 4096)
	if err := json.NewDecoder(limitReader).Decode(&req); err != nil {
		if err != io.EOF {
			s.log.Error("erro ao decodificar request", "error", err)
		}
		return
	}
	s.writeResp(conn, s.dispatch(strings.ToLower(strings.TrimSpace(req.Command))))
}

func (s *Server) dispatch(cmd string) Response {
	switch cmd {
	case "start":
		if err := s.node.Start(); err != nil {
			return Response{Error: err.Error()}
		}
		return Response{OK: true, Payload: "go-quai iniciado"}
	case "stop":
		if err := s.node.Stop(); err != nil {
			return Response{Error: err.Error()}
		}
		return Response{OK: true, Payload: "go-quai parado"}
	case "restart":
		if err := s.node.Restart(); err != nil {
			return Response{Error: err.Error()}
		}
		return Response{OK: true, Payload: "go-quai reiniciado"}
	case "status":
		return Response{OK: true, Payload: map[string]interface{}{
			"state":    s.node.State().String(),
			"uptime":   s.node.Uptime().String(),
			"restarts": s.node.Restarts(),
		}}
	case "metrics":
		return Response{OK: true, Payload: s.mon.Snapshot()}
	case "version":
		return Response{OK: true, Payload: "qeloxd v1.1.0"}
	default:
		return Response{Error: "comando desconhecido: " + cmd}
	}
}

func (s *Server) writeResp(conn net.Conn, resp Response) {
	data, _ := json.Marshal(resp)
	conn.Write(data)
}
