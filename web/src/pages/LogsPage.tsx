import React, { useState, useEffect } from 'react';
import { getLogs, type NotificationLog } from '../api/client';

interface LogsPageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const LogsPage: React.FC<LogsPageProps> = ({ onShowToast }) => {
  const [logs, setLogs] = useState<NotificationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [limit, setLimit] = useState(50);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [countdown, setCountdown] = useState(5);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const fetchLogs = async (showLoading = false) => {
    if (showLoading) setLoading(true);
    setError(null);
    const res = await getLogs(limit);
    setLoading(false);

    if (res.success && res.data) {
      setLogs(res.data.logs || []);
    } else {
      setError(res.error || 'Failed to load logs');
    }
  };

  // Initial fetch + limit change fetch
  useEffect(() => {
    fetchLogs(true);
  }, [limit]);

  // Auto Refresh Polling (5s)
  useEffect(() => {
    if (!autoRefresh) return;

    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          fetchLogs(false);
          return 5;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [autoRefresh, limit]);

  const handleCopyRequestId = (reqId: string) => {
    navigator.clipboard.writeText(reqId);
    setCopiedId(reqId);
    onShowToast('Request ID copied to clipboard', 'success');
    setTimeout(() => setCopiedId(null), 2000);
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'delivered': return 'badge-success';
      case 'failed': return 'badge-danger';
      case 'dead_letter': return 'badge-dead';
      default: return 'badge-warning'; // queued, processing, scheduled
    }
  };

  return (
    <div className="card">
      {/* Top Controls Bar */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: '20px',
        flexWrap: 'wrap',
        gap: '12px'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <h3 style={{ fontSize: '16px', margin: 0 }}>Delivery History</h3>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            Showing latest {logs.length} notifications
          </span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          {/* Limit selector */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12px' }}>
            <span style={{ color: 'var(--text-secondary)' }}>Limit:</span>
            <select
              className="form-select"
              style={{ padding: '4px 8px', fontSize: '12px', width: 'auto' }}
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
            >
              <option value={25}>25</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </select>
          </div>

          {/* Auto Refresh Toggle */}
          <button
            className={`btn btn-sm ${autoRefresh ? 'btn-accent' : ''}`}
            onClick={() => {
              setAutoRefresh(!autoRefresh);
              setCountdown(5);
            }}
          >
            {autoRefresh ? `⏸ Auto (${countdown}s)` : '▶ Enable Auto-Refresh'}
          </button>

          {/* Manual Refresh Button */}
          <button
            className="btn btn-sm"
            onClick={() => {
              fetchLogs(true);
              setCountdown(5);
            }}
          >
            🔄 Refresh
          </button>
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

      {loading && logs.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-secondary)' }}>
          Loading delivery logs...
        </div>
      ) : logs.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-faint)' }}>
          No delivery records found for this tenant. Dispatch your first notification!
        </div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{
            width: '100%',
            borderCollapse: 'collapse',
            fontSize: '13px',
            textAlign: 'left'
          }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                <th style={{ padding: '10px 12px' }}>Request ID</th>
                <th style={{ padding: '10px 12px' }}>Channel</th>
                <th style={{ padding: '10px 12px' }}>Recipient</th>
                <th style={{ padding: '10px 12px' }}>Status</th>
                <th style={{ padding: '10px 12px' }}>Attempts</th>
                <th style={{ padding: '10px 12px' }}>Fallback</th>
                <th style={{ padding: '10px 12px' }}>Created At</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => {
                const truncatedId = `${log.request_id.slice(0, 8)}...${log.request_id.slice(-4)}`;
                const isCopied = copiedId === log.request_id;
                return (
                  <tr key={log.id} style={{ borderBottom: '1px solid var(--border-strong)', transition: 'background-color 0.15s' }}>
                    <td style={{ padding: '10px 12px' }}>
                      <button
                        onClick={() => handleCopyRequestId(log.request_id)}
                        title="Click to copy full Request ID"
                        style={{
                          background: 'none',
                          border: 'none',
                          color: isCopied ? 'var(--success)' : 'var(--accent)',
                          fontFamily: 'var(--font-mono)',
                          fontSize: '12px',
                          cursor: 'pointer',
                          padding: 0
                        }}
                      >
                        {isCopied ? 'Copied!' : truncatedId}
                      </button>
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)' }}>
                      {log.channel}
                    </td>
                    <td style={{
                      padding: '10px 12px',
                      maxWidth: '200px',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      fontFamily: 'var(--font-mono)',
                      fontSize: '12px'
                    }}>
                      {log.recipient || log.email_recipient || log.discord_recipient || '-'}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <span className={`badge ${getStatusBadgeClass(log.status)}`}>
                        {log.status}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)' }}>
                      {log.attempts}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      {log.fallback_used ? (
                        <span style={{ color: 'var(--warning)', fontWeight: 600 }}>Yes</span>
                      ) : (
                        <span style={{ color: 'var(--text-faint)' }}>No</span>
                      )}
                    </td>
                    <td style={{ padding: '10px 12px', color: 'var(--text-secondary)', fontSize: '12px' }}>
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
