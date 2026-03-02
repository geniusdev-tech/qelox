#!/usr/bin/env bash
# uninstall.sh — Remove o QELO-X do sistema.
set -euo pipefail
CYAN='\033[0;36m'; GREEN='\033[0;32m'; NC='\033[0m'
info() { echo -e "${CYAN}[qelox]${NC} $*"; }
ok()   { echo -e "${GREEN}[ok]${NC} $*"; }

info "Parando serviço systemd..."
sudo systemctl stop    qeloxd 2>/dev/null || true
sudo systemctl disable qeloxd 2>/dev/null || true
sudo rm -f /etc/systemd/system/qeloxd.service
sudo systemctl daemon-reload
ok "Serviço removido"

info "Removendo binários..."
sudo rm -f /usr/local/bin/qelox /usr/local/bin/qeloxd
ok "Binários removidos"

info "Removendo runtime e binários compilados..."
rm -rf ~/qelox/runtime ~/qelox/bin
ok "Feito"

echo ""
echo "  Logs e config preservados em ~/qelox/"
echo "  Para remover tudo: rm -rf ~/qelox"
echo ""
ok "QELO-X desinstalado."
