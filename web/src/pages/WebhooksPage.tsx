import React, { useState, useEffect } from 'react';
import { Link2, Trash2, Code2, ChevronDown, ChevronUp, Save, Send } from 'lucide-react';
import { getWebhookConfig, setWebhook, deleteWebhook, getWebhookDeliveries, type WebhookDelivery } from '../api/client';
import { SecretReveal } from '../components/SecretReveal';
import { StatusBadge } from '../components/StatusBadge';
import { ConfirmDialog } from '../components/ConfirmDialog';

interface WebhooksPageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const WebhooksPage: React.FC<WebhooksPageProps> = ({ onShowToast }) => {
  const [currentUrl, setCurrentUrl] = useState<string>('');
  const [inputUrl, setInputUrl] = useState<string>('');
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);

  const [loadingConfig, setLoadingConfig] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [showRemoveConfirm, setShowRemoveConfirm] = useState(false);
  const [removing, setRemoving] = useState(false);
  const [showSnippets, setShowSnippets] = useState(false);

  const fetchConfigAndDeliveries = async () => {
    setLoadingConfig(true);
    const [cfgRes, delRes] = await Promise.all([
      getWebhookConfig(),
      getWebhookDeliveries(50),
    ]);
    setLoadingConfig(false);

    if (cfgRes.success && cfgRes.data) {
      setCurrentUrl(cfgRes.data.webhook_url || '');
      setInputUrl(cfgRes.data.webhook_url || '');
    }

    if (delRes.success && delRes.data) {
      setDeliveries(delRes.data.deliveries || []);
    }
  };

  useEffect(() => {
    fetchConfigAndDeliveries();
  }, []);

  const handleSaveWebhook = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputUrl.trim()) return;

    if (!inputUrl.startsWith('https://')) {
      onShowToast('Webhook URL must use HTTPS protocol', 'error');
      return;
    }

    setSubmitting(true);
    const res = await setWebhook(inputUrl.trim());
    setSubmitting(false);

    if (res.success && res.data) {
      setCurrentUrl(res.data.webhook_url);
      if (res.data.webhook_secret) {
        setRevealedSecret(res.data.webhook_secret);
      }
      onShowToast('Webhook endpoint configured successfully', 'success');
      fetchConfigAndDeliveries();
    } else {
      onShowToast(res.error || 'Failed to configure webhook', 'error');
    }
  };

  const handleRemoveWebhook = async () => {
    setRemoving(true);
    const res = await deleteWebhook();
    setRemoving(false);

    if (res.success) {
      setCurrentUrl('');
      setInputUrl('');
      setRevealedSecret(null);
      setShowRemoveConfirm(false);
      onShowToast('Webhook configuration removed', 'success');
    } else {
      onShowToast(res.error || 'Failed to remove webhook', 'error');
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Configure Webhook Card */}
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px', borderBottom: '1px solid var(--border)', paddingBottom: '12px' }}>
          <Link2 size={20} style={{ color: 'var(--accent)' }} />
          <h3 style={{ fontSize: '16px', margin: 0 }}>Outbound Webhooks</h3>
        </div>

        <p style={{ color: 'var(--text-secondary)', fontSize: '13px', marginBottom: '20px' }}>
          RelayHub pushes HMAC-SHA256 signed HTTP POST events to your server whenever a notification reaches a final status (<code style={{ color: 'var(--success)' }}>notification.delivered</code> or <code style={{ color: 'var(--danger)' }}>notification.failed</code>).
        </p>

        {revealedSecret && (
          <SecretReveal
            title="Save Webhook Secret"
            warning="Save your HMAC webhook secret now. It will not be shown in full again."
            secret={revealedSecret}
            onDismiss={() => setRevealedSecret(null)}
          />
        )}

        <form onSubmit={handleSaveWebhook} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div className="form-group" style={{ margin: 0 }}>
            <label className="form-label">Webhook Endpoint URL (HTTPS Only)</label>
            <div style={{ display: 'flex', gap: '10px' }}>
              <input
                type="url"
                className="form-input font-mono"
                placeholder="https://api.yourdomain.com/webhooks/relayhub"
                value={inputUrl}
                onChange={(e) => setInputUrl(e.target.value)}
                required
              />
              <button type="submit" className="btn btn-primary" disabled={submitting}>
                <Save size={16} />
                <span>{submitting ? 'Saving...' : 'Save Endpoint'}</span>
              </button>
              {currentUrl && (
                <button
                  type="button"
                  className="btn btn-danger"
                  onClick={() => setShowRemoveConfirm(true)}
                  title="Remove Webhook"
                >
                  <Trash2 size={16} />
                </button>
              )}
            </div>
            <span className="form-hint">
              {currentUrl ? `Active endpoint: ${currentUrl}` : 'No webhook currently configured.'}
            </span>
          </div>
        </form>
      </div>

      {/* Code Snippet Verification Panel */}
      <div className="card">
        <div
          onClick={() => setShowSnippets(!showSnippets)}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            cursor: 'pointer',
            userSelect: 'none'
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Code2 size={18} style={{ color: 'var(--accent)' }} />
            <h4 style={{ fontSize: '14px', margin: 0 }}>How to Verify Webhook Signatures</h4>
          </div>
          {showSnippets ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
        </div>

        {showSnippets && (
          <div style={{ marginTop: '16px', borderTop: '1px solid var(--border)', paddingTop: '16px' }}>
            <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '12px' }}>
              Every webhook request includes an <code style={{ color: 'var(--accent)' }}>X-RelayHub-Signature</code> header containing an HMAC-SHA256 signature calculated over the raw JSON payload using your webhook secret.
            </p>

            <pre style={{
              backgroundColor: 'var(--bg)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              padding: '14px',
              fontSize: '12px',
              color: 'var(--text-primary)',
              overflowX: 'auto'
            }}>
{`// Go Signature Verification Example
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}`}
            </pre>
          </div>
        )}
      </div>

      {/* Recent Deliveries Log */}
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Send size={16} style={{ color: 'var(--accent)' }} />
            <h4 style={{ fontSize: '14px', margin: 0 }}>Recent Webhook Deliveries</h4>
          </div>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            Showing latest {deliveries.length} attempts
          </span>
        </div>

        {loadingConfig && deliveries.length === 0 ? (
          <div style={{ padding: '30px', textAlign: 'center', color: 'var(--text-secondary)' }}>
            Loading webhook deliveries...
          </div>
        ) : deliveries.length === 0 ? (
          <div style={{ padding: '30px', textAlign: 'center', color: 'var(--text-faint)' }}>
            No webhook delivery attempts recorded yet.
          </div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                  <th style={{ padding: '10px 12px' }}>Request ID</th>
                  <th style={{ padding: '10px 12px' }}>HTTP Status</th>
                  <th style={{ padding: '10px 12px' }}>Attempt</th>
                  <th style={{ padding: '10px 12px' }}>Result</th>
                  <th style={{ padding: '10px 12px' }}>Timestamp</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map((del) => (
                  <tr key={del.id} style={{ borderBottom: '1px solid var(--border-strong)' }}>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', color: 'var(--accent)', fontSize: '12px' }}>
                      {del.notification_request_id}
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)' }}>
                      <code>{del.status_code || '0'}</code>
                    </td>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)' }}>
                      {del.attempt}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <StatusBadge status={del.success ? 'success' : 'failed'} />
                    </td>
                    <td style={{ padding: '10px 12px', color: 'var(--text-faint)', fontSize: '12px' }}>
                      {new Date(del.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Remove Confirmation Dialog */}
      <ConfirmDialog
        isOpen={showRemoveConfirm}
        title="Remove Webhook Endpoint"
        message="Are you sure you want to remove your outbound webhook endpoint? RelayHub will stop sending delivery event payloads to your server."
        confirmText="Remove Webhook"
        isDanger={true}
        loading={removing}
        onConfirm={handleRemoveWebhook}
        onCancel={() => setShowRemoveConfirm(false)}
      />
    </div>
  );
};
