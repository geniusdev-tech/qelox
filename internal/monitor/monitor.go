// Package monitor — collects system and go-quai node metrics.
// Detects freeze when block height stops advancing.
// FIX: http.Client singleton, fixed freeze detection, expanded metrics.
package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/zeus/qelox/internal/config"
	"github.com/zeus/qelox/internal/log"
	"github.com/zeus/qelox/internal/node"
)

// Metrics contains all collected metrics at an instant.
type Metrics struct {
	Timestamp time.Time `json:"timestamp"`
	NodeState string    `json:"node_state"`
	Uptime    string    `json:"uptime"`
	Restarts  int       `json:"restarts"`

	// CPU / RAM sistema
	CPUPercent float64 `json:"cpu_percent"`
	RAMBytes   uint64  `json:"ram_bytes"`
	RAMPercent float64 `json:"ram_percent"`
	LoadAvg1   float64 `json:"load_avg_1"`
	LoadAvg5   float64 `json:"load_avg_5"`
	LoadAvg15  float64 `json:"load_avg_15"`

	// Disco (data dir do go-quai)
	DiskUsedBytes  uint64  `json:"disk_used_bytes"`
	DiskFreeBytes  uint64  `json:"disk_free_bytes"`
	DiskUsedPct    float64 `json:"disk_used_pct"`
	DiskReadBytes  uint64  `json:"disk_read_bytes"`
	DiskWriteBytes uint64  `json:"disk_write_bytes"`

	// Rede
	NetRecvBytes uint64 `json:"net_recv_bytes"`
	NetSentBytes uint64 `json:"net_sent_bytes"`

	// RAM e Processo específica do go-quai
	GoQuaiRAMBytes   uint64 `json:"go_quai_ram_bytes"`
	GoQuaiThreads    int32  `json:"go_quai_threads"`
	GoQuaiTCPSockets int    `json:"go_quai_tcp_sockets"`

	// Node RPC
	BlockHeight       uint64 `json:"block_height"`
	PeerCount         int    `json:"peer_count"`
	SyncStatus        string `json:"sync_status"`
	TxPending         int    `json:"tx_pending"`
	TxQueued          int    `json:"tx_queued"`
	GasPrice          string `json:"gas_price"`
	NetworkID         string `json:"network_id"`
	NodeClientVersion string `json:"node_client_version"`
	RPCURL            string `json:"rpc_url"`

	// Saúde
	HealthScore int  `json:"health_score"`
	LowPeers    bool `json:"low_peers"`

	// Operacional
	Frozen          bool      `json:"frozen"`
	FreezeFor       string    `json:"freeze_for,omitempty"`
	BlocksPerMinute float64   `json:"blocks_per_minute"`
	LastRestartAt   time.Time `json:"last_restart_at,omitempty"`
	LastCrashReason string    `json:"last_crash_reason,omitempty"`
	CurrentEnv      string    `json:"current_env"`
	SliceID         string    `json:"slice_id"`
}

// Monitor gerencia o loop de coleta.
type Monitor struct {
	cfg  *config.Config
	node *node.Controller
	log  *log.Logger

	mu      sync.RWMutex
	current Metrics
	ticker  *time.Ticker
	done    chan struct{}

	// Estado interno de freeze / blocks per minute
	lastHeight   uint64
	lastHeightAt time.Time
	heightWindow []heightSample // janela para calcular blocos/min

	// FIX: singleton http.Client com pool de conexões TCP.
	httpClient *http.Client

	// Cache de IO para delta
	prevDiskRead  uint64
	prevDiskWrite uint64
	prevNetRecv   uint64
	prevNetSent   uint64
}

type heightSample struct {
	height uint64
	at     time.Time
}

// New cria um Monitor.
func New(cfg *config.Config, nc *node.Controller, logger *log.Logger) *Monitor {
	return &Monitor{
		cfg:  cfg,
		node: nc,
		log:  logger,
		done: make(chan struct{}),
		// FIX: http.Client criado uma única vez, reutiliza pool TCP.
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    5,
				IdleConnTimeout: 30 * time.Second,
			},
		},
	}
}

// Start inicia o loop de coleta em goroutine.
func (m *Monitor) Start() {
	interval := time.Duration(m.cfg.Monitor.IntervalSec) * time.Second
	m.ticker = time.NewTicker(interval)
	go m.loop()
	m.log.Info("monitor started", "interval", interval)
}

// Stop encerra o loop.
func (m *Monitor) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	close(m.done)
}

// Snapshot retorna cópia thread-safe das métricas.
func (m *Monitor) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Monitor) loop() {
	for {
		select {
		case <-m.done:
			return
		case <-m.ticker.C:
			m.collect()
		}
	}
}

