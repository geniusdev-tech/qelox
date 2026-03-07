import { motion } from 'framer-motion';
import { Download, CheckCircle, AlertCircle, Terminal as TerminalIcon } from 'lucide-react';
import { useOrchestratorStore } from '../../store';

export default function InstallerPage() {
    const { installer, runInstall, setActiveTab, startNode } = useOrchestratorStore();

    const handleFinish = () => {
        startNode();
        setActiveTab('dashboard');
    };

    return (
        <div className="flex flex-col items-center justify-center min-h-[400px] p-8 text-center">
            <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                className="max-w-xl w-full bg-white/5 border border-white/10 p-10 rounded-lg"
            >
                {!installer.installing && !installer.installed && (
                    <>
                        <div className="w-16 h-16 bg-cyan-500/20 text-cyan-400 rounded-full flex items-center justify-center mx-auto mb-6">
                            <Download size={32} />
                        </div>
                        <h2 className="text-2xl font-black italic tracking-tighter uppercase mb-4">Engine Download Required</h2>
                        <p className="text-white/60 mb-8">
                            To run a node, QELO-X needs the latest <span className="text-white font-mono">go-quai</span> binary. We will download and configure it for you automatically.
                        </p>
                        <button
                            onClick={runInstall}
                            className="bg-cyan-500 text-black px-10 py-3 rounded-sm font-black uppercase italic tracking-tighter hover:bg-cyan-400"
                        >
                            Start Installation
                        </button>
                    </>
                )}

                {installer.installing && (
                    <>
                        <h2 className="text-xl font-bold uppercase tracking-widest mb-8">Installing Node Engine...</h2>
                        <div className="w-full bg-white/10 h-1.5 rounded-full overflow-hidden mb-4">
                            <motion.div
                                className="bg-cyan-500 h-full"
                                initial={{ width: 0 }}
                                animate={{ width: `${installer.progress}%` }}
                            />
                        </div>
                        <div className="flex justify-between items-center text-[10px] font-mono mb-10">
                            <span className="text-white/40 uppercase uppercase tracking-widest">{installer.status}</span>
                            <span className="text-cyan-400">{installer.progress}%</span>
                        </div>

                        <div className="bg-black/40 p-4 rounded border border-white/5 text-left font-mono text-[10px] text-cyan-400/80 mb-6 flex gap-3">
                            <TerminalIcon size={14} className="shrink-0" />
                            <div className="truncate">GET https://github.com/dominant-strategies/go-quai/releases/latest ...</div>
                        </div>
                    </>
                )}

                {installer.installed && !installer.installing && (
                    <>
                        <div className="w-16 h-16 bg-neon-green/20 text-neon-green rounded-full flex items-center justify-center mx-auto mb-6" style={{ color: '#00ff88', backgroundColor: 'rgba(0, 255, 136, 0.2)' }}>
                            <CheckCircle size={32} />
                        </div>
                        <h2 className="text-2xl font-black italic tracking-tighter uppercase mb-4">Installation Successful</h2>
                        <p className="text-white/60 mb-8">
                            The <span className="text-white font-mono">go-quai</span> engine has been installed and verified. Your node is ready to be initialized.
                        </p>
                        <button
                            onClick={handleFinish}
                            className="bg-white text-black px-10 py-3 rounded-sm font-black uppercase italic tracking-tighter hover:bg-cyan-400"
                        >
                            Go to Dashboard
                        </button>
                    </>
                )}

                {installer.status.startsWith('Error') && (
                    <div className="mt-6 p-4 bg-red-500/10 border border-red-500/30 rounded flex items-center gap-3 text-red-400 text-xs">
                        <AlertCircle size={16} />
                        <span>{installer.status}</span>
                    </div>
                )}
            </motion.div>
        </div>
    );
}
