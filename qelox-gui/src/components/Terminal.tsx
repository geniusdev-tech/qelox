import { useRef, useEffect } from 'react';
import { Terminal as TerminalIcon, ChevronRight } from 'lucide-react';
import { useOrchestratorStore } from '../store';

export const Terminal = () => {
    const { logs } = useOrchestratorStore();
    const scrollRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [logs]);

    return (
        <aside className="terminal-panel">
            <div className="terminal-topbar">
                <div className="terminal-topbar-title">
                    <TerminalIcon size={13} style={{ color: 'var(--neon)' }} />
                    Node Terminal
                </div>
                <div className="terminal-dots">
                    <div className="terminal-dot red" />
                    <div className="terminal-dot yellow" />
                    <div className="terminal-dot green" />
                </div>
            </div>

            <div ref={scrollRef} className="terminal-body">
                {logs.map((log) => (
                    <div key={log.id} className="log-row">
                        <span className="log-time">[{log.timestamp}]</span>
                        <span className={`log-level ${log.level}`}>{log.level}</span>
                        <span className="log-message">{log.message}</span>
                    </div>
                ))}
                <div className="terminal-cursor">
                    <ChevronRight size={12} />
                    <span className="cursor-blink">▍</span>
                </div>
            </div>

            <div className="terminal-input-area">
                <div className="terminal-input-wrap">
                    <div className="terminal-prompt">
                        <ChevronRight size={12} />
                    </div>
                    <input
                        type="text"
                        placeholder="EXECUTE COMMAND..."
                        className="terminal-input"
                    />
                </div>
            </div>
        </aside>
    );
};
