import React from 'react';
import { Puzzle, Link2, Skull, HeartPulse, BarChart3, Clock } from 'lucide-react';
import type { NavView } from '../components/Layout';

interface PlaceholderPageProps {
  view: NavView;
}

const VIEW_TITLES: Record<NavView, { title: string; desc: string; icon: React.ReactNode }> = {
  dashboard: { title: 'Dashboard Overview', desc: 'Real-time overview', icon: <BarChart3 size={40} /> },
  send: { title: 'Send Notification', desc: 'Dispatch notifications across providers', icon: <Puzzle size={40} /> },
  logs: { title: 'Delivery Logs', desc: 'Real-time audit log of all notifications', icon: <Puzzle size={40} /> },
  templates: { title: 'Template Management', desc: 'Create and edit dynamic Handlebars-style templates with variables', icon: <Puzzle size={40} /> },
  webhooks: { title: 'Outbound Webhooks', desc: 'Configure tenant webhook endpoints and HMAC-SHA256 signature verification', icon: <Link2 size={40} /> },
  dlq: { title: 'Dead Letter Queue (DLQ)', desc: 'Inspect and replay notifications that failed after max worker attempts', icon: <Skull size={40} /> },
  health: { title: 'Provider Health & Circuit Breakers', desc: 'Monitor real-time circuit breaker status across Discord, Email, and SMTP', icon: <HeartPulse size={40} /> },
  usage: { title: 'Tenant Quota & Usage', desc: 'View 24-hour rate limit usage and tier quotas', icon: <BarChart3 size={40} /> },
};

export const PlaceholderPage: React.FC<PlaceholderPageProps> = ({ view }) => {
  const info = VIEW_TITLES[view] || { title: view, desc: 'Feature coming in Phase 5 Step 2', icon: <Puzzle size={40} /> };

  return (
    <div className="card" style={{ textAlign: 'center', padding: '64px 24px' }}>
      <div style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '72px',
        height: '72px',
        borderRadius: '20px',
        backgroundColor: 'var(--accent-bg)',
        color: 'var(--accent)',
        marginBottom: '20px'
      }}>
        {info.icon}
      </div>

      <h3 style={{ fontSize: '20px', marginBottom: '8px' }}>{info.title}</h3>
      <p style={{ color: 'var(--text-secondary)', maxWidth: '480px', margin: '0 auto 24px auto', fontSize: '14px' }}>
        {info.desc}
      </p>

      <div style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        padding: '6px 14px',
        borderRadius: '20px',
        backgroundColor: 'var(--surface-2)',
        border: '1px solid var(--border)',
        fontSize: '12px',
        color: 'var(--accent)',
        fontWeight: 500
      }}>
        <Clock size={14} />
        <span>Scheduled for Phase 5 — Step 2</span>
      </div>
    </div>
  );
};
