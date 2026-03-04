/* ── QELO-X Dashboard — app.js ──────────────────────────────────────── */
'use strict';

const API = {
    metrics: '/api/stats',
    status: '/api/status',
    logs: '/api/logs',
    command: '/api/command',
};

let metricsInterval = null;
let logsInterval = null;

// ── Bootstrap ─────────────────────────────────────────────────────────
window.addEventListener('DOMContentLoaded', () => {
    startPolling();
});

function startPolling() {
    fetchMetrics();
    fetchLogs();
    metricsInterval = setInterval(fetchMetrics, 2000);
    logsInterval = setInterval(fetchLogs, 3000);
}

// ── Metrics ───────────────────────────────────────────────────────────
async function fetchMetrics() {
    try {
        const res = await fetch(API.metrics);
        if (res.status === 401) {
            setFooter('authentication required (refresh page)', '#f43f5e');
            return;
        }
        if (!res.ok) throw new Error('HTTP ' + res.status);
        const m = await res.json();
        renderMetrics(m);
        setFooter('connected', '#00ff65');
    } catch (e) {
        setFooter('daemon offline — ' + e.message, '#f43f5e');
        setBadge('OFFLINE', 'stopped');
    }
}

function renderMetrics(m) {
    // Badge / header
    const state = (m.node_state || 'UNKNOWN').toLowerCase();
    const badgeClass = state === 'running' ? 'running' : (state === 'crashed' ? 'crashed' : 'stopped');
    setBadge(m.node_state || '—', badgeClass);

    // Top cards
    setText('node-state', m.node_state || '—', stateClass(m.node_state));
    setText('node-uptime', 'uptime: ' + (m.uptime || '—'));
    setText('block-height', '#' + (m.block_height || 0).toLocaleString());
    setText('blocks-per-min', (m.blocks_per_minute > 0
        ? m.blocks_per_minute.toFixed(2) + ' blocks/min' : 'no new blocks'));
    setText('peer-count', m.peer_count === -1 ? 'N/A' : (m.peer_count ?? '—'));
    setText('sync-status', (m.sync_status || '—').toUpperCase());
    setText('restarts', m.restarts ?? '—');
    setText('last-crash', m.last_crash_reason
        ? 'last crash: ' + m.last_crash_reason.slice(0, 40)
        : (m.last_restart_at ? 'restarted: ' + fmtTime(m.last_restart_at) : '—'));

    // CPU
    const cpu = m.cpu_percent || 0;
    setText('cpu-pct-label', cpu.toFixed(1) + '%');
    setBar('cpu-bar', cpu);

    // RAM
    const ramPct = m.ram_percent || 0;
    const ramMB = ((m.ram_bytes || 0) / 1024 / 1024).toFixed(0);
    const gqMB = ((m.go_quai_ram_bytes || 0) / 1024 / 1024).toFixed(0);
    setText('ram-pct-label', ramPct.toFixed(1) + '%');
    setBar('ram-bar', ramPct);
    setText('ram-detail', `system: ${ramMB} MB · go-quai: ${gqMB} MB`);

    // Load avg
    setText('load-avg', `load avg: ${(m.load_avg_1 || 0).toFixed(2)} / ${(m.load_avg_5 || 0).toFixed(2)} / ${(m.load_avg_15 || 0).toFixed(2)}`);
    setText('load1', (m.load_avg_1 || 0).toFixed(2));
    setText('load5', (m.load_avg_5 || 0).toFixed(2));
    setText('load15', (m.load_avg_15 || 0).toFixed(2));

    // Disk
    const diskPct = m.disk_used_pct || 0;
    const diskUsed = fmtBytes(m.disk_used_bytes || 0);
    const diskFree = fmtBytes(m.disk_free_bytes || 0);
    setBar('disk-bar', diskPct);
    setText('disk-detail', `${diskUsed} used · ${diskFree} free`);

    // Network
    setText('net-recv', '↓ ' + fmtBytes(m.net_recv_bytes || 0) + '/s');
    setText('net-sent', '↑ ' + fmtBytes(m.net_sent_bytes || 0) + '/s');

    // Health Score
    const hScore = m.health_score || 0;
    setText('health-score', hScore);
    const hPath = document.getElementById('health-path');
    const hCard = document.getElementById('health-card');
    if (hPath) {
        hPath.style.strokeDasharray = `${hScore}, 100`;
        if (hScore >= 90) {
            hPath.style.stroke = 'var(--brand)';
            if (hCard) hCard.className = 'card card-glow-green';
            setText('health-status', 'Excellent');
        } else if (hScore >= 70) {
            hPath.style.stroke = 'var(--amber)';
            if (hCard) hCard.className = 'card card-glow-amber';
            setText('health-status', 'Warning');
        } else {
            hPath.style.stroke = 'var(--red)';
            if (hCard) hCard.className = 'card card-warn';
            setText('health-status', 'Critical');
        }
    }

    // Process Info (Substituindo TxPool inativo na API)
    setText('tcp-sockets', m.go_quai_tcp_sockets ?? '—');
    setText('active-threads', m.go_quai_threads ?? '—');
    setText('target-slice', m.slice_id || 'Global');

    // Alerta piscante de Peers
    const pCard = document.getElementById('process-card');
    if (pCard) {
        if (m.low_peers) {
            pCard.classList.add('card-warn');
        } else {
            pCard.classList.remove('card-warn');
        }
    }

    // Peer Count Fallback usando os Sockets:
    if (m.peer_count === -1 && m.go_quai_tcp_sockets > 0) {
        setText('peer-count', `~${m.go_quai_tcp_sockets}`);
    }

    // Node info
    setText('network-id', m.network_id || '—');
    setText('client-version', m.node_client_version || '—');
    setText('rpc-url', m.rpc_url || '—');

    // Environment Select
    if (m.current_env) {
        const sel = document.getElementById('env-select');
        if (sel && sel.value !== m.current_env && document.activeElement !== sel) {
            sel.value = m.current_env;
        }
    }

    // Freeze alert
    const freezeEl = document.getElementById('freeze-alert');
    if (m.frozen) {
        freezeEl.style.display = '';
        setText('freeze-msg', `FREEZE DETECTED — no new blocks for ${m.freeze_for || '?'}`);
    } else {
        freezeEl.style.display = 'none';
    }

    // Footer timestamp
    setText('last-update', new Date().toLocaleTimeString('en-US'));
}

