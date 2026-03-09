import { useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Cpu, Database, HardDrive, Users, Wifi, Timer, Layers } from 'lucide-react';
import { useOrchestratorStore } from './store';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { MetricCard } from './components/MetricCard';
import { Terminal } from './components/Terminal';
import ExplorerLayout from './components/Explorer/ExplorerLayout';
import NodesPage from './components/pages/NodesPage';
import ClustersPage from './components/pages/ClustersPage';
import AlertsPage from './components/pages/AlertsPage';
import LogsPage from './components/pages/LogsPage';
import SettingsPage from './components/pages/SettingsPage';
import SetupWizardPage from './components/pages/SetupWizardPage';
import InstallerPage from './components/pages/InstallerPage';
import { UpdaterBanner } from './components/UpdaterBanner';
import { useUpdater } from './hooks/useUpdater';
import { invoke } from '@tauri-apps/api/core';


const ACCENT_CYAN = '#00d9ff';

const ACCENT_NEON = '#00ff88';
const ACCENT_YELLOW = '#ffc857';
const isTauriRuntime = () => typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;

// Page titles config
const PAGE_META: Record<string, { title: string; subtitle: string; tag: string }> = {
    dashboard: {
        title: 'System Overview',
        subtitle: 'Real-time node cluster orchestration & multi-metrics synchronization',
        tag: '// GRID-01',
    },
    nodes: {
        title: 'Node Registry',
        subtitle: 'Manage and monitor individual QUAI nodes in your cluster',
        tag: '// ZONE: EU-WEST-1A',
    },
    clusters: {
        title: 'Cluster Registry',
        subtitle: 'Multi-node cluster orchestration and slice management',
        tag: '// QUAI NETWORK',
    },
    alerts: {
        title: 'Active Alerts',
        subtitle: 'Real-time fault detection and severity monitoring',
        tag: '// FAULT DETECTOR',
    },
    explorer: {
        title: 'Block Explorer',
        subtitle: 'Query blocks, transactions, and addresses on the QUAI network',
        tag: '// QUAI CHAIN',
    },
    logs: {
        title: 'System Logs',
        subtitle: 'Live log stream from the go-quai node daemon process',
        tag: '// DAEMON OUTPUT',
    },
    settings: {
        title: 'Configuration',
        subtitle: 'Adjust node, network, and web API settings',
        tag: '// SYSTEM CONFIG',
    },
    setup: {
        title: 'Initial Setup',
        subtitle: 'System audit and initial configuration',
        tag: '// WIZARD',
    },
    installer: {
        title: 'Node Installer',
        subtitle: 'Downloading and verifying go-quai engine',
        tag: '// DOWNLOADER',
    },
};


