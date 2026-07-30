import React from 'react';

export interface ToastMessage {
  id: string;
  message: string;
  type: 'success' | 'error';
}

interface ToastProps {
  toasts: ToastMessage[];
  onDismiss: (id: string) => void;
}

export const ToastContainer: React.FC<ToastProps> = ({ toasts, onDismiss }) => {
  return (
    <div style={{
      position: 'fixed',
      bottom: '24px',
      right: '24px',
      display: 'flex',
      flexDirection: 'column',
      gap: '10px',
      zIndex: 9999,
      pointerEvents: 'none'
    }}>
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`toast ${toast.type === 'success' ? 'toast-success' : 'toast-error'}`}
          style={{ pointerEvents: 'auto', cursor: 'pointer' }}
          onClick={() => onDismiss(toast.id)}
        >
          <span>{toast.type === 'success' ? '✅' : '❌'}</span>
          <span style={{ fontSize: '13px' }}>{toast.message}</span>
        </div>
      ))}
    </div>
  );
};
