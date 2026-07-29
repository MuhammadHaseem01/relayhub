import React, { useState } from 'react';
import {
  Zap,
  LayoutDashboard,
  Send,
  ScrollText,
  Puzzle,
  Link2,
  Skull,
  HeartPulse,
  BarChart3,
  Copy,
  Check,
  LogOut,
  Key
} from 'lucide-react';
import { getApiKey, clearApiKey } from '../api/client';

export type NavView = 'dashboard' | 'send' | 'logs' | 'templates' | 'webhooks' | 'dlq' | 'health' | 'usage';

interface LayoutProps {
  currentView: NavView;
  onNavigate: (view: NavView) => void;
  onLogout: () => void;
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ currentView, onNavigate, onLogout, children }) => {
  const apiKey = getApiKey() || '';
  const maskedKey = apiKey ? `${apiKey.slice(0, 7)}...${apiKey.slice(-6)}` : '';
  const [copiedKey, setCopiedKey] = useState(false);

  const navItems: { id: NavView; label: string; icon: React.ReactNode; disabled?: boolean }[] = [
    { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard size={18} /> },
    { id: 'send', label: 'Send Notification', icon: <Send size={18} /> },
    { id: 'logs', label: 'Delivery Logs', icon: <ScrollText size={18} /> },
    { id: 'templates', label: 'Templates', icon: <Puzzle size={18} />, disabled: true },
    { id: 'webhooks', label: 'Webhooks', icon: <Link2 size={18} />, disabled: true },
    { id: 'dlq', label: 'Dead Letter', icon: <Skull size={18} />, disabled: true },
    { id: 'health', label: 'Provider Health', icon: <HeartPulse size={18} />, disabled: true },
    { id: 'usage', label: 'Usage', icon: <BarChart3 size={18} />, disabled: true },
  ];

  const handleCopyKey = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey);
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
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
          padding: '18px 20px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          gap: '10px'
        }}>
          <div style={{
            width: '32px',
            height: '32px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: '#1f6feb',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#ffffff'
          }}>
            <Zap size={20} />
          </div>
          <div>
            <h1 style={{ fontSize: '15px', letterSpacing: '-0.02em', margin: 0 }}>RelayHub</h1>
            <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>Control Panel</span>
          </div>
        </div>

        {/* Navigation Items */}
        <nav style={{ flex: 1, padding: '16px 10px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {navItems.map((item) => {
            const isActive = currentView === item.id;
            return (
              <button
                key={item.id}
                onClick={() => !item.disabled && onNavigate(item.id)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  padding: '9px 12px',
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
                <span style={{ color: isActive ? 'var(--accent)' : 'inherit', display: 'flex', alignItems: 'center' }}>
                  {item.icon}
                </span>
                <span style={{ flex: 1 }}>{item.label}</span>
                {item.disabled && (
                  <span style={{
                    fontSize: '10px',
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    padding: '2px 6px',
                    borderRadius: '4px',
                    backgroundColor: 'var(--surface-2)',
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
          padding: '14px 16px',
          borderTop: '1px solid var(--border)',
          fontSize: '11px',
          color: 'var(--text-faint)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between'
        }}>
          <span>RelayHub v1.0.0</span>
          <span>Phase 5</span>
        </div>
      </aside>

      {/* Main Content Area */}
      <div style={{ flex: 1, marginLeft: '240px', display: 'flex', flexDirection: 'column' }}>
        {/* Top Header Bar */}
        <header style={{
          height: '60px',
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
            {/* Unified Tenant API Key Chip */}
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              backgroundColor: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: '20px',
              padding: '4px 12px',
              fontSize: '12px'
            }}>
              <Key size={14} style={{ color: 'var(--text-secondary)' }} />
              <code style={{ color: 'var(--accent)', fontSize: '12px' }}>{maskedKey}</code>
              <button
                onClick={handleCopyKey}
                title="Copy Full API Key"
                style={{
                  background: 'none',
                  border: 'none',
                  color: copiedKey ? 'var(--success)' : 'var(--text-secondary)',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  padding: '2px',
                  marginLeft: '2px'
                }}
              >
                {copiedKey ? <Check size={14} /> : <Copy size={14} />}
              </button>
            </div>

            {/* Ghost Logout Button */}
            <button className="btn btn-ghost btn-sm" onClick={handleLogout}>
              <LogOut size={14} />
              <span>Log out</span>
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
