import React, { useState, useEffect } from 'react';
import {
  HeartPulse,
  RefreshCw,
  Pause,
  Play,
  MessageSquare,
  Mail,
  Server,
  AlertTriangle,
  CheckCircle2
} from 'lucide-react';
import { getProviderHealth, type ProviderHealthResponse } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';

interface ProviderHealthPageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const ProviderHealthPage: React.FC<ProviderHealthPageProps> = () => {
  const [healthMap, setHealthMap] = useState<ProviderHealthResponse>({
    discord: 'healthy',
    email: 'healthy',
    smtp: 'healthy',
  });
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [countdown, setCountdown] = useState(5);

  const fetchHealth = async (showLoading = false) => {
    if (showLoading) {
      // optional loading state
    }
    const res = await getProviderHealth();

    if (res.success && res.data) {
      setHealthMap(res.data);
    }
  };

  useEffect(() => {
    fetchHealth(true);
  }, []);

  useEffect(() => {
    if (!autoRefresh) return;

    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          fetchHealth(false);
          return 5;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [autoRefresh]);

  const hasUnhealthy = Object.values(healthMap).some(s => s === 'unhealthy' || s === 'half-open');

  const getProviderIcon = (name: string) => {
    switch (name.toLowerCase()) {
      case 'discord': return <MessageSquare size={24} />;
      case 'email': return <Mail size={24} />;
      case 'smtp': return <Server size={24} />;
      default: return <HeartPulse size={24} />;
    }
  };

  const getProviderDescription = (name: string) => {
    switch (name.toLowerCase()) {
      case 'discord': return 'Outbound Webhooks Engine';
      case 'email': return 'Resend HTTP API Adapter';
      case 'smtp': return 'Standard net/smtp Transport';
      default: return 'Notification Channel Provider';
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header Controls */}
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <HeartPulse size={20} style={{ color: 'var(--success)' }} />
            <h3 style={{ fontSize: '16px', margin: 0 }}>Provider Health & Circuit Breakers</h3>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <button
              className={`btn btn-sm ${autoRefresh ? 'btn-ghost' : ''}`}
              onClick={() => {
                setAutoRefresh(!autoRefresh);
                setCountdown(5);
              }}
            >
              {autoRefresh ? <Pause size={14} /> : <Play size={14} />}
              <span>{autoRefresh ? `Auto (${countdown}s)` : 'Auto-Refresh'}</span>
            </button>

            <button className="btn btn-sm" onClick={() => fetchHealth(true)}>
              <RefreshCw size={14} />
              <span>Refresh</span>
            </button>
          </div>
        </div>
      </div>

      {/* Alert Banner if Any Circuit is Open */}
      {hasUnhealthy && (
        <div style={{
          backgroundColor: 'rgba(210, 153, 34, 0.12)',
          border: '1px solid var(--warning)',
          borderRadius: 'var(--radius-md)',
          padding: '16px',
          display: 'flex',
          alignItems: 'center',
          gap: '12px'
        }}>
          <AlertTriangle size={24} style={{ color: 'var(--warning)', flexShrink: 0 }} />
          <div>
            <h4 style={{ color: 'var(--warning)', margin: 0, fontSize: '14px' }}>Circuit Open Detected</h4>
            <p style={{ fontSize: '12px', color: 'var(--text-primary)', margin: 0, marginTop: '2px' }}>
              One or more providers have experienced consecutive delivery failures. Requests sent to channel="auto" are being routed to fallback channels automatically.
            </p>
          </div>
        </div>
      )}

      {/* Provider Cards Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px' }}>
        {Object.entries(healthMap).map(([name, status]) => {
          const isHealthy = status === 'healthy';
          return (
            <div key={name} className="card card-interactive">
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
                <div style={{
                  padding: '10px',
                  borderRadius: 'var(--radius-md)',
                  backgroundColor: isHealthy ? 'rgba(63, 185, 80, 0.1)' : 'rgba(248, 81, 73, 0.1)',
                  color: isHealthy ? 'var(--success)' : 'var(--danger)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center'
                }}>
                  {getProviderIcon(name)}
                </div>
                <StatusBadge status={status} />
              </div>

              <h4 style={{ fontSize: '18px', marginBottom: '4px', textTransform: 'capitalize' }}>
                {name} Provider
              </h4>
              <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                {getProviderDescription(name)}
              </p>

              <div style={{
                borderTop: '1px solid var(--border-strong)',
                paddingTop: '12px',
                fontSize: '12px',
                color: isHealthy ? 'var(--text-faint)' : 'var(--warning)',
                display: 'flex',
                alignItems: 'center',
                gap: '6px'
              }}>
                {isHealthy ? (
                  <>
                    <CheckCircle2 size={14} style={{ color: 'var(--success)' }} />
                    <span>Circuit closed — operational</span>
                  </>
                ) : (
                  <>
                    <AlertTriangle size={14} />
                    <span>Circuit open — auto fallback active</span>
                  </>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
