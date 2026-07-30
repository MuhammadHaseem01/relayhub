import React, { useState } from 'react';
import { AlertTriangle, Copy, Check } from 'lucide-react';

interface SecretRevealProps {
  title?: string;
  warning: string;
  secret: string;
  onDismiss?: () => void;
  dismissText?: string;
}

export const SecretReveal: React.FC<SecretRevealProps> = ({
  title = 'Save Secret',
  warning,
  secret,
  onDismiss,
  dismissText = 'I have saved this secret'
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(secret);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div style={{
      backgroundColor: 'rgba(210, 153, 34, 0.12)',
      border: '1px solid var(--warning)',
      borderRadius: 'var(--radius-md)',
      padding: '16px',
      marginBottom: '20px'
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
        <AlertTriangle size={18} style={{ color: 'var(--warning)' }} />
        <h4 style={{ color: 'var(--warning)', margin: 0, fontSize: '14px' }}>{title}</h4>
      </div>

      <p style={{ fontSize: '12px', color: 'var(--text-primary)', marginBottom: '12px', lineHeight: 1.4 }}>
        {warning}
      </p>

      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        backgroundColor: 'var(--bg)',
        padding: '8px 12px',
        borderRadius: 'var(--radius-sm)',
        border: '1px solid var(--border)',
        marginBottom: onDismiss ? '12px' : '0'
      }}>
        <code style={{ flex: 1, fontSize: '12px', color: 'var(--accent)', wordBreak: 'break-all', fontFamily: 'var(--font-mono)' }}>
          {secret}
        </code>
        <button className="btn btn-sm" onClick={handleCopy}>
          {copied ? <Check size={14} style={{ color: 'var(--success)' }} /> : <Copy size={14} />}
          <span>{copied ? 'Copied' : 'Copy'}</span>
        </button>
      </div>

      {onDismiss && (
        <button className="btn btn-primary btn-sm" style={{ width: '100%' }} onClick={onDismiss}>
          {dismissText}
        </button>
      )}
    </div>
  );
};
