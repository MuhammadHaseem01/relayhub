import React, { useState, useEffect } from 'react';
import { sendNotification, getNotificationStatus, type SendNotifyRequest, type NotificationLog } from '../api/client';

interface SendPageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const SendPage: React.FC<SendPageProps> = ({ onShowToast }) => {
  const [channel, setChannel] = useState<'email' | 'discord' | 'smtp' | 'auto'>('discord');
  const [recipient, setRecipient] = useState('');
  const [discordRecipient, setDiscordRecipient] = useState('');
  const [emailRecipient, setEmailRecipient] = useState('');
  const [message, setMessage] = useState('Hello from RelayHub Dashboard! 🚀');
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [sendAt, setSendAt] = useState('');

  const [submitting, setSubmitting] = useState(false);
  const [rawResponse, setRawResponse] = useState<string | null>(null);
  const [lastRequestId, setLastRequestId] = useState<string | null>(null);

  // Live status tracking
  const [trackedStatus, setTrackedStatus] = useState<NotificationLog | null>(null);
  const [polling, setPolling] = useState(false);

  const handleGenerateUUID = () => {
    setIdempotencyKey(`idem-${crypto.randomUUID()}`);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setRawResponse(null);
    setTrackedStatus(null);
    setPolling(false);

    const payload: SendNotifyRequest = {
      channel,
      message: message.trim(),
    };

    if (channel === 'auto') {
      payload.discord_recipient = discordRecipient.trim();
      payload.email_recipient = emailRecipient.trim();
    } else {
      payload.recipient = recipient.trim();
    }

    if (idempotencyKey.trim()) {
      payload.idempotency_key = idempotencyKey.trim();
    }

    if (sendAt) {
      // Convert datetime-local value to ISO UTC string
      payload.send_at = new Date(sendAt).toISOString();
    }

    const res = await sendNotification(payload);
    setSubmitting(false);

    setRawResponse(JSON.stringify(res, null, 2));

    if (res.success && res.data?.request_id) {
      setLastRequestId(res.data.request_id);
      onShowToast(`Notification enqueued (status: ${res.data.status})`, 'success');
      startPollingStatus(res.data.request_id);
    } else {
      onShowToast(res.error || 'Failed to send notification', 'error');
    }
  };

  const startPollingStatus = (reqId: string) => {
    setLastRequestId(reqId);
    setPolling(true);
  };

