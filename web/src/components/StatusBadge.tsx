import React from 'react';

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className = '' }) => {
  const getBadgeClass = (s: string) => {
    switch (s.toLowerCase()) {
      case 'delivered':
      case 'healthy':
      case 'success':
      case 'true':
        return 'badge-success';
      case 'queued':
      case 'processing':
      case 'scheduled':
      case 'half-open':
        return 'badge-warning';
      case 'failed':
      case 'unhealthy':
      case 'false':
        return 'badge-danger';
      case 'dead_letter':
      case 'cancelled':
      default:
        return 'badge-dead';
    }
  };

  return (
    <span className={`badge ${getBadgeClass(status)} ${className}`}>
      {status}
    </span>
  );
};
