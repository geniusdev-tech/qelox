import { create } from 'zustand';

// Bug fix #3: monotonically increasing counter for stable log IDs across fetches.
let _logSeq = 0;

// Connectivity fix: robustly detect if we are NOT being served directly by the Go backend.
// In prod web mode, the Go server serves us on port 9201. If we are on ANY other port (Vite 1420, Tauri, etc.), we use absolute URL.
const API_BASE: string = window.location.port === '9201' ? '' : 'http://127.0.0.1:9201';
const isTauriRuntime = () => typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;

interface MetricData {
    time: string;
    value: number;
}

interface Metrics {
    cpu: number;
    ram: number;
    disk: number;
    peers: number;
    latency: number;
    syncLag: number;
    blockHeight: number;
    history: {
        cpu: MetricData[];
        ram: MetricData[];
        disk: MetricData[];
        peers: MetricData[];
        latency: MetricData[];
        syncLag: MetricData[];
        blockHeight: MetricData[];
    };
}

interface Log {
    id: string;
    timestamp: string;
    message: string;
    level: 'INFO' | 'WARN' | 'ERROR' | 'SUCCESS';
}

// Explorer Types
export type ExplorerViewType = 'search' | 'block' | 'tx' | 'address';

export interface ExplorerStateData {
    view: ExplorerViewType;
    loading: boolean;
    error: string | null;
    data: any; // Can be Block, Tx, or Address data
}

interface OrchestratorStore {
    // UI State
    viewMode: 'enterprise' | 'operator';
    activeTab: string;
    healthScore: number;
    status: 'RUNNING' | 'DEGRADED' | 'OFFLINE';
    nodeName: string;
    network: string;

    // Data State
    metrics: Metrics;
    logs: Log[];
    explorer: ExplorerStateData;
    currentConfig: any;

    // New Feature State
    installer: {
        installed: boolean;
        progress: number;
        status: string;
        installing: boolean;
    };
    hardware: {
        cpu_cores: number;
        cpu_model: string;
        total_ram_gb: number;
        os_name: string;
        os_version: string;
    } | null;
    update: {
        available: boolean;
        version: string | null;
        dismissed: boolean;
    };

    // Actions
    toggleViewMode: () => void;
    setActiveTab: (tab: string) => void;
    fetchMetrics: () => Promise<void>;
    fetchLogs: () => Promise<void>;
    fetchConfig: () => Promise<void>;
    saveConfig: (cfg: any) => Promise<boolean>;
    searchExplorer: (query: string) => Promise<void>;

    // Member Actions
    startNode: () => Promise<void>;
    stopNode: () => Promise<void>;
    restartNode: () => Promise<void>;
    checkInstall: () => Promise<void>;
    runInstall: () => Promise<void>;
    detectHardware: () => Promise<void>;
}


const createEmptyHistory = () => {
    return Array.from({ length: 30 }, (_, i) => ({
        time: '',
        value: 0,
    }));
};

const sendCommand = async (action: 'start' | 'stop' | 'restart') => {
    const res = await fetch(`${API_BASE}/api/command`, {
        method: 'POST',
        mode: 'cors',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
    });

    const payload = await res.json().catch(() => null);
    if (!res.ok || payload?.error) {
        throw new Error(payload?.error || `Command failed with HTTP ${res.status}`);
    }
};

