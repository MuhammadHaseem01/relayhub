import React, { useState, useEffect } from 'react';
import {
  Send,
  Activity,
  Zap,
  ArrowRight,
  TrendingUp,
  Clock
} from 'lucide-react';
import { getLogs, type NotificationLog } from '../api/client';

interface DashboardPageProps {
  onNavigateToSend: () => void;
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const DashboardPage: React.FC<DashboardPageProps> = ({ onNavigateToSend }) => {
  const [logs, setLogs] = useState<NotificationLog[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      const res = await getLogs(50);
      setLoading(false);
      if (res.success && res.data) {
        setLogs(res.data.logs || []);
      }
    };
    load();
  }, []);

  // Compute stat metrics
  const totalSent = logs.length;
  const deliveredCount = logs.filter(l => l.status === 'delivered').length;
  const failedCount = logs.filter(l => l.status === 'failed' || l.status === 'dead_letter').length;
  const successRate = totalSent > 0 ? Math.round((deliveredCount / totalSent) * 100) : 100;
  const recentLogs = logs.slice(0, 5);

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'delivered': return 'badge-success';
      case 'failed': return 'badge-danger';
      case 'dead_letter': return 'badge-dead';
      default: return 'badge-warning';
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Welcome Banner */}
      <div className="card" style={{
        background: 'linear-gradient(135deg, var(--surface) 0%, var(--surface-2) 100%)',
        border: '1px solid var(--border)',
        padding: '24px 28px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: '16px'
      }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
            <span style={{ fontSize: '12px', fontWeight: 600, color: 'var(--accent)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              RelayHub Overview
            </span>
          </div>
          <h2 style={{ fontSize: '22px', marginBottom: '4px' }}>Welcome to your Notification Dashboard</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '13px', maxWidth: '600px' }}>
            Monitor message delivery status, inspect raw payloads, and test multi-channel routing across Discord, Resend, and SMTP.
          </p>
        </div>

        <button className="btn btn-primary" onClick={onNavigateToSend} style={{ padding: '10px 20px' }}>
          <Send size={16} />
          <span>Send Test Notification</span>
          <ArrowRight size={16} />
        </button>
      </div>

      {/* 4 Stat Cards Row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '16px' }}>
        {/* Stat Card 1: Sent Today */}
        <div className="card card-interactive" onClick={onNavigateToSend}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 500 }}>Total Sent</span>
            <div style={{ color: 'var(--accent)', padding: '6px', borderRadius: '8px', backgroundColor: 'var(--accent-bg)' }}>
              <Send size={18} />
            </div>
          </div>
          <div style={{ fontSize: '28px', fontWeight: 700, fontFamily: 'var(--font-display)', marginBottom: '4px' }}>
            {totalSent}
          </div>
          <div style={{ fontSize: '12px', color: 'var(--text-faint)' }}>
            Total notifications logged
          </div>
        </div>

        {/* Stat Card 2: Success Rate */}
        <div className="card card-interactive">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 500 }}>Success Rate</span>
            <div style={{ color: 'var(--success)', padding: '6px', borderRadius: '8px', backgroundColor: 'rgba(63, 185, 80, 0.1)' }}>
              <TrendingUp size={18} />
            </div>
          </div>
          <div style={{ fontSize: '28px', fontWeight: 700, fontFamily: 'var(--font-display)', marginBottom: '4px' }}>
            {successRate}%
          </div>
          <div style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'flex', gap: '8px' }}>
            <span style={{ color: 'var(--success)' }}>✓ {deliveredCount} delivered</span>
            {failedCount > 0 && <span style={{ color: 'var(--danger)' }}>✕ {failedCount} failed</span>}
          </div>
        </div>

        {/* Stat Card 3: Active Providers */}
        <div className="card card-interactive">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 500 }}>Active Providers</span>
            <div style={{ color: 'var(--accent)', padding: '6px', borderRadius: '8px', backgroundColor: 'var(--accent-bg)' }}>
              <Zap size={18} />
            </div>
          </div>
          <div style={{ fontSize: '28px', fontWeight: 700, fontFamily: 'var(--font-display)', marginBottom: '4px' }}>
            3
          </div>
          <div style={{ fontSize: '12px', color: 'var(--text-faint)' }}>
            Discord, Resend Email, SMTP
          </div>
        </div>

        {/* Stat Card 4: Rate Limit Remaining */}
        <div className="card card-interactive">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 500 }}>Daily Rate Limit</span>
            <div style={{ color: 'var(--warning)', padding: '6px', borderRadius: '8px', backgroundColor: 'rgba(210, 153, 34, 0.1)' }}>
              <Activity size={18} />
            </div>
          </div>
          <div style={{ fontSize: '28px', fontWeight: 700, fontFamily: 'var(--font-display)', marginBottom: '4px' }}>
            100 / day
          </div>
          <div style={{ fontSize: '12px', color: 'var(--text-faint)' }}>
            Free tier default quota
          </div>
        </div>
      </div>

      {/* Recent Activity Table */}
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Clock size={16} style={{ color: 'var(--text-secondary)' }} />
            <h3 style={{ fontSize: '15px', margin: 0 }}>Recent Activity</h3>
          </div>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            Last 5 notifications
          </span>
        </div>

        {loading ? (
          <div style={{ padding: '30px', textAlign: 'center', color: 'var(--text-secondary)' }}>
            Loading recent logs...
          </div>
        ) : recentLogs.length === 0 ? (
          <div className="empty-state">
            <Send size={32} className="empty-state-icon" />
            <p className="empty-state-text">No notifications sent yet. Click below to send your first test notification!</p>
            <button className="btn btn-primary btn-sm" onClick={onNavigateToSend}>
              Send Notification
            </button>
          </div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                  <th style={{ padding: '10px 12px' }}>Request ID</th>
                  <th style={{ padding: '10px 12px' }}>Channel</th>
                  <th style={{ padding: '10px 12px' }}>Recipient</th>
                  <th style={{ padding: '10px 12px' }}>Status</th>
                  <th style={{ padding: '10px 12px' }}>Time</th>
                </tr>
              </thead>
              <tbody>
                {recentLogs.map((log) => {
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
                      <td style={{ padding: '10px 12px' }}>
                        <span className={`badge ${getStatusBadgeClass(log.status)}`}>
                          {log.status}
                        </span>
                      </td>
                      <td style={{ padding: '10px 12px', color: 'var(--text-faint)', fontSize: '12px' }}>
                        {new Date(log.created_at).toLocaleTimeString()}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
