import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { useOrchestratorStore } from '../../store';

const getEnvironmentArg = (extraArgs: string[] = []) =>
    extraArgs.find((arg) => arg.startsWith('--node.environment='))?.split('=')[1] || 'colosseum';

const replaceEnvironmentArg = (extraArgs: string[] = [], network: string) => {
    const filtered = extraArgs.filter((arg) => !arg.startsWith('--node.environment='));
    filtered.push(`--node.environment=${network}`);
    return filtered;
};

export default function SettingsPage() {
    const { currentConfig, fetchConfig, saveConfig } = useOrchestratorStore();

    const [rpcUrl, setRpcUrl] = useState('');
    const [apiHost, setApiHost] = useState('');
    const [apiPort, setApiPort] = useState('');
    const [network, setNetwork] = useState('');
    const [autoStart, setAutoStart] = useState(true);
    const [maxRestarts, setMaxRestarts] = useState(0);
    const [saved, setSaved] = useState(false);

    // Initialize from currentConfig
    useEffect(() => {
        if (!currentConfig) {
            fetchConfig();
            return;
        }
        setRpcUrl(currentConfig.monitor?.rpc_url || '');
        setApiHost(currentConfig.web?.bind || '');
        setApiPort(String(currentConfig.web?.port || ''));
        setNetwork(getEnvironmentArg(currentConfig.node?.extra_args));
        setAutoStart(currentConfig.node?.auto_start ?? true);
        setMaxRestarts(currentConfig.daemon?.max_restarts || 0);
    }, [currentConfig, fetchConfig]);

    const handleSave = async () => {
        const updated = {
            ...currentConfig,
            monitor: { ...currentConfig.monitor, rpc_url: rpcUrl },
            web: { ...currentConfig.web, bind: apiHost, port: parseInt(apiPort) },
            daemon: { ...currentConfig.daemon, max_restarts: parseInt(String(maxRestarts)) },
            node: {
                ...currentConfig.node,
                auto_start: autoStart,
                extra_args: replaceEnvironmentArg(currentConfig.node?.extra_args, network)
            }
        };

        const ok = await saveConfig(updated);
        if (ok) {
            setSaved(true);
            setTimeout(() => setSaved(false), 2500);
        }
    };

    return (
        <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            style={{ display: 'flex', flexDirection: 'column', gap: 14, maxWidth: 640 }}
        >
            {/* Node / RPC */}
            <div className="panel">
                <div className="panel-section-label">NODE / RPC</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    <div className="form-field">
                        <label className="form-label">RPC URL</label>
                        <input className="form-input" value={rpcUrl} onChange={(e) => setRpcUrl(e.target.value)} />
                    </div>
                    <div className="form-field">
                        <label className="form-label">Network</label>
                        <select
                            className="form-input"
                            value={network}
                            onChange={(e) => setNetwork(e.target.value)}
                        >
                            <option value="colosseum">Colosseum</option>
                            <option value="orchard">Orchard</option>
                            <option value="garden">Garden</option>
                            <option value="local">Local Dev</option>
                        </select>
                    </div>
                    <div className="form-field">
                        <label className="form-label">Max Restarts (0 = unlimited)</label>
                        <input className="form-input" type="number" value={maxRestarts} onChange={(e) => setMaxRestarts(parseInt(e.target.value) || 0)} />
                    </div>
                    <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
                        <input type="checkbox" checked={autoStart} onChange={(e) => setAutoStart(e.target.checked)} style={{ accentColor: 'var(--neon)', width: 14, height: 14 }} />
                        <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--text)' }}>Auto-start node on daemon launch</span>
                    </label>
                </div>
            </div>

            {/* Web API */}
            <div className="panel">
                <div className="panel-section-label">WEB API</div>
                <div className="form-grid-2">
                    <div className="form-field">
                        <label className="form-label">Bind Address</label>
                        <input className="form-input" value={apiHost} onChange={(e) => setApiHost(e.target.value)} />
                    </div>
                    <div className="form-field">
                        <label className="form-label">Port</label>
                        <input className="form-input" type="number" value={apiPort} onChange={(e) => setApiPort(e.target.value)} />
                    </div>
                </div>
            </div>

            {/* Actions */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
                <button className="btn btn-neon" onClick={handleSave}>SAVE CONFIG</button>
                {saved && (
                    <motion.span
                        initial={{ opacity: 0, x: -8 }}
                        animate={{ opacity: 1, x: 0 }}
                        style={{ fontSize: 10, fontWeight: 700, color: 'var(--neon)', letterSpacing: '0.1em' }}
                    >
                        ✓ SAVED
                    </motion.span>
                )}
            </div>
        </motion.div>
    );
}