export default function App() {
    const { fetchMetrics, fetchLogs, viewMode, activeTab, setActiveTab, metrics, installer } = useOrchestratorStore();
    useUpdater();

    // Setup / First run logic
    useEffect(() => {
        const init = async () => {
            if (!isTauriRuntime()) return;
            try {
                const res = await invoke('setup_first_run') as any;
                if (res.is_first_run) setActiveTab('setup');
                else if (!installer.installed) setActiveTab('installer');
            } catch (err) {
                console.error('Setup first run failed:', err);
            }
        };
        init();
    }, [setActiveTab, installer.installed]);



    // Bug fix #4: stabilise action refs so the useEffect below doesn't re-subscribe

    // on every render. Zustand actions are stable by default, but using refs is the
    // idiomatic guard against accidental re-runs.
    const fetchMetricsRef = useRef(fetchMetrics);
    const fetchLogsRef = useRef(fetchLogs);
    fetchMetricsRef.current = fetchMetrics;
    fetchLogsRef.current = fetchLogs;

    // Metric & Log Polling
    useEffect(() => {
        fetchMetricsRef.current();
        if (activeTab === 'logs') fetchLogsRef.current();

        const interval = setInterval(() => {
            fetchMetricsRef.current();
            // Improvement: only poll logs when the user is actually looking at them
            if (activeTab === 'logs') {
                fetchLogsRef.current();
            }
        }, 1000);
        return () => clearInterval(interval);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeTab]); // Trigger re-run if activeTab changes to ensure immediate first fetch

    const cards = [
        { label: 'CPU Utilization', value: `${metrics.cpu}%`, subtext: 'THREAD OPT ACTIVE', icon: Cpu, color: ACCENT_CYAN, data: metrics.history.cpu },
        { label: 'Memory Usage', value: `${metrics.ram}GB`, subtext: '25% BUFFER FREE', icon: Database, color: ACCENT_NEON, data: metrics.history.ram },
        { label: 'Disk I/O', value: `${metrics.disk}MB/s`, subtext: 'READ / WRITE OPS', icon: HardDrive, color: ACCENT_YELLOW, data: metrics.history.disk },
        { label: 'Active Peers', value: metrics.peers, subtext: 'P2P MESH GLOBAL', icon: Users, color: ACCENT_CYAN, data: metrics.history.peers },
        { label: 'RPC Latency', value: `${metrics.latency}ms`, subtext: 'GATEWAY RESPONSE', icon: Wifi, color: ACCENT_NEON, data: metrics.history.latency },
        { label: 'Sync Lag', value: `${metrics.syncLag}s`, subtext: 'CHAIN TIP DELTA', icon: Timer, color: ACCENT_YELLOW, data: metrics.history.syncLag },
        { label: 'Block Height', value: `#${metrics.blockHeight}`, subtext: 'LATEST SETTLED', icon: Layers, color: ACCENT_CYAN, data: metrics.history.blockHeight },
    ];

    const meta = PAGE_META[activeTab] || PAGE_META['dashboard'];

    const renderPage = () => {
        switch (activeTab) {
            case 'nodes': return <NodesPage />;
            case 'clusters': return <ClustersPage />;
            case 'alerts': return <AlertsPage />;
            case 'explorer': return <ExplorerLayout />;
            case 'logs': return <LogsPage />;
            case 'settings': return <SettingsPage />;
            case 'setup': return <SetupWizardPage />;
            case 'installer': return <InstallerPage />;
            default:

                return (
                    <AnimatePresence mode="wait">
                        <motion.div
                            key={viewMode}
                            initial={{ opacity: 0, y: 8 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: -8 }}
                            transition={{ duration: 0.25 }}
                            className={`metric-grid ${viewMode === 'enterprise' ? 'enterprise' : 'operator'}`}
                        >
                            {cards.map((card) => (
                                <MetricCard
                                    key={card.label}
                                    label={card.label}
                                    value={card.value}
                                    subtext={card.subtext}
                                    icon={card.icon}
                                    accentColor={card.color}
                                    data={card.data}
                                />
                            ))}
                        </motion.div>
                    </AnimatePresence>
                );
        }
    };

    // Hide terminal on content-heavy pages
    const hideTerminal = ['logs', 'settings'].includes(activeTab);

    return (
        <div className="app">
            <Sidebar />
            <div className="main">
                <UpdaterBanner />
                <Header />

                <div className={`workspace ${hideTerminal ? 'no-terminal' : ''}`}>
                    <div className="content">
                        {/* Page header */}
                        <AnimatePresence mode="wait">
                            <motion.div
                                key={activeTab}
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                exit={{ opacity: 0 }}
                                transition={{ duration: 0.15 }}
                                className="page-header"
                            >
                                <div className="page-title">
                                    {meta.title}
                                    <span className="page-title-tag">{meta.tag}</span>
                                </div>
                                <div className="page-subtitle">{meta.subtitle}</div>
                            </motion.div>
                        </AnimatePresence>

                        {/* Page content */}
                        <AnimatePresence mode="wait">
                            <motion.div
                                key={activeTab}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                transition={{ duration: 0.2 }}
                                className="flex-1 min-h-0"
                            >
                                {renderPage()}
                            </motion.div>
                        </AnimatePresence>
                    </div>
                    {!hideTerminal && <Terminal />}
                </div>
            </div>
        </div>
    );
}