// ── Logs ──────────────────────────────────────────────────────────────
async function fetchLogs() {
    const tail = document.getElementById('tail-select')?.value || 100;
    try {
        const res = await fetch(`${API.logs}?tail=${tail}`);
        if (!res.ok) return;
        const data = await res.json();
        renderLogs(data.lines || []);
    } catch (_) { }
}

function renderLogs(lines) {
    const box = document.getElementById('logs-box');
    if (!box) return;
    const autoscroll = document.getElementById('autoscroll')?.checked;
    const wasAtBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 40;

    box.innerHTML = lines.map(line => {
        let cls = 'log-line';
        try {
            const parsed = JSON.parse(line);
            cls += ' ' + (parsed.level || 'INFO');
        } catch (_) { }
        return `<div class="${cls}">${escHtml(line)}</div>`;
    }).join('');

    if (autoscroll && wasAtBottom) {
        box.scrollTop = box.scrollHeight;
    }
}

// ── Commands ──────────────────────────────────────────────────────────
async function sendCommand(cmd) {
    const fb = document.getElementById('cmd-feedback');
    fb.className = 'cmd-feedback';
    fb.textContent = '';

    try {
        const res = await fetch(API.command, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ command: cmd }),
        });
        const data = await res.json();
        if (data.ok || res.ok) {
            showFeedback(`✓ ${cmd.toUpperCase()} executed`, 'ok');
        } else {
            showFeedback(`✗ ${data.error || 'erro'}`, 'err');
        }
    } catch (e) {
        showFeedback('✗ daemon offline', 'err');
    }

    // Força atualização imediata das métricas após 600ms.
    setTimeout(fetchMetrics, 600);
}

function showFeedback(msg, cls) {
    const fb = document.getElementById('cmd-feedback');
    fb.textContent = msg;
    fb.className = `cmd-feedback ${cls} show`;
    setTimeout(() => { fb.className = 'cmd-feedback'; }, 3500);
}

// ── Environment ───────────────────────────────────────────────────────
async function changeEnvironment(env) {
    if (!confirm(`Switch network to ${env.toUpperCase()}? The node will be paused, config rewritten and restarted automatically.`)) {
        setTimeout(fetchMetrics, 100);
        return;
    }
    const fb = document.getElementById('cmd-feedback');
    fb.className = 'cmd-feedback';
    fb.textContent = 'Migrating...';

    try {
        const res = await fetch('/api/config/environment', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ environment: env }),
        });
        const data = await res.json();
        if (data.ok || res.ok) {
            showFeedback(`✓ Migrated to ${env}`, 'ok');
        } else {
            showFeedback(`✗ Error: ${data.error}`, 'err');
        }
    } catch (e) {
        showFeedback('✗ daemon unavailable', 'err');
    }
    setTimeout(fetchMetrics, 2000);
}

// ── Helpers ───────────────────────────────────────────────────────────
function setText(id, val, cls) {
    const el = document.getElementById(id);
    if (!el) return;
    el.textContent = val;
    if (cls) { el.className = `card-value ${cls}`; }
}

function setBar(id, pct) {
    const el = document.getElementById(id);
    if (!el) return;
    el.style.width = Math.min(pct, 100).toFixed(1) + '%';
}

function setBadge(text, cls) {
    const badge = document.getElementById('node-badge');
    const dot = document.getElementById('badge-dot');
    const label = document.getElementById('badge-text');
    if (badge) badge.className = 'node-badge ' + cls;
    if (dot) dot.className = 'badge-dot ' + cls;
    if (label) label.textContent = text;
}

function setFooter(msg, color) {
    const el = document.getElementById('footer-status');
    if (el) { el.textContent = msg; el.style.color = color || ''; }
}

function stateClass(state) {
    if (!state) return '';
    const s = state.toLowerCase();
    if (s === 'running') return 'card-value state-running';
    if (s === 'crashed') return 'card-value state-crashed';
    if (s === 'stopped') return 'card-value state-stopped';
    return 'card-value state-starting';
}

function fmtBytes(b) {
    if (b === 0) return '0 B';
    const k = 1024, sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(b) / Math.log(k));
    return (b / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
}

function fmtTime(iso) {
    try { return new Date(iso).toLocaleString('en-US'); } catch { return iso; }
}

function escHtml(str) {
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}