func (m *Monitor) collect() {
	var currentEnv, sliceID string
	for _, arg := range m.cfg.Node.ExtraArgs {
		if strings.HasPrefix(arg, "--node.environment=") {
			currentEnv = strings.TrimPrefix(arg, "--node.environment=")
		}
		if strings.HasPrefix(arg, "--node.slices=") {
			sliceID = strings.TrimPrefix(arg, "--node.slices=")
		}
	}

	metrics := Metrics{
		Timestamp:       time.Now(),
		NodeState:       m.node.State().String(),
		CurrentEnv:      currentEnv,
		SliceID:         sliceID,
		Uptime:          m.node.Uptime().Round(time.Second).String(),
		Restarts:        m.node.Restarts(),
		LastRestartAt:   m.node.LastRestartAt(),
		LastCrashReason: m.node.LastCrashReason(),
		RPCURL:          m.cfg.Monitor.RPCURL,
	}

	// FIX: cpu.Percent bloqueante (100ms) chamado FORA do lock principal.
	if percs, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(percs) > 0 {
		metrics.CPUPercent = float64(int(percs[0]*100)) / 100
	}

	// RAM sistema.
	if vm, err := mem.VirtualMemory(); err == nil {
		metrics.RAMBytes = vm.Used
		metrics.RAMPercent = vm.UsedPercent
	}

	// Load average.
	if avg, err := load.Avg(); err == nil {
		metrics.LoadAvg1 = avg.Load1
		metrics.LoadAvg5 = avg.Load5
		metrics.LoadAvg15 = avg.Load15
	}

	// Disco (data dir do go-quai).
	if usage, err := disk.Usage(m.cfg.Node.DataDir); err == nil {
		metrics.DiskUsedBytes = usage.Used
		metrics.DiskFreeBytes = usage.Free
		metrics.DiskUsedPct = usage.UsedPercent
	}

	// I/O de disco (delta).
	if counters, err := disk.IOCounters(); err == nil {
		var rBytes, wBytes uint64
		for _, c := range counters {
			rBytes += c.ReadBytes
			wBytes += c.WriteBytes
		}
		if m.prevDiskRead > 0 {
			metrics.DiskReadBytes = rBytes - m.prevDiskRead
			metrics.DiskWriteBytes = wBytes - m.prevDiskWrite
		}
		m.prevDiskRead = rBytes
		m.prevDiskWrite = wBytes
	}

	// I/O de rede (delta).
	if iocs, err := net.IOCounters(false); err == nil && len(iocs) > 0 {
		recv := iocs[0].BytesRecv
		sent := iocs[0].BytesSent
		if m.prevNetRecv > 0 {
			metrics.NetRecvBytes = recv - m.prevNetRecv
			metrics.NetSentBytes = sent - m.prevNetSent
		}
		m.prevNetRecv = recv
		m.prevNetSent = sent
	}

	// RAM e Métricas do processo go-quai especificamente.
	if m.node.IsRunning() {
		if pid := m.getGoQuaiPID(); pid > 0 {
			if p, err := process.NewProcess(int32(pid)); err == nil {
				if mi, err := p.MemoryInfo(); err == nil {
					metrics.GoQuaiRAMBytes = mi.RSS
				}
				if numThreads, err := p.NumThreads(); err == nil {
					metrics.GoQuaiThreads = numThreads
				}
				if conns, err := p.Connections(); err == nil {
					// Conta apenas sockets TCP ESTABELECIDOS (status "ESTABLISHED" no linux = 1)
					var established int
					for _, c := range conns {
						if c.Type == 1 && c.Status == "ESTABLISHED" {
							established++
						}
					}
					metrics.GoQuaiTCPSockets = established
				}
			}
		}
	}

	// Métricas do node via RPC.
	if m.node.IsRunning() {
		metrics.BlockHeight, metrics.PeerCount, metrics.SyncStatus,
			metrics.TxPending, metrics.TxQueued, metrics.GasPrice,
			metrics.NetworkID, metrics.NodeClientVersion = m.queryNodeMetrics()
		// PeerCount -1 = API unavailable in this go-quai version
	} else {
		metrics.SyncStatus = "offline"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// FIX: freeze detection — só verificar quando node está rodando.
	if m.node.IsRunning() {
		if metrics.BlockHeight > m.lastHeight {
			m.lastHeight = metrics.BlockHeight
			m.lastHeightAt = time.Now()
			// Janela deslizante para blocos por minuto.
			m.heightWindow = append(m.heightWindow, heightSample{metrics.BlockHeight, m.lastHeightAt})
			if len(m.heightWindow) > 30 {
				m.heightWindow = m.heightWindow[1:]
			}
		}
		if !m.lastHeightAt.IsZero() {
			frozen := time.Since(m.lastHeightAt)
			limit := time.Duration(m.cfg.Monitor.FreezeTimeoutMin) * time.Minute
			if frozen > limit {
				metrics.Frozen = true
				metrics.FreezeFor = frozen.Round(time.Second).String()
				m.log.Warn("ALERT: go-quai possibly frozen", "no_new_block", frozen.Round(time.Second).String())
			}
		}
		// Calcula blocos/min over janela de 5 amostras.
		if len(m.heightWindow) >= 2 {
			first := m.heightWindow[0]
			last := m.heightWindow[len(m.heightWindow)-1]
			elapsed := last.at.Sub(first.at).Minutes()
			if elapsed > 0 {
				metrics.BlocksPerMinute = float64(last.height-first.height) / elapsed
			}
		}
	} else {
		// Zerar estado de freeze e altura quando node não está rodando,
		// para que blocos pós-reinício sejam corretamente detectados.
		m.lastHeight = 0
		m.lastHeightAt = time.Time{}
		m.heightWindow = nil
	}

	// Calcula Health Score
	if !m.node.IsRunning() || metrics.Frozen {
		metrics.HealthScore = 0
	} else {
		score := 100

		// Penalidade de Peers
		metrics.LowPeers = metrics.GoQuaiTCPSockets < m.cfg.Monitor.MinPeers
		if metrics.LowPeers {
			diff := m.cfg.Monitor.MinPeers - metrics.GoQuaiTCPSockets
			score -= diff * 3
		}

		// Penalidade de Recursos
		if metrics.CPUPercent > 90 {
			score -= 10
		}
		if metrics.RAMPercent > 90 {
			score -= 10
		}
		if metrics.DiskUsedPct > 95 {
			score -= 20
		}

		if score < 0 {
			score = 0
		}
		metrics.HealthScore = score
	}

	m.current = metrics
}

func (m *Monitor) getGoQuaiPID() int {
	// Procura em /proc pelo processo go-quai.
	procs, err := process.Processes()
	if err != nil {
		return 0
	}
	for _, p := range procs {
		name, err := p.Name()
		if err == nil && name == "go-quai" {
			return int(p.Pid)
		}
	}
	return 0
}

// queryNodeMetrics consulta RPC do go-quai.
// Nota: net_peerCount e quai_syncing não existem nesta versão do go-quai.
func (m *Monitor) queryNodeMetrics() (blockHeight uint64, peers int, syncStatus string,
	txPending, txQueued int, gasPrice, networkID, clientVersion string) {
	rpcURL := m.cfg.Monitor.RPCURL
	peers = -1 // -1 = API indisponível nesta versão
	syncStatus = "synced"

	// 1. Block Number — retorna hexutil.Uint64 sem parâmetros.
	//    O resultado JSON vem como string hex tipo "0x1a2b".
	if res, err := m.rpcCall(rpcURL, "quai_blockNumber", nil); err == nil {
		switch v := res.(type) {
		case string:
			// TrimPrefix case-insensitive para cobrir "0X..." eventualmente.
			s := strings.TrimPrefix(strings.ToLower(v), "0x")
			if val, err := strconv.ParseUint(s, 16, 64); err == nil {
				blockHeight = val
			}
		case float64: // fallback caso o decoder JSON retorne número
			blockHeight = uint64(v)
		}
	}

	// 2. Peer Count — net_peerCount não existe nesta versão do go-quai.
	//    Mantém peers = -1 como indicador de "não disponível".

	// 3. Sync Status — quai_syncing não existe nesta versão do go-quai.
	//    Usa net_listening como proxy: se o node está escutando, está online.
	if res, err := m.rpcCall(rpcURL, "net_listening", nil); err == nil {
		if listening, ok := res.(bool); ok {
			if listening {
				syncStatus = "listening"
			} else {
				syncStatus = "not listening"
			}
		}
	} else {
		syncStatus = "offline"
	}

	// 4. TxPool Status — disponível apenas em zone chains.
	if res, err := m.rpcCall(rpcURL, "txpool_status", nil); err == nil {
		if obj, ok := res.(map[string]interface{}); ok {
			if v, ok := obj["pending"].(string); ok {
				v = strings.TrimPrefix(v, "0x")
				if val, err := strconv.ParseInt(v, 16, 32); err == nil {
					txPending = int(val)
				}
			}
			if v, ok := obj["queued"].(string); ok {
				v = strings.TrimPrefix(v, "0x")
				if val, err := strconv.ParseInt(v, 16, 32); err == nil {
					txQueued = int(val)
				}
			}
		}
	}

	// 5. Gas Price (zone-chain only, pode falhar em prime/region)
	if res, err := m.rpcCall(rpcURL, "quai_gasPrice", nil); err == nil {
		if s, ok := res.(string); ok {
			gasPrice = s
		}
	}

	// 6. Network ID
	if res, err := m.rpcCall(rpcURL, "net_version", nil); err == nil {
		if s, ok := res.(string); ok {
			networkID = s
		}
	}

	// 7. Client Version — quai_clientVersion substitui web3_clientVersion nesta versão
	if res, err := m.rpcCall(rpcURL, "quai_clientVersion", nil); err == nil {
		if s, ok := res.(string); ok {
			clientVersion = s
		}
	}

	return
}

// rpcCall faz uma chamada JSON-RPC ao go-quai usando o http.Client singleton.
func (m *Monitor) rpcCall(url, method string, params []interface{}) (interface{}, error) {
	if params == nil {
		params = []interface{}{}
	}
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	})

	// FIX: usa m.httpClient (singleton) em vez de criar novo a cada chamada.
	resp, err := m.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result interface{} `json:"result"`
		Error  interface{} `json:"error"`
	}
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %v", rpcResp.Error)
	}

	return rpcResp.Result, nil
}
