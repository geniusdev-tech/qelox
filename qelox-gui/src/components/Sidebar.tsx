import { motion } from 'framer-motion';
import { BarChart2, Layers, Activity, AlertTriangle, Terminal, Settings, Boxes, Search } from 'lucide-react';
import { useOrchestratorStore } from '../store';

const NAV_ITEMS = [
    { id: 'dashboard', label: 'Dashboard', icon: BarChart2 },
    { id: 'nodes', label: 'Nodes', icon: Activity },
    { id: 'clusters', label: 'Clusters', icon: Layers },
    { id: 'explorer', label: 'Explorer', icon: Search },
    { id: 'alerts', label: 'Alerts', icon: AlertTriangle },
    { id: 'logs', label: 'Logs', icon: Terminal },
    { id: 'settings', label: 'Settings', icon: Settings },
];

export const Sidebar = () => {
    const { activeTab, setActiveTab } = useOrchestratorStore();

    return (
        <aside className="sidebar">
            {/* Brand */}
            <div className="sidebar-brand">
                <div className="sidebar-brand-logo">
                    <div className="sidebar-brand-icon">
                        <Boxes size={15} />
                    </div>
                    <span className="sidebar-brand-name">QELO-X</span>
                </div>
                <div className="sidebar-brand-sub">Orchestrator v1.1</div>
            </div>

            {/* Nav */}
            <nav className="sidebar-nav">
                {NAV_ITEMS.map(({ id, label, icon: Icon }) => {
                    const isActive = activeTab === id;
                    return (
                        <div
                            key={id}
                            className={`nav-item ${isActive ? 'active' : ''}`}
                            onClick={() => setActiveTab(id)}
                        >
                            {isActive && (
                                <motion.div layoutId="nav-indicator" className="nav-item-indicator" />
                            )}
                            <Icon size={15} />
                            <span>{label}</span>
                        </div>
                    );
                })}
            </nav>

            {/* Footer */}
            <div className="sidebar-footer">
                <div className="sidebar-status">
                    <div className="status-dot" />
                    <span style={{ fontSize: '9px', fontWeight: 700, letterSpacing: '0.15em', textTransform: 'uppercase', color: 'var(--text-muted)' }}>
                        System Online
                    </span>
                </div>
                <div className="sidebar-meta">
                    USER: ROOT · ZONE: EU-WEST-1A
                </div>
            </div>
        </aside>
    );
};