  // Status Polling Effect
  useEffect(() => {
    if (!polling || !lastRequestId) return;

    let isMounted = true;

    const poll = async () => {
      const res = await getNotificationStatus(lastRequestId);
      if (!isMounted) return;

      if (res.success && res.data) {
        setTrackedStatus(res.data);
        const finalStates = ['delivered', 'failed', 'dead_letter', 'cancelled'];
        if (finalStates.includes(res.data.status)) {
          setPolling(false);
        }
      }
    };

    poll();
    const interval = setInterval(poll, 1500);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [polling, lastRequestId]);

  const getStatusBadgeClass = (status?: string) => {
    switch (status) {
      case 'delivered': return 'badge-success';
      case 'failed': return 'badge-danger';
      case 'dead_letter': return 'badge-dead';
      default: return 'badge-warning';
    }
  };

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px', alignItems: 'start' }}>
      {/* Send Form Column */}
      <div className="card">
        <h3 style={{ fontSize: '16px', marginBottom: '16px', borderBottom: '1px solid var(--border)', paddingBottom: '10px' }}>
          Dispatch Notification
        </h3>

        <form onSubmit={handleSubmit}>
          {/* Channel Selector */}
          <div className="form-group">
            <label className="form-label">Channel Provider</label>
            <select
              className="form-select"
              value={channel}
              onChange={(e) => setChannel(e.target.value as any)}
            >
              <option value="discord">Discord Webhook (channel=discord)</option>
              <option value="email">Email via Resend (channel=email)</option>
              <option value="smtp">Email via Plain SMTP (channel=smtp)</option>
              <option value="auto">Auto Fallback (channel=auto — Discord then Email)</option>
            </select>
          </div>

          {/* Dynamic Recipient Fields */}
          {channel === 'auto' ? (
            <>
              <div className="form-group">
                <label className="form-label">Primary Discord Webhook URL</label>
                <input
                  type="url"
                  className="form-input font-mono"
                  placeholder="https://discord.com/api/webhooks/..."
                  value={discordRecipient}
                  onChange={(e) => setDiscordRecipient(e.target.value)}
                  required
                />
              </div>
              <div className="form-group">
                <label className="form-label">Fallback Email Address</label>
                <input
                  type="email"
                  className="form-input font-mono"
                  placeholder="user@example.com"
                  value={emailRecipient}
                  onChange={(e) => setEmailRecipient(e.target.value)}
                  required
                />
              </div>
            </>
          ) : (
            <div className="form-group">
              <label className="form-label">
                Recipient {channel === 'discord' ? '(Webhook URL)' : '(Email Address)'}
              </label>
              <input
                type={channel === 'discord' ? 'url' : 'email'}
                className="form-input font-mono"
                placeholder={
                  channel === 'discord'
                    ? 'https://discord.com/api/webhooks/...'
                    : 'user@example.com'
                }
                value={recipient}
                onChange={(e) => setRecipient(e.target.value)}
                required
              />
            </div>
          )}

          {/* Message Body */}
          <div className="form-group">
            <label className="form-label">Message Content</label>
            <textarea
              className="form-textarea"
              rows={4}
              placeholder="Enter message text..."
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              required
            />
            <span className="form-hint">{message.length} characters</span>
          </div>

          {/* Idempotency Key */}
          <div className="form-group">
            <label className="form-label">Idempotency Key (Optional)</label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <input
                type="text"
                className="form-input font-mono"
                placeholder="idem-unique-key-123"
                value={idempotencyKey}
                onChange={(e) => setIdempotencyKey(e.target.value)}
              />
              <button type="button" className="btn btn-sm" onClick={handleGenerateUUID}>
                UUID
              </button>
            </div>
            <span className="form-hint">Prevents duplicate sending if retried within 24h.</span>
          </div>

          {/* Scheduled For (send_at) */}
          <div className="form-group">
            <label className="form-label">Schedule Send (send_at - Optional)</label>
            <input
              type="datetime-local"
              className="form-input font-mono"
              value={sendAt}
              onChange={(e) => setSendAt(e.target.value)}
            />
            <span className="form-hint">Leave blank to send immediately via Redis stream.</span>
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            style={{ width: '100%', marginTop: '8px' }}
            disabled={submitting}
          >
            {submitting ? 'Dispatching...' : '🚀 Send Notification'}
          </button>
        </form>
      </div>

      {/* Response & Live Inspector Column */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
        {/* Live Status Tracker Panel */}
        {lastRequestId && (
          <div className="card" style={{ borderColor: 'var(--border-strong)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
              <h4 style={{ fontSize: '14px', margin: 0 }}>Live Delivery Inspector</h4>
              {polling ? (
                <span className="badge badge-warning" style={{ animation: 'pulse 1.5s infinite' }}>
                  ● Polling status...
                </span>
              ) : (
                <button
                  className="btn btn-sm"
                  onClick={() => startPollingStatus(lastRequestId)}
                >
                  Refresh Status
                </button>
              )}
            </div>

            <div style={{ fontSize: '12px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
              <div>
                <span style={{ color: 'var(--text-secondary)' }}>Request ID: </span>
                <code style={{ color: 'var(--accent)' }}>{lastRequestId}</code>
              </div>

              {trackedStatus && (
                <>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span style={{ color: 'var(--text-secondary)' }}>Current Status: </span>
                    <span className={`badge ${getStatusBadgeClass(trackedStatus.status)}`}>
                      {trackedStatus.status}
                    </span>
                  </div>
                  <div>
                    <span style={{ color: 'var(--text-secondary)' }}>Delivery Attempts: </span>
                    <code>{trackedStatus.attempts}</code>
                  </div>
                  {trackedStatus.fallback_used && (
                    <div style={{ color: 'var(--warning)' }}>
                      ⚠️ Fallback channel was utilized for this delivery
                    </div>
                  )}
                  {trackedStatus.error_message && (
                    <div style={{
                      backgroundColor: 'rgba(248, 81, 73, 0.1)',
                      color: 'var(--danger)',
                      padding: '8px',
                      borderRadius: 'var(--radius-sm)',
                      marginTop: '4px'
                    }}>
                      Error: {trackedStatus.error_message}
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        )}

        {/* Raw Response Panel */}
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
            <h4 style={{ fontSize: '14px', margin: 0 }}>Raw API Response</h4>
            {rawResponse && (
              <button
                className="btn btn-sm"
                onClick={() => {
                  navigator.clipboard.writeText(rawResponse);
                  onShowToast('JSON copied to clipboard', 'success');
                }}
              >
                Copy JSON
              </button>
            )}
          </div>

          <pre style={{
            backgroundColor: 'var(--bg)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            padding: '12px',
            fontSize: '12px',
            color: rawResponse ? 'var(--accent)' : 'var(--text-faint)',
            overflowX: 'auto',
            minHeight: '180px',
            maxHeight: '360px'
          }}>
            {rawResponse || '// API JSON response will appear here after submission...'}
          </pre>
        </div>
      </div>
    </div>
  );
};
