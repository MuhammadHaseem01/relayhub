import React, { useState, useEffect } from 'react';
import { BarChart3, Clock, ShieldCheck } from 'lucide-react';
import { getTenantUsage, type UsageResponse } from '../api/client';

interface UsagePageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const UsagePage: React.FC<UsagePageProps> = () => {
  const [usage, setUsage] = useState<UsageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchUsage = async () => {
      setLoading(true);
      setError(null);
      const res = await getTenantUsage();
      setLoading(false);

      if (res.success && res.data) {
        setUsage(res.data);
      } else {
        setError(res.error || 'Failed to fetch tenant usage');
      }
    };

    fetchUsage();
  }, []);

  const formatRelativeTime = (isoString?: string): string => {
    if (!isoString) return '24 hours';
    const target = new Date(isoString).getTime();
    const now = new Date().getTime();
    const diffMs = target - now;

    if (diffMs <= 0) return 'shortly';

    const hours = Math.floor(diffMs / (1000 * 60 * 60));
    const mins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));

    if (hours > 0) {
      return `${hours} hour${hours > 1 ? 's' : ''} ${mins} min${mins > 1 ? 's' : ''}`;
    }
    return `${mins} min${mins > 1 ? 's' : ''}`;
  };

  const used = usage?.used || 0;
  const limit = usage?.limit || 100;
  const remaining = usage?.remaining ?? (limit - used);
  const percentage = Math.min(100, Math.round((used / limit) * 100));

  const getProgressBarColor = (pct: number) => {
    if (pct >= 90) return 'var(--danger)';
    if (pct >= 70) return 'var(--warning)';
    return 'var(--accent)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Overview Card */}
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <BarChart3 size={20} style={{ color: 'var(--accent)' }} />
            <h3 style={{ fontSize: '16px', margin: 0 }}>Tenant Quota & Usage</h3>
          </div>

          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            padding: '4px 12px',
            borderRadius: '16px',
            backgroundColor: 'var(--accent-bg)',
            color: 'var(--accent)',
            fontWeight: 600,
            fontSize: '12px',
            border: '1px solid rgba(88, 166, 255, 0.3)',
            textTransform: 'uppercase'
          }}>
            <ShieldCheck size={14} />
            <span>{usage?.plan || 'Free'} Plan</span>
          </div>
        </div>

        {error && (
          <div style={{
            backgroundColor: 'rgba(248, 81, 73, 0.15)',
            border: '1px solid var(--danger)',
            color: 'var(--danger)',
            padding: '10px',
            borderRadius: 'var(--radius-md)',
            fontSize: '13px',
            marginBottom: '16px'
          }}>
            {error}
          </div>
        )}

        {loading ? (
          <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-secondary)' }}>
            Loading usage statistics...
          </div>
        ) : (
          <div>
            {/* Usage Progress Bar */}
            <div style={{ marginBottom: '24px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px', fontSize: '13px' }}>
                <span style={{ color: 'var(--text-secondary)' }}>24-Hour Rolling Quota Usage</span>
                <span style={{ fontWeight: 600, fontFamily: 'var(--font-mono)' }}>
                  {used} / {limit} requests ({percentage}%)
                </span>
              </div>

              <div style={{
                height: '12px',
                width: '100%',
                backgroundColor: 'var(--bg)',
                borderRadius: '6px',
                overflow: 'hidden',
                border: '1px solid var(--border)'
              }}>
                <div style={{
                  height: '100%',
                  width: `${percentage}%`,
                  backgroundColor: getProgressBarColor(percentage),
                  transition: 'width 0.3s ease, background-color 0.3s ease',
                  borderRadius: '6px'
                }} />
              </div>
            </div>

            {/* Metrics Breakdown Grid */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
              <div style={{ backgroundColor: 'var(--bg)', padding: '16px', borderRadius: 'var(--radius-md)', border: '1px solid var(--border)' }}>
                <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Requests Used</div>
                <div style={{ fontSize: '24px', fontWeight: 700, fontFamily: 'var(--font-display)', color: 'var(--text-primary)' }}>
                  {used}
                </div>
              </div>

              <div style={{ backgroundColor: 'var(--bg)', padding: '16px', borderRadius: 'var(--radius-md)', border: '1px solid var(--border)' }}>
                <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Requests Remaining</div>
                <div style={{ fontSize: '24px', fontWeight: 700, fontFamily: 'var(--font-display)', color: 'var(--success)' }}>
                  {remaining}
                </div>
              </div>

              <div style={{ backgroundColor: 'var(--bg)', padding: '16px', borderRadius: 'var(--radius-md)', border: '1px solid var(--border)' }}>
                <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Quota Reset Window</div>
                <div style={{ fontSize: '14px', fontWeight: 600, color: 'var(--accent)', display: 'flex', alignItems: 'center', gap: '6px', marginTop: '6px' }}>
                  <Clock size={16} />
                  <span>Resets in {formatRelativeTime(usage?.resets_at)}</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
