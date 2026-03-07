import { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useOrchestratorStore } from '../../store';

const LEVELS = ['ALL', 'INFO', 'SUCCESS', 'WARN', 'ERROR'] as const;

export default function LogsPage() {
    const { logs } = useOrchestratorStore();
    const [search, setSearch] = useState('');
    const [level, setLevel] = useState<typeof LEVELS[number]>('ALL');
    const [autoScroll, setAutoScroll] = useState(true);
    const bottomRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [logs, autoScroll]);

    const filtered = logs.filter((log) => {
        const matchSearch = !search || log.message.toLowerCase().includes(search.toLowerCase());
        const matchLevel = level === 'ALL' || log.level === level;
        return matchSearch && matchLevel;
    });

    return (
        <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            style={{ display: 'flex', flexDirection: 'column', gap: 12, height: '100%' }}
        >
            {/* Controls */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap', flexShrink: 0 }}>
                <input
                    type="text"
                    className="form-input"
                    style={{ flex: 1, minWidth: 160, padding: '6px 12px' }}
                    placeholder="Filter logs..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
                {LEVELS.map((lvl) => (
                    <button
                        key={lvl}
                        className={`btn ${level === lvl ? 'btn-neon' : 'btn-dim'}`}
                        style={level === lvl ? { background: 'rgba(0,255,136,0.08)' } : undefined}
                        onClick={() => setLevel(lvl)}
                    >
                        {lvl}
                    </button>
                ))}
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 9, fontWeight: 700, letterSpacing: '0.1em', textTransform: 'uppercase', color: 'var(--text-muted)', cursor: 'pointer' }}>
                    <input type="checkbox" checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} style={{ accentColor: 'var(--neon)' }} />
                    AUTO-SCROLL
                </label>
            </div>

            {/* Log Table */}
            <div className="panel" style={{ flex: 1, overflow: 'auto', padding: 0, minHeight: 0 }}>
                <table className="log-table">
                    <thead>
                        <tr>
                            <th style={{ width: 90 }}>TIME</th>
                            <th style={{ width: 80 }}>LEVEL</th>
                            <th>MESSAGE</th>
                        </tr>
                    </thead>
                    <tbody>
                        {filtered.map((log) => (
                            <tr key={log.id}>
                                <td>{log.timestamp}</td>
                                <td className={`lvl-${log.level}`}>{log.level}</td>
                                <td className="log-msg">{log.message}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
                <div ref={bottomRef} />
            </div>

            <div className="page-label">{filtered.length} entries {search || level !== 'ALL' ? `(filtered from ${logs.length} total)` : ''}</div>
        </motion.div>
    );
}
