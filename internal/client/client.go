// Package client — conecta ao daemon via UNIX socket e envia comandos.
// FIX: buffer dinâmico, socket path via UserHomeDir, deadline antes do Write.
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

const dialTimeout = 5 * time.Second

// Client envia comandos ao daemon.
type Client struct {
	socketPath string
}

// Response espelha a resposta do daemon.
type Response struct {
	OK      bool            `json:"ok"`
	Error   string          `json:"error,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// New cria um Client. Usa QELOX_SOCKET env se definido.
// FIX: socketPath padrão usa UserHomeDir() em vez de path hardcoded.
func New() *Client {
	path := os.Getenv("QELOX_SOCKET")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
		}
		path = filepath.Join(home, "qelox", "runtime", "qelox.sock")
	}
	return &Client{socketPath: path}
}

// Send envia um comando e retorna string formatada.
func (c *Client) Send(command string) (string, error) {
	resp, err := c.SendRaw(command)
	if err != nil {
		return "", fmt.Errorf("não foi possível conectar ao daemon: %w\n"+
			"  Verifique se qeloxd está rodando: systemctl status qeloxd", err)
	}
	if !resp.OK {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return formatPayload(resp.Payload), nil
}

// SendRaw envia um comando e retorna Response bruta.
func (c *Client) SendRaw(command string) (*Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, dialTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	req, _ := json.Marshal(map[string]string{"command": command})

	// FIX: SetDeadline ANTES de Write e Read.
	if err := conn.SetDeadline(time.Now().Add(dialTimeout)); err != nil {
		return nil, err
	}
	if _, err := conn.Write(req); err != nil {
		return nil, err
	}

	// FIX: buffer dinâmico via io.ReadAll — sem truncamento de resposta.
	data, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("resposta inválida do daemon")
	}
	return &resp, nil
}

func formatPayload(raw json.RawMessage) string {
	if raw == nil {
		return "ok"
	}
	var obj interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		out, _ := json.MarshalIndent(obj, "", "  ")
		return string(out)
	}
	return string(raw)
}
