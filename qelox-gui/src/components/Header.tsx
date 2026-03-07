import { motion } from 'framer-motion';
import { Server, Globe, Layers } from 'lucide-react';
import { useOrchestratorStore } from '../store';

const HealthScore = ({ score, status }: { score: number; status: string }) => {
    const color = score > 90 ? 'var(--neon)' : score > 70 ? 'var(--yellow)' : 'var(--red)';
    const circumference = 2 * Math.PI * 15;
    const offset = circumference - (circumference * score) / 100;

    // Map status string to a human-readable label
    const statusLabel = status === 'RUNNING' ? 'Operational' : status === 'DEGRADED' ? 'Degraded' : 'Offline';

    return (
        <div className="health-widget">
            <div className="health-ring">
                <svg width="38" height="38" viewBox="0 0 38 38" style={{ transform: 'rotate(-90deg)' }}>
                    <circle cx="19" cy="19" r="15" fill="none" stroke="#1a2030" strokeWidth="3" />
                    <motion.circle
                        cx="19" cy="19" r="15"
                        fill="none"
                        stroke={color}
                        strokeWidth="3"
                        strokeLinecap="round"
                        strokeDasharray={circumference}
                        initial={{ strokeDashoffset: circumference }}
                        animate={{ strokeDashoffset: offset }}
                        transition={{ duration: 1, ease: 'easeOut' }}
                    />
                </svg>
                <div className="health-ring-label" style={{ color }}>
                    {score}
                </div>
            </div>
            <div>
                <div className="health-info-label">Health Score</div>
                <div className="health-info-value" style={{ color }}>{statusLabel}</div>
            </div>
        </div>
    );
};

export const Header = () => {
    const { viewMode, toggleViewMode, status, healthScore, nodeName, network, metrics } = useOrchestratorStore();

    return (
        <div className="topbar">
            <div className="topbar-left">
                <div className="topbar-info">
                    <div className="topbar-info-icon">
                        <Server size={15} style={{ color: 'var(--cyan)' }} />
                    </div>
                    <div>
                        <div className="topbar-info-label">Active Node</div>
                        <div className="topbar-info-value">{nodeName || 'CONNECTING...'}</div>
                    </div>
                </div>

                <div className="topbar-info">
                    <div className="topbar-info-icon">
                        <Globe size={15} style={{ color: 'var(--yellow)' }} />
                    </div>
                    <div>
                        <div className="topbar-info-label">Network</div>
                        <div className="topbar-info-value">{network || 'CONNECTING...'}</div>
                    </div>
                </div>

                <div className="topbar-info">
                    <div className="topbar-info-icon">
                        <Layers size={13} style={{ color: 'var(--cyan)' }} />
                    </div>
                    <div>
                        <div className="topbar-info-label">Block Height</div>
                        <div className="topbar-info-value neon">#{metrics.blockHeight}</div>
                    </div>
                </div>

                <div className={`status-badge ${status === 'RUNNING' ? 'running' : 'degraded'}`}>
                    {status}
                </div>
            </div>

            <div className="topbar-right">
                <HealthScore score={healthScore} status={status} />
                <div className="mode-toggle">
                    <button
                        className={`mode-btn ${viewMode === 'enterprise' ? 'active-enterprise' : ''}`}
                        onClick={() => viewMode !== 'enterprise' && toggleViewMode()}
                    >
                        Enterprise
                    </button>
                    <button
                        className={`mode-btn ${viewMode === 'operator' ? 'active-operator' : ''}`}
                        onClick={() => viewMode !== 'operator' && toggleViewMode()}
                    >
                        Operator
                    </button>
                </div>
            </div>
        </div>
    );
};
