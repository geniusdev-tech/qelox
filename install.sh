#!/usr/bin/env bash
# install.sh — Instala o QELO-X no sistema Linux.
# Uso: bash install.sh
# Requer: Go 1.22+, git, sudo

set -euo pipefail

INSTALL_USER="${QELOX_USER:-$(whoami)}"
INSTALL_HOME="/home/${INSTALL_USER}"
QELOX_BASE="${INSTALL_HOME}/qelox"
BINARY_DIR="${QELOX_BASE}/bin"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${CYAN}[qelox]${NC} $*"; }
ok()    { echo -e "${GREEN}[ok]${NC} $*"; }
error() { echo -e "${RED}[erro]${NC} $*"; exit 1; }

info "Instalando QELO-X para usuário: ${INSTALL_USER}"

command -v go >/dev/null 2>&1 || error "Go não encontrado. Instale Go 1.22+: https://golang.org/dl/"
info "Go version: $(go version)"

VERSION="$(cat VERSION)"

info "Criando estrutura de diretórios..."
mkdir -p "${BINARY_DIR}" "${QELOX_BASE}/runtime" "${QELOX_BASE}/logs"

info "Compilando qeloxd..."
go build -ldflags="-s -w -X github.com/zeus/qelox/internal/buildinfo.Version=${VERSION}" -o "${BINARY_DIR}/qeloxd" ./cmd/qeloxd/

info "Compilando qelox..."
go build -ldflags="-s -w" -o "${BINARY_DIR}/qelox" ./cmd/qelox/

chmod 755 "${BINARY_DIR}/qeloxd" "${BINARY_DIR}/qelox"
ok "Binários compilados em ${BINARY_DIR}"

info "Instalando binários em /usr/local/bin..."
sudo ln -sf "${BINARY_DIR}/qelox"  /usr/local/bin/qelox
sudo ln -sf "${BINARY_DIR}/qeloxd" /usr/local/bin/qeloxd
ok "qelox e qeloxd disponíveis no PATH"

if [ ! -f "${QELOX_BASE}/config.toml" ]; then
    cp config.toml "${QELOX_BASE}/config.toml"
    ok "config.toml instalado — edite antes de iniciar"
fi

chown -R "${INSTALL_USER}:${INSTALL_USER}" "${QELOX_BASE}" 2>/dev/null || true

info "Instalando serviço systemd..."
sed "s/User=zeus/User=${INSTALL_USER}/g; s/Group=zeus/Group=${INSTALL_USER}/g; s|/home/zeus|/home/${INSTALL_USER}|g" \
    deploy/qeloxd.service | sudo tee /etc/systemd/system/qeloxd.service > /dev/null

sudo systemctl daemon-reload
sudo systemctl enable qeloxd
ok "qeloxd.service instalado e habilitado"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  QELO-X instalado com sucesso!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  1. Edite:  nano ${QELOX_BASE}/config.toml"
echo "  2. Inicie: sudo systemctl start qeloxd"
echo "  3. Logs:   journalctl -u qeloxd -f"
echo "  4. TUI:    qelox tui"
echo ""
