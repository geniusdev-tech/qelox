# QELO-X — go-quai Node Orchestrator

Professional daemon and SaaS dashboard for orchestrating the **go-quai** node (Quai Network).

## What's new in the SaaS version?
QELO-X has evolved from a simple script into a powerful advanced monitoring hub with exclusive features:
- **🔄 Auto-Environment Switch:** Dynamically switch networks (Cyprus, Orchard, Colosseum, Garden) directly from the Web panel with 1 click. QELO-X handles data-dir adaptation automatically.
- **💯 Health Score System:** An aggregated index (0 to 100) calculates RAM, CPU, Disk health, and P2P TCP connection status in real-time.
- **🚨 Peer-Drop Alert:** Actively detects TCP socket decline below healthy levels (<10), triggering vibrant alerts for operators.
- **🔒 Web Protected (Basic Auth):** Native, lightweight authentication support to safely expose your node publicly.
- **🌐 Multi-Node Aware:** Natively detects which slice (Slice ID) the host is allocated to (e.g., `[0 0]`).

## Structure

```text
qelox/
├── cmd/qeloxd/main.go       # Main Daemon (Orchestrator)
├── cmd/qelox/main.go        # Interactive CLI and TUI
├── internal/
│   ├── config/              # config.toml loader & Auth Configs
│   ├── daemon/              # Daemon core
│   ├── node/                # Controller (Supervises go-quai)
│   ├── socket/              # Local UNIX socket server
│   ├── monitor/             # Rich telemetry (Processes + Network + JSON API)
│   ├── client/              # CLI Client
│   ├── tui/                 # Dual-column Bubble Tea Panel
│   ├── web/                 # Embedded Web UI Server
│   └── log/                 # Modular Logger
├── deploy/qeloxd.service    # Systemd Daemon
├── config.toml.example      # Configuration template
├── install.sh               # Quick Installer
└── uninstall.sh             # Uninstaller
```

## Quick Installation

```bash
git clone https://github.com/zeus/qelox.git
cd qelox
bash install.sh
```

## Usage and Monitoring

- **Web Dashboard:** Open your browser at `http://localhost:9201` (Ensure you set your password in `config.toml`).
- **Interactive TUI:** `qelox tui`

Daemon Commands:
```bash
qelox start | stop | restart | status | metrics
sudo systemctl start qeloxd
journalctl -u qeloxd -f
```

### Terminal UI (TUI) Keys

| Key | Action |
|-----|--------|
| S   | Start  |
| X   | Stop   |
| R   | Restart|
| Q   | Quit   |

Check the complete guide on the QELO-X web interface.
