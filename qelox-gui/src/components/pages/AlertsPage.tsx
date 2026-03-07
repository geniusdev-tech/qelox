import { useState } from 'react';
import { motion } from 'framer-motion';
import { useOrchestratorStore } from '../../store';

const FILTERS = ['ALL', 'CRITICAL', 'WARNING'] as const;

export default function AlertsPage() {
    const { logs } = useOrchestratorStore();
    const [filter, setFilter] = useState<typeof FILTERS[number]>('ALL');

    const alerts = logs.filter((log) => {
        if (filter === 'CRITICAL') return log.level === 'ERROR';
        if (filter === 'WARNING') return log.level === 'WARN';
        return log.level === 'ERROR' || log.level === 'WARN';
    });

    return (
        <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
        >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span className="page-label">Active Alerts</span>
                <div style={{ display: 'flex', gap: 6 }}>
                    {FILTERS.map((f) => (
                        <button
                            key={f}
                            className={`btn ${filter === f ? 'btn-neon' : 'btn-dim'}`}
                            style={filter === f ? { background: 'rgba(0,255,136,0.08)' } : undefined}
                            onClick={() => setFilter(f)}
                        >
                            {f}
                        </button>
                    ))}
                </div>
            </div>

            {alerts.length === 0 ? (
                <div className="panel" style={{ textAlign: 'center', padding: 48, opacity: 0.4 }}>
                    <div style={{ fontSize: 24, marginBottom: 8 }}>✓</div>
                    <span className="page-label">No active alerts</span>
                </div>
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                    {alerts.map((alert) => (
                        <div
                            key={alert.id}
                            className="panel"
                            style={{
                                padding: '10px 14px',
                                borderLeft: `3px solid ${alert.level === 'ERROR' ? 'var(--red)' : 'var(--yellow)'}`,
                                display: 'flex',
                                alignItems: 'flex-start',
                                gap: 12,
                            }}
                        >
                            <div style={{ display: 'flex', flexDirection: 'column', gap: 3, minWidth: 0, flex: 1 }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                    <span className={`badge ${alert.level === 'ERROR' ? 'badge-red' : 'badge-yellow'}`}>
                                        {alert.level === 'ERROR' ? 'CRITICAL' : 'WARNING'}
                                    </span>
                                    <span className="page-label">{alert.timestamp}</span>
                                </div>
                                <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text)' }}>{alert.message}</span>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </motion.div>
    );
}
