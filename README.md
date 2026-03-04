# QELO-X — go-quai Node Orchestrator

Daemon profissional e SaaS dashboard para orquestração do node **go-quai** (Quai Network).

## O que há de novo na versão SaaS?
O QELO-X evoluiu de um script simples para uma verdadeira central de monitoramento avançada com features exclusivas:
- **🔄 Auto-Environment Switch:** Troque dinamicamente de rede (Cyprus, Orchard, Colosseum, Garden) diretamente do painel Web com 1 clique. O QELO-X cuidará de adaptar o *data-dir* correspondente sem vazamentos.
- **💯 Sistema de Health Score:** Um índice aglutinado (0 a 100) calcula a pureza dos recursos de RAM, CPU, Disco e sua conexão TCP P2P em tempo real para sinalizar nós em perigo.
- **🚨 Alerta de Peer-Drop:** Detecta ativamente o declínio de sockets TCP abaixo de níveis saudáveis (<10), acendendo alertas vibrantes para operadores intervirem.
- **🔒 Web Protected (Basic Auth):** Injeção de autenticação nativa e leve para poder expor seu nó na nuvem livremente via IP.
- **🌐 Multi-Node Aware:** Sabe ler exatamente em qual fatia (Slice ID) o host está alocado (`[0 0]` etc).

## Estrutura

```text
qelox/
├── cmd/qeloxd/main.go       # Daemon principal (Orquestrador)
├── cmd/qelox/main.go        # CLI e TUI Interativa
├── internal/
│   ├── config/              # config.toml loader & Auth Configs
│   ├── daemon/              # Núcleo do daemon
│   ├── node/                # Controller (Supervisiona o go-quai)
│   ├── socket/              # Servidor UNIX socket local
│   ├── monitor/             # Telemetria rica (Processos + Rede + API JSON)
│   ├── client/              # Client CLI
│   ├── tui/                 # Painel Bubble Tea Dual-column
│   ├── web/                 # Servidor Web UI Embutido
│   └── log/                 # Logger modular
├── deploy/qeloxd.service    # Daemon Systemd
├── config.toml              # Arquivo de configurações
├── install.sh               # Instalador Rápido
└── uninstall.sh             # Desinstalador
```

## Instalação rápida

```bash
git clone https://github.com/zeus/qelox.git
cd qelox
bash install.sh
```

## Uso e Monitoramento

- **Web Dashboard:** Abra seu navegador em `http://localhost:9201` ou usando seu IP em caso de VPS (Lembre-se de por sua senha no `config.toml`).
- **TUI Interativa:** `qelox tui`

Comandos do daemon:
```bash
qelox start | stop | restart | status | metrics
sudo systemctl start qeloxd
journalctl -u qeloxd -f
```

### Teclas do Terminal UI (TUI)

| Tecla | Ação |
|-------|------|
| S | Start |
| X | Stop |
| R | Restart |
| Q | Quit |

Veja o guia completo na interface web do QELO-X.