export const useOrchestratorStore = create<OrchestratorStore>((set, get) => ({
    viewMode: 'enterprise',
    activeTab: 'dashboard',
    healthScore: 0,
    status: 'OFFLINE',
    nodeName: 'CONNECTING...',
    network: 'UNKNOWN',

    metrics: {
        cpu: 0,
        ram: 0,
        disk: 0,
        peers: 0,
        latency: 0,
        syncLag: 0,
        blockHeight: 0,
        history: {
            cpu: createEmptyHistory(),
            ram: createEmptyHistory(),
            disk: createEmptyHistory(),
            peers: createEmptyHistory(),
            latency: createEmptyHistory(),
            syncLag: createEmptyHistory(),
            blockHeight: createEmptyHistory(),
        }
    },

    logs: [],

    explorer: {
        view: 'search',
        loading: false,
        error: null,
        data: null,
    },

    currentConfig: null,

    installer: {
        installed: true,
        progress: 0,
        status: '',
        installing: false,
    },
    hardware: null,
    update: {
        available: false,
        version: null,
        dismissed: false,
    },

    toggleViewMode: () => set((state) => ({
        viewMode: state.viewMode === 'enterprise' ? 'operator' : 'enterprise'
    })),

    setActiveTab: (tab: any) => set({ activeTab: tab }),

    startNode: async () => {
        try {
            await sendCommand('start');
            await get().fetchMetrics();
        } catch (err) {
            console.error('Failed to start node:', err);
        }
    },

    stopNode: async () => {
        try {
            await sendCommand('stop');
            await get().fetchMetrics();
        } catch (err) {
            console.error('Failed to stop node:', err);
        }
    },

    restartNode: async () => {
        try {
            await sendCommand('restart');
            await get().fetchMetrics();
        } catch (err) {
            console.error('Failed to restart node:', err);
        }
    },

    checkInstall: async () => {
        if (!isTauriRuntime()) return;
        try {
            // We use the configured path, or default
            const cfg = get().currentConfig;
            const path = cfg?.node?.binary_path || '/usr/local/bin/go-quai';
            const { invoke } = await import('@tauri-apps/api/core');
            const installed = await invoke('check_node_installed', { path }) as boolean;
            set((state) => ({ installer: { ...state.installer, installed } }));
        } catch (err) {
            console.error('Check install failed:', err);
        }
    },

    detectHardware: async () => {
        if (!isTauriRuntime()) return;
        try {
            const { invoke } = await import('@tauri-apps/api/core');
            const hardware = await invoke('detect_hardware') as any;
            set({ hardware });
        } catch (err) {
            console.error('Hardware detection failed:', err);
        }
    },

    runInstall: async () => {
        if (!isTauriRuntime()) {
            set((state) => ({ installer: { ...state.installer, installing: false, status: 'Error: installer is only available in the desktop app' } }));
            return;
        }
        try {
            const { invoke } = await import('@tauri-apps/api/core');
            const { listen } = await import('@tauri-apps/api/event');

            set((state) => ({ installer: { ...state.installer, installing: true, progress: 0 } }));

            const unlisten = await listen('install-progress', (event: any) => {
                const payload = event.payload;
                set((state) => ({
                    installer: {
                        ...state.installer,
                        progress: payload.percentage,
                        status: payload.status
                    }
                }));
            });

            const homeDir = await import('@tauri-apps/api/path').then(p => p.homeDir());
            const targetDir = `${homeDir}/qelox/bin`;

            await invoke('install_node', { targetDir });

            unlisten();
            set((state) => ({ installer: { ...state.installer, installing: false, installed: true, progress: 100 } }));
        } catch (err: any) {
            set((state) => ({ installer: { ...state.installer, installing: false, status: `Error: ${err}` } }));
        }
    },

    fetchMetrics: async () => {
        try {
            const res = await fetch(`${API_BASE}/api/stats`, {
                mode: 'cors',
                headers: { 'Accept': 'application/json' }
            });
            if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`);
            const data = await res.json();

            // Map the Go backend response to our frontend state
            const newCpu = data.cpu_percent || 0;
            const newRam = data.go_quai_ram_bytes ? parseFloat((data.go_quai_ram_bytes / 1024 / 1024 / 1024).toFixed(2)) : 0;
            const newDisk = data.disk_read_bytes || data.disk_write_bytes ?
                Math.round((data.disk_read_bytes + data.disk_write_bytes) / 1024 / 1024) : 0;
            const newPeers = Math.max(0, data.go_quai_tcp_sockets || 0);
            const blockHeight = data.block_height || 0;

            // Go-Quai block sync speed / latency pseudo-mapping
            const latency = Math.round(data.blocks_per_minute > 0 ? (60000 / data.blocks_per_minute) / 10 : 0);

            // Improvement C: derive syncLag from sync_status
            const syncLag = (data.sync_status === 'offline' || data.sync_status === 'not listening') ? 999 : 0;
            const timestamp = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
            const currentState = get();

            const updateHistory = (key: keyof Metrics['history'], value: number) => {
                const newHistory = [...currentState.metrics.history[key], { time: timestamp, value }];
                return newHistory.slice(-30);
            };

            set({
                status: data.frozen ? 'DEGRADED' : (data.node_state || '').toUpperCase() === 'RUNNING' ? 'RUNNING' : 'OFFLINE',
                healthScore: data.health_score || 0,
                network: data.network_id || 'CYPRUS-1 MAINNET',
                nodeName: data.slice_id || 'QN-ARC-9201',
                metrics: {
                    cpu: newCpu,
                    ram: newRam,
                    disk: newDisk,
                    peers: newPeers,
                    latency: latency,
                    syncLag: syncLag,
                    blockHeight: blockHeight,
                    history: {
                        cpu: updateHistory('cpu', newCpu),
                        ram: updateHistory('ram', newRam),
                        disk: updateHistory('disk', newDisk),
                        peers: updateHistory('peers', newPeers),
                        latency: updateHistory('latency', latency),
                        syncLag: updateHistory('syncLag', syncLag),
                        blockHeight: updateHistory('blockHeight', blockHeight),
                    }
                }
            });
        } catch (err: any) {
            console.error('Error fetching metrics:', err);
            set({
                status: 'OFFLINE',
                healthScore: 0,
                nodeName: `ERR: ${err.message || 'Unknown'}`
            });
        }
    },

    fetchLogs: async () => {
        try {
            const res = await fetch(`${API_BASE}/api/logs?tail=50`, {
                mode: 'cors'
            });
            if (!res.ok) throw new Error('Failed to fetch logs');
            const data = await res.json();

            if (data.lines && Array.isArray(data.lines)) {
                const parsedLogs = data.lines.map((line: string) => {
                    let level: Log['level'] = 'INFO';
                    if (line.includes('level=error') || line.includes('level=fatal') || line.includes('panic')) level = 'ERROR';
                    else if (line.includes('level=warning') || line.includes('ALERT')) level = 'WARN';
                    else if (line.includes('success') || line.includes('validated') || line.includes('sync completed')) level = 'SUCCESS';

                    let timestamp = new Date().toLocaleTimeString();
                    const timeMatch = line.match(/time="([^"]+)"/);
                    if (timeMatch && timeMatch[1]) {
                        timestamp = new Date(timeMatch[1]).toLocaleTimeString();
                    }

                    let message = line;
                    const msgMatch = line.match(/msg="([^"]+)"/);
                    if (msgMatch && msgMatch[1]) {
                        message = msgMatch[1];
                    }

                    return {
                        id: `log-${++_logSeq}`,
                        timestamp,
                        level,
                        message
                    };
                });
                set({ logs: parsedLogs });
            }
        } catch (err) {
            console.error('Error fetching logs:', err);
        }
    },

    fetchConfig: async () => {
        try {
            const res = await fetch(`${API_BASE}/api/config/environment`, { mode: 'cors' });
            if (!res.ok) throw new Error('Failed to fetch config');
            const data = await res.json();
            set({ currentConfig: data.config ?? null });
        } catch (err) {
            console.error('Error fetching config:', err);
        }
    },

    saveConfig: async (cfg: any) => {
        try {
            const res = await fetch(`${API_BASE}/api/config`, {
                method: 'POST',
                mode: 'cors',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(cfg)
            });
            if (!res.ok) throw new Error('Failed to save config');
            set({ currentConfig: cfg });
            return true;
        } catch (err) {
            console.error('Error saving config:', err);
            return false;
        }
    },

    searchExplorer: async (query: string) => {
        if (!query.trim()) return;
        set((state) => ({ explorer: { ...state.explorer, loading: true, error: null } }));
        try {
            const res = await fetch(`${API_BASE}/api/explorer/search?q=${encodeURIComponent(query)}`, {
                mode: 'cors'
            });
            const json = await res.json();

            if (!res.ok || json.error) {
                throw new Error(json.error || 'Search failed');
            }

            set({
                explorer: {
                    view: json.type as ExplorerViewType,
                    loading: false,
                    error: null,
                    data: json.data,
                }
            });
        } catch (err: any) {
            set((state) => ({
                explorer: {
                    ...state.explorer,
                    loading: false,
                    error: err.message
                }
            }));
        }
    },
}));
