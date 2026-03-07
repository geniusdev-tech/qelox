import { motion } from 'framer-motion';
import { useOrchestratorStore } from '../../store';

export default function NodesPage() {
    const { metrics, status, network, nodeName, startNode, stopNode, restartNode, setActiveTab } = useOrchestratorStore();

    const statusClass = status === 'RUNNING' ? 'badge-neon' : status === 'DEGRADED' ? 'badge-yellow' : 'badge-red';
    const dotColor = status === 'RUNNING' ? 'var(--neon)' : status === 'DEGRADED' ? 'var(--yellow)' : 'var(--red)';

    const isRunning = status === 'RUNNING';

    return (
        <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
        >
            {/* Page count */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span className="page-label">Registered Nodes</span>
                <span className="page-label" style={{ color: 'var(--neon)' }}>{isRunning ? '1 ACTIVE' : '0 ACTIVE'}</span>
            </div>

            {/* Node Card */}
            <div className="panel">
                {/* Node Header */}
                <div className="panel-header">
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <div style={{ width: 8, height: 8, borderRadius: '50%', background: dotColor, boxShadow: `0 0 7px ${dotColor}` }} />
                        <span style={{ fontSize: 13, fontWeight: 900, letterSpacing: '-0.02em' }}>{nodeName || 'QN-ARC-9201'}</span>
                    </div>
                    <span className={`badge ${statusClass}`}>{status}</span>
                </div>

                {/* Metrics grid */}
                <div className="data-grid">
                    <div className="data-cell">
                        <div className="data-cell-label">Network</div>
                        <div className="data-cell-value cyan">{(network || 'COLOSSEUM').toUpperCase()}</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Slice</div>
                        <div className="data-cell-value">{nodeName || '[0 0]'}</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Block Height</div>
                        <div className="data-cell-value neon">#{metrics.blockHeight}</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Peers</div>
                        <div className="data-cell-value neon">{metrics.peers === -1 ? 'N/A' : metrics.peers}</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">CPU</div>
                        <div className="data-cell-value cyan">{metrics.cpu}%</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">RAM</div>
                        <div className="data-cell-value cyan">{metrics.ram} GB</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Disk I/O</div>
                        <div className="data-cell-value yellow">{metrics.disk} MB/s</div>
                    </div>
                </div>

                {/* Actions */}
                <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
                    {isRunning ? (
                        <button
                            className="btn btn-red"
                            style={{ borderColor: 'rgba(255,77,77,0.4)', color: 'var(--red)' }}
                            onClick={stopNode}
                        >
                            STOP NODE
                        </button>
                    ) : (
                        <button
                            className="btn btn-neon"
                            onClick={startNode}
                        >
                            START NODE
                        </button>
                    )}
                    <button className="btn btn-yellow" onClick={restartNode} disabled={!isRunning}>RESTART</button>
                    <button className="btn btn-dim" onClick={() => setActiveTab('logs')}>VIEW LOGS</button>
                </div>
            </div>
        </motion.div>
    );
}

