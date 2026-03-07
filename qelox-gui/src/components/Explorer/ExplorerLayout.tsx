import React, { useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { useOrchestratorStore } from '../../store';

export default function ExplorerLayout() {
    const { explorer, searchExplorer } = useOrchestratorStore();
    const [input, setInput] = useState('');

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault();
        searchExplorer(input.trim());
    };

    return (
        <div className="explorer-wrap">
            {/* Search bar */}
            <form className="explorer-search-bar" onSubmit={handleSearch}>
                <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" style={{ color: 'var(--text-muted)', flexShrink: 0 }}>
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input
                    className="explorer-input"
                    placeholder="Search block number, tx hash (0x…66 chars) or address (0x…42 chars)..."
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    autoFocus
                />
                <button type="submit" className="btn btn-dim" style={{ flexShrink: 0 }}>
                    SEARCH
                </button>
            </form>

            {/* Error */}
            {explorer.error && (
                <div className="panel" style={{ borderLeft: '3px solid var(--red)', fontSize: 11, color: 'var(--red)' }}>
                    {explorer.error}
                </div>
            )}

            {/* Loading */}
            {explorer.loading && (
                <div className="panel" style={{ textAlign: 'center', padding: 48, fontSize: 11, color: 'var(--cyan)' }}>
                    QUERYING BLOCKCHAIN...
                </div>
            )}

            {/* Result */}
            {!explorer.loading && explorer.data && (
                <AnimatePresence mode="wait">
                    <motion.div
                        key={explorer.view}
                        className="explorer-result"
                        initial={{ opacity: 0, y: 6 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.18 }}
                    >
                        {explorer.view === 'block' && <BlockView data={explorer.data} onSearch={(q) => { setInput(q); searchExplorer(q); }} />}
                        {explorer.view === 'tx' && <TransactionView data={explorer.data} />}
                        {explorer.view === 'address' && <AddressView data={explorer.data} />}
                    </motion.div>
                </AnimatePresence>
            )}

            {/* Initial state */}
            {!explorer.loading && !explorer.error && !explorer.data && (
                <div className="explorer-placeholder">
                    <svg width="40" height="40" fill="none" viewBox="0 0 24 24" stroke="currentColor" style={{ color: 'var(--text-muted)' }}>
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
                    </svg>
                    <span style={{ fontSize: 10, letterSpacing: '0.1em', textTransform: 'uppercase' }}>
                        Enter a block number, transaction hash, or address
                    </span>
                </div>
            )}
        </div>
    );
}

// ── Sub-views ──────────────────────────────────────────────────────────────────

function Row({ label, value, color }: { label: string; value: any; color?: string }) {
    return (
        <div style={{ display: 'flex', gap: 12, padding: '8px 0', borderBottom: '1px solid var(--border)', alignItems: 'flex-start' }}>
            <span style={{ fontSize: 9, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', color: 'var(--text-muted)', minWidth: 140, flexShrink: 0 }}>{label}</span>
            <span style={{ fontSize: 11, fontWeight: 700, color: color || 'var(--text)', wordBreak: 'break-all', fontFamily: 'var(--font)' }}>{String(value ?? '—')}</span>
        </div>
    );
}

function CopyHash({ hash, onSearch }: { hash: string; onSearch?: (q: string) => void }) {
    const [copied, setCopied] = useState(false);
    const copy = (e: React.MouseEvent) => {
        e.stopPropagation();
        navigator.clipboard.writeText(hash);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
    };
    return (
        <div
            style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: onSearch ? 'pointer' : 'default' }}
            onClick={() => onSearch && onSearch(hash)}
        >
            <span style={{
                fontFamily: 'var(--font)',
                fontSize: 10,
                color: onSearch ? 'var(--cyan)' : 'var(--text-muted)',
                wordBreak: 'break-all',
                textDecoration: onSearch ? 'underline' : 'none'
            }}>
                {hash}
            </span>
            <button onClick={copy} className="btn btn-dim" style={{ padding: '2px 8px', fontSize: 8, flexShrink: 0 }}>
                {copied ? '✓' : 'COPY'}
            </button>
        </div>
    );
}

