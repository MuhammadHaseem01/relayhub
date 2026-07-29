import React from 'react';
import type { NavView } from '../components/Layout';

interface PlaceholderPageProps {
  view: NavView;
}

const VIEW_TITLES: Record<NavView, { title: string; desc: string; icon: string }> = {
  send: { title: 'Send Notification', desc: 'Dispatch notifications across providers', icon: '🚀' },
  logs: { title: 'Delivery Logs', desc: 'Real-time audit log of all notifications', icon: '📋' },
  templates: { title: 'Template Management', desc: 'Create and edit dynamic Handlebars-style templates with variables', icon: '🧩' },
  webhooks: { title: 'Outbound Webhooks', desc: 'Configure tenant webhook endpoints and HMAC-SHA256 signature verification', icon: '🔗' },
  dlq: { title: 'Dead Letter Queue (DLQ)', desc: 'Inspect and replay notifications that failed after max worker attempts', icon: '💀' },
  health: { title: 'Provider Health & Circuit Breakers', desc: 'Monitor real-time circuit breaker status across Discord, Email, and SMTP', icon: '💚' },
  usage: { title: 'Tenant Quota & Usage', desc: 'View 24-hour rate limit usage and tier quotas', icon: '📊' },
};

export const PlaceholderPage: React.FC<PlaceholderPageProps> = ({ view }) => {
  const info = VIEW_TITLES[view] || { title: view, desc: 'Feature coming in Phase 5 Step 2', icon: '⚡' };

  return (
    <div className="card" style={{ textAlign: 'center', padding: '48px 24px' }}>
      <div style={{ fontSize: '48px', marginBottom: '16px' }}>{info.icon}</div>
      <h3 style={{ fontSize: '20px', marginBottom: '8px' }}>{info.title}</h3>
      <p style={{ color: 'var(--text-secondary)', maxWidth: '480px', margin: '0 auto 24px auto', fontSize: '14px' }}>
        {info.desc}
      </p>
      <div style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        padding: '6px 12px',
        borderRadius: '20px',
        backgroundColor: 'var(--surface-2)',
        border: '1px solid var(--border)',
        fontSize: '12px',
        color: 'var(--accent)'
      }}>
        <span>⚙️ Scheduled for Phase 5 — Step 2</span>
      </div>
    </div>
  );
};
