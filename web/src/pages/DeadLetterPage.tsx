import React, { useState, useEffect } from 'react';
import { Skull, RefreshCw, RotateCcw, CheckCircle2 } from 'lucide-react';
import { getDeadLetterNotifications, replayNotification, type NotificationLog } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { ConfirmDialog } from '../components/ConfirmDialog';

interface DeadLetterPageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const DeadLetterPage: React.FC<DeadLetterPageProps> = ({ onShowToast }) => {
  const [deadLetters, setDeadLetters] = useState<NotificationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Replay dialog state
  const [replayTarget, setReplayTarget] = useState<string | null>(null);
  const [replaying, setReplaying] = useState(false);

  const fetchDeadLetters = async (showLoading = false) => {
    if (showLoading) setLoading(true);
    setError(null);
    const res = await getDeadLetterNotifications(50);
    setLoading(false);

    if (res.success && res.data) {
      setDeadLetters(res.data.notifications || []);
    } else {
      setError(res.error || 'Failed to fetch dead letter notifications');
    }
  };

  useEffect(() => {
    fetchDeadLetters(true);
  }, []);

  const handleReplayConfirm = async () => {
    if (!replayTarget) return;
    setReplaying(true);
    const res = await replayNotification(replayTarget);
    setReplaying(false);

    if (res.success) {
      onShowToast(`Notification ${replayTarget.slice(0, 8)}... re-enqueued for delivery`, 'success');
      setReplayTarget(null);
      fetchDeadLetters(false);
    } else {
      onShowToast(res.error || 'Failed to replay notification', 'error');
    }
  };

  return (
    <div className="card">
      {/* Top Bar Controls */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: '20px',
        flexWrap: 'wrap',
        gap: '12px'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Skull size={20} style={{ color: 'var(--dead)' }} />
          <h3 style={{ fontSize: '16px', margin: 0 }}>Dead Letter Queue (DLQ)</h3>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            ({deadLetters.length} jobs in DLQ)
          </span>
        </div>

        <button className="btn btn-sm" onClick={() => fetchDeadLetters(true)}>
          <RefreshCw size={14} />
          <span>Refresh List</span>
        </button>
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

      {loading && deadLetters.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-secondary)' }}>
          Checking Dead Letter Queue...
        </div>
      ) : deadLetters.length === 0 ? (
        /* Positive Empty State */
        <div className="empty-state" style={{ padding: '48px 20px' }}>
          <div style={{
            width: '64px',
            height: '64px',
            borderRadius: '50%',
            backgroundColor: 'rgba(63, 185, 80, 0.12)',
            color: 'var(--success)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '8px'
          }}>
            <CheckCircle2 size={36} />
          </div>
          <h4 style={{ fontSize: '16px', color: 'var(--text-primary)', margin: 0 }}>Dead Letter Queue is Clear</h4>
          <p className="empty-state-text" style={{ maxWidth: '420px' }}>
            No dead-lettered notifications found — all queued messages are being processed and delivered successfully!
          </p>
        </div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                <th style={{ padding: '10px 12px' }}>Request ID</th>
                <th style={{ padding: '10px 12px' }}>Channel</th>
                <th style={{ padding: '10px 12px' }}>Recipient</th>
                <th style={{ padding: '10px 12px' }}>Worker Attempts</th>
                <th style={{ padding: '10px 12px' }}>Error Details</th>
                <th style={{ padding: '10px 12px' }}>Timestamp</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {deadLetters.map((log) => {
                const truncatedId = `${log.request_id.slice(0, 8)}...${log.request_id.slice(-4)}`;
                return (
                  <tr key={log.id} style={{ borderBottom: '1px solid var(--border-strong)' }}>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', color: 'var(--accent)', fontSize: '12px' }}>
                      {truncatedId}
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)' }}>
                      {log.channel}
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', fontSize: '12px', color: 'var(--text-secondary)' }}>
                      {log.recipient || log.email_recipient || log.discord_recipient || '-'}
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)' }}>
                      <StatusBadge status={`${log.attempts} attempts`} />
                    </td>
                    <td style={{ padding: '10px 12px', color: 'var(--danger)', fontSize: '12px', maxWidth: '240px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {log.error_message || 'Exceeded max worker attempts'}
                    </td>
                    <td style={{ padding: '10px 12px', color: 'var(--text-faint)', fontSize: '12px' }}>
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                    <td style={{ padding: '10px 12px', textAlign: 'right' }}>
                      <button
                        className="btn btn-primary btn-sm"
                        onClick={() => setReplayTarget(log.request_id)}
                      >
                        <RotateCcw size={14} />
                        <span>Replay</span>
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Replay Confirm Dialog */}
      <ConfirmDialog
        isOpen={Boolean(replayTarget)}
        title="Replay Dead-Lettered Notification"
        message="This will re-enqueue the notification job onto the main Redis stream with worker_attempts reset to 0 for another delivery attempt."
        confirmText="Replay Job Now"
        isDanger={false}
        loading={replaying}
        onConfirm={handleReplayConfirm}
        onCancel={() => setReplayTarget(null)}
      />
    </div>
  );
};
