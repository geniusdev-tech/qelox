# QELO-X — go-quai Node Orchestrator

Daemon profissional para orquestração do node **go-quai**.

## Estrutura

```
qelox/
├── cmd/qeloxd/main.go       # Daemon principal
├── cmd/qelox/main.go        # CLI/TUI client
├── internal/
│   ├── config/              # config.toml loader
│   ├── daemon/              # Núcleo do daemon
│   ├── node/                # Controller do go-quai
│   ├── socket/              # Servidor UNIX socket
│   ├── monitor/             # CPU, RAM, peers, sync
│   ├── client/              # Client do socket
│   ├── tui/                 # Interface Bubble Tea
│   └── log/                 # Logger estruturado JSON
├── deploy/qeloxd.service    # Unit systemd
├── config.toml              # Configuração exemplo
├── install.sh               # Instalador
└── uninstall.sh             # Desinstalador
```

## Instalação rápida

```bash
git clone https://github.com/zeus/qelox.git
cd qelox
bash install.sh
```

## Uso

```bash
qelox start|stop|restart|status|metrics|tui
sudo systemctl start qeloxd
journalctl -u qeloxd -f
```

## Teclas TUI

| Tecla | Ação |
|-------|------|
| S | Start |
| X | Stop |
| R | Restart |
| Q | Quit |

Veja o guia completo na interface web do QELO-X.
