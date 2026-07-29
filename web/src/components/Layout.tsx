import React from 'react';
import { getApiKey, clearApiKey } from '../api/client';

export type NavView = 'send' | 'logs' | 'templates' | 'webhooks' | 'dlq' | 'health' | 'usage';

interface LayoutProps {
  currentView: NavView;
  onNavigate: (view: NavView) => void;
  onLogout: () => void;
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ currentView, onNavigate, onLogout, children }) => {
  const apiKey = getApiKey() || '';
  const maskedKey = apiKey ? `${apiKey.slice(0, 7)}...${apiKey.slice(-6)}` : '';

  const navItems: { id: NavView; label: string; icon: string; disabled?: boolean }[] = [
    { id: 'send', label: 'Send Notification', icon: '🚀' },
    { id: 'logs', label: 'Delivery Logs', icon: '📋' },
    { id: 'templates', label: 'Templates', icon: '🧩', disabled: true },
    { id: 'webhooks', label: 'Webhooks', icon: '🔗', disabled: true },
    { id: 'dlq', label: 'Dead Letter', icon: '💀', disabled: true },
    { id: 'health', label: 'Provider Health', icon: '💚', disabled: true },
    { id: 'usage', label: 'Usage', icon: '📊', disabled: true },
  ];

  const handleCopyKey = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey);
    }
  };

  const handleLogout = () => {
    clearApiKey();
    onLogout();
  };

  return (
    <div style={{ display: 'flex', minHeight: '100vh', backgroundColor: 'var(--bg)' }}>
      {/* Sidebar */}
      <aside style={{
        width: '240px',
        backgroundColor: 'var(--surface)',
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
        position: 'fixed',
        top: 0,
        bottom: 0,
        left: 0,
        zIndex: 10
      }}>
        {/* Sidebar Header */}
        <div style={{
          padding: '16px 20px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          gap: '10px'
        }}>
          <div style={{
            width: '28px',
            height: '28px',
            borderRadius: '6px',
            backgroundColor: '#1f6feb',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '16px'
          }}>⚡</div>
          <div>
            <h1 style={{ fontSize: '15px', letterSpacing: '-0.02em', margin: 0 }}>RelayHub</h1>
            <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>Control Panel</span>
          </div>
        </div>

        {/* Navigation Items */}
        <nav style={{ flex: 1, padding: '12px 8px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {navItems.map((item) => {
            const isActive = currentView === item.id;
            return (
              <button
                key={item.id}
                onClick={() => !item.disabled && onNavigate(item.id)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  padding: '8px 12px',
                  borderRadius: 'var(--radius-md)',
                  border: 'none',
                  backgroundColor: isActive ? 'var(--surface-2)' : 'transparent',
                  color: isActive ? 'var(--text-primary)' : item.disabled ? 'var(--text-faint)' : 'var(--text-secondary)',
                  fontWeight: isActive ? 600 : 400,
                  fontSize: '13px',
                  cursor: item.disabled ? 'not-allowed' : 'pointer',
                  textAlign: 'left',
                  width: '100%',
                  transition: 'all 0.15s ease'
                }}
              >
                <span>{item.icon}</span>
                <span style={{ flex: 1 }}>{item.label}</span>
                {item.disabled && (
                  <span style={{
                    fontSize: '10px',
                    padding: '2px 6px',
                    borderRadius: '8px',
                    backgroundColor: 'var(--bg)',
                    color: 'var(--text-faint)',
                    border: '1px solid var(--border)'
                  }}>
                    Step 2
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        {/* Sidebar Footer */}
        <div style={{
          padding: '12px 16px',
          borderTop: '1px solid var(--border)',
          fontSize: '11px',
          color: 'var(--text-faint)'
        }}>
          RelayHub v1.0.0 • Phase 5
        </div>
      </aside>

      {/* Main Content Area */}
      <div style={{ flex: 1, marginLeft: '240px', display: 'flex', flexDirection: 'column' }}>
        {/* Top Header Bar */}
        <header style={{
          height: '56px',
          backgroundColor: 'var(--surface)',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 24px',
          position: 'sticky',
          top: 0,
          zIndex: 5
        }}>
          <h2 style={{ fontSize: '16px', fontWeight: 600 }}>
            {navItems.find(n => n.id === currentView)?.label || 'Dashboard'}
          </h2>

          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {/* Masked API Key */}
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              backgroundColor: 'var(--bg)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              padding: '4px 10px',
              fontSize: '12px'
            }}>
              <span style={{ color: 'var(--text-faint)' }}>API Key:</span>
              <code style={{ color: 'var(--accent)', fontSize: '11px' }}>{maskedKey}</code>
              <button
                className="btn btn-sm"
                onClick={handleCopyKey}
                title="Copy Full API Key"
                style={{ padding: '2px 6px', fontSize: '10px', marginLeft: '4px' }}
              >
                Copy
              </button>
            </div>

            {/* Logout Button */}
            <button className="btn btn-sm" onClick={handleLogout} style={{ color: 'var(--danger)', borderColor: 'rgba(248,81,73,0.3)' }}>
              Log out
            </button>
          </div>
        </header>

        {/* Body Content */}
        <main style={{ flex: 1, padding: '24px', maxWidth: '1200px', width: '100%', margin: '0 auto' }}>
          {children}
        </main>
      </div>
    </div>
  );
};
