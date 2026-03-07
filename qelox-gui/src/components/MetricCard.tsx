import { ResponsiveContainer, AreaChart, Area } from 'recharts';
import { LucideIcon } from 'lucide-react';

interface MetricCardProps {
    label: string;
    value: string | number;
    subtext: string;
    icon: LucideIcon;
    accentColor: string;
    data: { time: string; value: number }[];
}

export const MetricCard = ({ label, value, subtext, icon: Icon, accentColor, data }: MetricCardProps) => {
    return (
        <div className="metric-card">
            <div className="metric-card-header">
                <div className="metric-card-icon" style={{ color: accentColor }}>
                    <Icon size={16} />
                </div>
                <div>
                    <div className="metric-card-label">{label}</div>
                    <div className="metric-card-value">{value}</div>
                </div>
            </div>

            <div className="metric-card-footer">
                <div className="sparkline-wrap">
                    <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
                            <defs>
                                <linearGradient id={`g-${label}`} x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="0%" stopColor={accentColor} stopOpacity={0.12} />
                                    <stop offset="100%" stopColor={accentColor} stopOpacity={0} />
                                </linearGradient>
                            </defs>
                            <Area
                                type="monotone"
                                dataKey="value"
                                stroke={accentColor}
                                strokeWidth={1.5}
                                fill={`url(#g-${label})`}
                                isAnimationActive={false}
                                dot={false}
                            />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>
                <div className="metric-card-subtext">{subtext}</div>
            </div>
        </div>
    );
};
