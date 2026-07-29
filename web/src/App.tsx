import { useState, useEffect } from 'react';
import { getApiKey } from './api/client';
import { AuthModal } from './components/AuthModal';
import { Layout, type NavView } from './components/Layout';
import { DashboardPage } from './pages/DashboardPage';
import { SendPage } from './pages/SendPage';
import { LogsPage } from './pages/LogsPage';
import { PlaceholderPage } from './pages/PlaceholderPage';
import { ToastContainer, type ToastMessage } from './components/Toast';

export function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [currentView, setCurrentView] = useState<NavView>('dashboard');
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  useEffect(() => {
    setIsAuthenticated(Boolean(getApiKey()));
  }, []);

  const showToast = (message: string, type: 'success' | 'error') => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((prev) => [...prev, { id, message, type }]);

    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  };

  const dismissToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  if (!isAuthenticated) {
    return <AuthModal onSuccess={() => setIsAuthenticated(true)} />;
  }

  return (
    <Layout
      currentView={currentView}
      onNavigate={setCurrentView}
      onLogout={() => setIsAuthenticated(false)}
    >
      {currentView === 'dashboard' && (
        <DashboardPage
          onNavigateToSend={() => setCurrentView('send')}
          onShowToast={showToast}
        />
      )}
      {currentView === 'send' && <SendPage onShowToast={showToast} />}
      {currentView === 'logs' && <LogsPage onShowToast={showToast} />}
      {currentView !== 'dashboard' && currentView !== 'send' && currentView !== 'logs' && (
        <PlaceholderPage view={currentView} />
      )}

      <ToastContainer toasts={toasts} onDismiss={dismissToast} />
    </Layout>
  );
}

export default App;