function BlockView({ data, onSearch }: { data: any; onSearch: (q: string) => void }) {
    const [rawOpen, setRawOpen] = useState(false);
    const [txOpen, setTxOpen] = useState(false);
    const txs: any[] = data?.transactions || [];

    const toTimestamp = (hex?: string) => {
        if (!hex) return '—';
        const ts = parseInt(hex, 16);
        return new Date(ts * 1000).toLocaleString();
    };

    return (
        <div className="panel" style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
            <div className="panel-header" style={{ marginBottom: 12 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <span className="badge badge-cyan">BLOCK</span>
                    <span style={{ fontSize: 18, fontWeight: 900, color: 'var(--text)', letterSpacing: '-0.03em' }}>
                        #{data?.number ? parseInt(data.number, 16).toLocaleString() : '—'}
                    </span>
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                    <button className="btn btn-dim" onClick={() => setRawOpen(!rawOpen)}>RAW JSON</button>
                </div>
            </div>

            <Row label="Hash" value={<CopyHash hash={data?.hash || '—'} />} />
            <Row label="Parent Hash" value={<CopyHash hash={data?.parentHash || '—'} onSearch={onSearch} />} />
            <Row label="Timestamp" value={toTimestamp(data?.timestamp)} />
            <Row label="Miner" value={<CopyHash hash={data?.miner || '—'} onSearch={onSearch} />} />
            <Row label="Gas Used" value={data?.gasUsed ? parseInt(data.gasUsed, 16).toLocaleString() : '—'} />
            <Row label="Gas Limit" value={data?.gasLimit ? parseInt(data.gasLimit, 16).toLocaleString() : '—'} />
            <Row label="Transactions" value={txs.length} color="var(--neon)" />

            {txs.length > 0 && (
                <div style={{ marginTop: 16 }}>
                    <button
                        className="btn btn-dim"
                        style={{ marginBottom: 8 }}
                        onClick={() => setTxOpen(!txOpen)}
                    >
                        {txOpen ? 'HIDE' : 'VIEW'} TRANSACTIONS ({txs.length})
                    </button>
                    {txOpen && (
                        <div style={{ maxHeight: 300, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
                            {txs.map((tx: any, i: number) => {
                                const txHash = typeof tx === 'string' ? tx : tx.hash;
                                return (
                                    <div key={i} className="data-cell" style={{ fontSize: 10, cursor: txHash ? 'pointer' : 'default' }}
                                        onClick={() => txHash && onSearch(txHash)}
                                        title={txHash ? 'Click to explore this transaction' : ''}
                                    >
                                        <span style={{ color: 'var(--cyan)', fontFamily: 'var(--font)', wordBreak: 'break-all' }}>
                                            {txHash || JSON.stringify(tx)}
                                        </span>
                                        {txHash && <span style={{ color: 'var(--text-muted)', marginLeft: 6 }}>→</span>}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            )}

            {rawOpen && (
                <pre style={{ marginTop: 16, background: 'var(--bg)', padding: 14, borderRadius: 'var(--radius-sm)', fontSize: 9, color: 'var(--text-muted)', overflowX: 'auto', maxHeight: 320, overflowY: 'auto', lineHeight: 1.6 }}>
                    {JSON.stringify(data, null, 2)}
                </pre>
            )}
        </div>
    );
}

function TransactionView({ data }: { data: any }) {
    const { searchExplorer } = useOrchestratorStore();
    const [inputOpen, setInputOpen] = useState(false);
    const receipt = data?.receipt || {};
    const isSuccess = receipt?.status === '0x1';

    return (
        <div className="panel" style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
            <div className="panel-header" style={{ marginBottom: 12 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <span className="badge badge-cyan">TRANSACTION</span>
                    <span className={`badge ${isSuccess ? 'badge-neon' : 'badge-red'}`}>
                        {isSuccess ? 'SUCCESS' : 'FAILED'}
                    </span>
                </div>
            </div>

            <Row label="Hash" value={<CopyHash hash={data?.hash || '—'} />} />
            <Row label="Block Number" value={data?.blockNumber ? parseInt(data.blockNumber, 16).toLocaleString() : '—'} color="var(--cyan)" />
            <Row label="From" value={<CopyHash hash={data?.from || '—'} onSearch={searchExplorer} />} />
            <Row label="To" value={<CopyHash hash={data?.to || '—'} onSearch={searchExplorer} />} />
            {/* Bug fix #5: use BigInt to avoid Number precision overflow for large QUAI amounts. */}
            <Row label="Value" value={data?.value ? `${(Number(BigInt(data.value)) / 1e18).toFixed(6)} QUAI` : '0 QUAI'} color="var(--neon)" />
            <Row label="Gas Price" value={data?.gasPrice ? `${parseInt(data.gasPrice, 16).toLocaleString()} wei` : '—'} />
            <Row label="Gas Used" value={receipt?.gasUsed ? parseInt(receipt.gasUsed, 16).toLocaleString() : '—'} />
            <Row label="Nonce" value={data?.nonce ? parseInt(data.nonce, 16) : '—'} />

            {data?.input && data.input !== '0x' && (
                <div style={{ marginTop: 14 }}>
                    <button className="btn btn-dim" onClick={() => setInputOpen(!inputOpen)} style={{ marginBottom: 8 }}>
                        {inputOpen ? 'HIDE' : 'VIEW'} INPUT DATA
                    </button>
                    {inputOpen && (
                        <pre style={{ background: 'var(--bg)', padding: 12, borderRadius: 'var(--radius-sm)', fontSize: 9, color: 'var(--text-muted)', wordBreak: 'break-all', whiteSpace: 'pre-wrap' }}>
                            {data.input}
                        </pre>
                    )}
                </div>
            )}
        </div>
    );
}

function AddressView({ data }: { data: any }) {
    // Bug fix #6: use BigInt to avoid Number precision overflow for large QUAI balances.
    const balanceQuai = data?.balance
        ? (Number(BigInt(data.balance)) / 1e18).toFixed(6)
        : '0.000000';
    const nonce = data?.nonce ? parseInt(data.nonce, 16) : 0;

    return (
        <div className="panel" style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
            <div className="panel-header" style={{ marginBottom: 12 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <span className="badge badge-cyan">ADDRESS</span>
                    <span style={{ fontSize: 11, fontFamily: 'var(--font)', color: 'var(--cyan)', wordBreak: 'break-all' }}>
                        {data?.address}
                    </span>
                </div>
            </div>

            <div className="data-grid" style={{ marginBottom: 16 }}>
                <div className="data-cell">
                    <div className="data-cell-label">Balance</div>
                    <div className="data-cell-value neon">{balanceQuai} QUAI</div>
                </div>
                <div className="data-cell">
                    <div className="data-cell-label">Nonce</div>
                    <div className="data-cell-value cyan">{nonce}</div>
                </div>
                <div className="data-cell">
                    <div className="data-cell-label">Network</div>
                    <div className="data-cell-value">QUAI</div>
                </div>
            </div>

            <Row label="Full Address" value={<CopyHash hash={data?.address || '—'} />} />
        </div>
    );
}
