import { useEffect } from 'react';
import { motion } from 'framer-motion';
import { Cpu, Database, Monitor, CheckCircle, ChevronRight, AlertCircle } from 'lucide-react';
import { useOrchestratorStore } from '../../store';

export default function SetupWizardPage() {
    const { hardware, detectHardware, installer, checkInstall, startNode, setActiveTab } = useOrchestratorStore();

    useEffect(() => {
        detectHardware();
        checkInstall();
    }, [detectHardware, checkInstall]);

    const handleContinue = () => {
        if (!installer.installed) {
            setActiveTab('installer');
        } else {
            startNode();
            setActiveTab('dashboard');
        }
    };

    return (
        <div className="flex flex-col items-center justify-center min-h-[400px] p-8 text-center">
            <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                className="max-w-2xl w-full bg-white/5 border border-white/10 p-10 rounded-lg"
                style={{ backdropFilter: 'blur(20px)' }}
            >
                <h1 className="text-3xl font-black italic tracking-tighter mb-2 uppercase">Welcome to QELO-X</h1>
                <p className="text-white/60 mb-8">System audit complete. We've detected your hardware specifications to optimize your node performance.</p>

                <div className="grid grid-cols-2 gap-4 mb-10 text-left">
                    <div className="bg-white/5 p-4 border border-white/10 rounded">
                        <div className="flex items-center gap-3 text-cyan-400 mb-2">
                            <Cpu size={20} />
                            <span className="text-xs font-bold uppercase tracking-widest">Processor</span>
                        </div>
                        <div className="text-sm font-bold truncate">{hardware?.cpu_model || 'Detecting...'}</div>
                        <div className="text-[10px] text-white/40">{hardware?.cpu_cores} Logical Cores Optimized</div>
                    </div>

                    <div className="bg-white/5 p-4 border border-white/10 rounded">
                        <div className="flex items-center gap-3 text-neon-green mb-2" style={{ color: '#00ff88' }}>
                            <Database size={20} />
                            <span className="text-xs font-bold uppercase tracking-widest">Memory</span>
                        </div>
                        <div className="text-sm font-bold">{hardware?.total_ram_gb} GB Total available</div>
                        <div className="text-[10px] text-white/40">GOMEMLIMIT set to {Math.floor(hardware?.total_ram_gb || 0 * 0.75)}GB</div>
                    </div>

                    <div className="bg-white/5 p-4 border border-white/10 rounded">
                        <div className="flex items-center gap-3 text-yellow-500 mb-2">
                            <Monitor size={20} />
                            <span className="text-xs font-bold uppercase tracking-widest">OS Environment</span>
                        </div>
                        <div className="text-sm font-bold">{hardware?.os_name} {hardware?.os_version}</div>
                        <div className="text-[10px] text-white/40">Cloud-Native Runtime</div>
                    </div>

                    <div className="bg-white/5 p-4 border border-white/10 rounded">
                        <div className="flex items-center gap-3 text-white mb-2">
                            {installer.installed ? <CheckCircle size={20} className="text-neon-green" style={{ color: '#00ff88' }} /> : <AlertCircle size={20} className="text-yellow-500" />}
                            <span className="text-xs font-bold uppercase tracking-widest">Node Engine</span>
                        </div>
                        <div className="text-sm font-bold">{installer.installed ? 'go-quai Detected' : 'go-quai Not Found'}</div>
                        <div className="text-[10px] text-white/40">{installer.installed ? 'Version: Latest stable' : 'Manual installation required'}</div>
                    </div>
                </div>

                <button
                    onClick={handleContinue}
                    className="group flex items-center gap-3 bg-white text-black px-8 py-3 rounded-sm font-black uppercase italic tracking-tighter hover:bg-cyan-400 transition-all"
                >
                    {installer.installed ? 'Initialize Node Control' : 'Begin Node Installation'}
                    <ChevronRight size={20} className="group-hover:translate-x-1 transition-transform" />
                </button>
            </motion.div>
        </div>
    );
}
