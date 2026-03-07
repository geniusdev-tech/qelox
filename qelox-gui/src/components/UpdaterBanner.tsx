import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, Download, X } from 'lucide-react';
import { useOrchestratorStore } from '../store';

export function UpdaterBanner() {
    const { update } = useOrchestratorStore();

    if (!update.available) return null;

    return (
        <AnimatePresence>
            <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                className="bg-cyan-500/10 border-b border-cyan-500/30 px-6 py-2 flex items-center justify-between text-sm"
                style={{ backdropFilter: 'blur(8px)' }}
            >
                <div className="flex items-center gap-3 text-cyan-400">
                    <AlertTriangle size={16} />
                    <span>A new version of QELO-X is available: <strong>v{update.version}</strong></span>
                </div>
                <div className="flex items-center gap-4">
                    <button
                        className="flex items-center gap-2 bg-cyan-500 text-black px-3 py-1 rounded-sm font-bold hover:bg-cyan-400 transition-colors uppercase tracking-tighter text-xs"
                        onClick={() => {
                            // In a real app, this would trigger the actual update process
                            window.open('https://github.com/geniusdev-tech/qelox/releases/latest', '_blank');
                        }}
                    >
                        <Download size={14} />
                        Update Now
                    </button>
                    <button className="text-white/40 hover:text-white">
                        <X size={16} />
                    </button>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}
