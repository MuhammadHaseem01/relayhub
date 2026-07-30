import React, { useState } from 'react';
import { Zap, PlusCircle, Key, Copy, Check, AlertTriangle } from 'lucide-react';
import { registerTenant, setApiKey } from '../api/client';

interface AuthModalProps {
  onSuccess: () => void;
}

export const AuthModal: React.FC<AuthModalProps> = ({ onSuccess }) => {
  const [tab, setTab] = useState<'create' | 'existing'>('create');
  const [tenantName, setTenantName] = useState('');
  const [existingKey, setExistingKey] = useState('');
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tenantName.trim()) return;

    setLoading(true);
    setError(null);
    const res = await registerTenant(tenantName.trim());
    setLoading(false);

    if (res.success && res.data) {
      setCreatedKey(res.data.api_key);
    } else {
      setError(res.error || 'Failed to create tenant');
    }
  };

  const handleUseExisting = (e: React.FormEvent) => {
    e.preventDefault();
    if (!existingKey.trim()) return;
    setApiKey(existingKey.trim());
    onSuccess();
  };

  const handleContinueWithCreated = () => {
    if (createdKey) {
      setApiKey(createdKey);
      onSuccess();
    }
  };

  const handleCopyKey = () => {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      backgroundColor: 'rgba(13, 17, 23, 0.85)',
      backdropFilter: 'blur(6px)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
      padding: '20px'
    }}>
      <div className="card" style={{ width: '100%', maxWidth: '480px', boxShadow: '0 20px 40px rgba(0,0,0,0.6)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '20px' }}>
          <div style={{
            width: '40px',
            height: '40px',
            borderRadius: 'var(--radius-md)',
            backgroundColor: '#1f6feb',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#ffffff'
          }}>
            <Zap size={24} />
          </div>
          <div>
            <h2 style={{ fontSize: '18px', margin: 0 }}>RelayHub Dashboard</h2>
            <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Universal Notification Engine</p>
          </div>
        </div>

        {createdKey ? (
          <div>
            <div style={{
              backgroundColor: 'rgba(210, 153, 34, 0.15)',
              border: '1px solid var(--warning)',
              borderRadius: 'var(--radius-md)',
              padding: '16px',
              marginBottom: '16px'
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                <AlertTriangle size={16} style={{ color: 'var(--warning)' }} />
                <h4 style={{ color: 'var(--warning)', margin: 0, fontSize: '14px' }}>Save Your API Key Now</h4>
              </div>
              <p style={{ fontSize: '12px', color: 'var(--text-primary)', marginBottom: '12px' }}>
                This key will <strong>never be displayed again</strong>. Copy it and store it in a safe place.
              </p>
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                backgroundColor: 'var(--bg)',
                padding: '8px 12px',
                borderRadius: 'var(--radius-sm)',
                border: '1px solid var(--border)'
              }}>
                <code style={{ flex: 1, fontSize: '12px', color: 'var(--accent)', wordBreak: 'break-all' }}>
                  {createdKey}
                </code>
                <button className="btn btn-sm" onClick={handleCopyKey}>
                  {copied ? <Check size={14} style={{ color: 'var(--success)' }} /> : <Copy size={14} />}
                  <span>{copied ? 'Copied' : 'Copy'}</span>
                </button>
              </div>
            </div>
            <button className="btn btn-primary" style={{ width: '100%' }} onClick={handleContinueWithCreated}>
              Continue to Dashboard
            </button>
          </div>
        ) : (
          <>
            <div style={{
              display: 'flex',
              borderBottom: '1px solid var(--border)',
              marginBottom: '20px'
            }}>
              <button
                style={{
                  flex: 1,
                  padding: '10px',
                  background: 'none',
                  border: 'none',
                  borderBottom: tab === 'create' ? '2px solid var(--accent)' : '2px solid transparent',
                  color: tab === 'create' ? 'var(--text-primary)' : 'var(--text-secondary)',
                  fontWeight: tab === 'create' ? 600 : 400,
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '8px'
                }}
                onClick={() => { setTab('create'); setError(null); }}
              >
                <PlusCircle size={16} />
                <span>Register Tenant</span>
              </button>
              <button
                style={{
                  flex: 1,
                  padding: '10px',
                  background: 'none',
                  border: 'none',
                  borderBottom: tab === 'existing' ? '2px solid var(--accent)' : '2px solid transparent',
                  color: tab === 'existing' ? 'var(--text-primary)' : 'var(--text-secondary)',
                  fontWeight: tab === 'existing' ? 600 : 400,
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '8px'
                }}
                onClick={() => { setTab('existing'); setError(null); }}
              >
                <Key size={16} />
                <span>Existing Key</span>
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

            {tab === 'create' ? (
              <form onSubmit={handleCreate}>
                <div className="form-group">
                  <label className="form-label">Tenant Application Name</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="e.g. My Next.js Web App"
                    value={tenantName}
                    onChange={(e) => setTenantName(e.target.value)}
                    required
                  />
                  <span className="form-hint">Registers a new isolated tenant account in RelayHub.</span>
                </div>
                <button type="submit" className="btn btn-primary" style={{ width: '100%' }} disabled={loading}>
                  {loading ? 'Creating Tenant...' : 'Create Tenant & Get API Key'}
                </button>
              </form>
            ) : (
              <form onSubmit={handleUseExisting}>
                <div className="form-group">
                  <label className="form-label">API Key</label>
                  <input
                    type="password"
                    className="form-input font-mono"
                    placeholder="rh_a3f9c2d1..."
                    value={existingKey}
                    onChange={(e) => setExistingKey(e.target.value)}
                    required
                  />
                  <span className="form-hint">Enter your existing 64-character RelayHub API key.</span>
                </div>
                <button type="submit" className="btn btn-primary" style={{ width: '100%' }}>
                  Authenticate & Continue
                </button>
              </form>
            )}

            <div style={{ marginTop: '20px', paddingTop: '12px', borderTop: '1px solid var(--border)' }}>
              <p style={{ fontSize: '11px', color: 'var(--text-faint)', textAlign: 'center' }}>
                Note: API keys are stored in browser memory & localStorage for local testing convenience.
              </p>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
