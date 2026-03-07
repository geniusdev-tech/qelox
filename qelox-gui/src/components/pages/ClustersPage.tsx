import { motion } from 'framer-motion';

export default function ClustersPage() {
    return (
        <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
        >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span className="page-label">Cluster Registry</span>
                <span className="page-label" style={{ color: 'var(--neon)' }}>1 CLUSTER</span>
            </div>

            <div className="panel">
                <div className="panel-header">
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <div style={{ width: 8, height: 8, borderRadius: '50%', background: 'var(--neon)', boxShadow: '0 0 7px var(--neon)' }} />
                        <span style={{ fontSize: 13, fontWeight: 900, letterSpacing: '-0.02em' }}>CLUSTER-ALPHA</span>
                    </div>
                    <span className="badge badge-neon">ONLINE</span>
                </div>

                <div className="data-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
                    <div className="data-cell">
                        <div className="data-cell-label">Region</div>
                        <div className="data-cell-value cyan">EU-WEST-1A</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Nodes</div>
                        <div className="data-cell-value">1</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Network</div>
                        <div className="data-cell-value cyan">COLOSSEUM</div>
                    </div>
                    <div className="data-cell">
                        <div className="data-cell-label">Health</div>
                        <div className="data-cell-value neon">100%</div>
                    </div>
                </div>

                <div style={{ marginTop: 16 }}>
                    <div className="data-cell-label" style={{ marginBottom: 8 }}>SLICES</div>
                    <div style={{ display: 'flex', gap: 8 }}>
                        <span className="badge badge-cyan">[0 0]</span>
                    </div>
                </div>
            </div>

            {/* Add cluster placeholder */}
            <div className="panel" style={{ border: '1px dashed var(--border)', opacity: 0.4, textAlign: 'center', cursor: 'pointer' }}>
                <span className="page-label">+ ADD CLUSTER</span>
            </div>
        </motion.div>
    );
}
