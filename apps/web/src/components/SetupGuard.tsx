import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { api } from '../lib/api';
import { useAuth } from '../lib/auth';
import { LoadingScreen } from '../lib/components/ui';

interface SetupStatusResponse {
  state: 'no_db' | 'no_admin' | 'ready';
}

interface SetupGuardProps {
  children: ReactNode;
}

export default function SetupGuard({ children }: SetupGuardProps) {
  const [checking, setChecking] = useState(true);
  const [setupState, setSetupState] = useState<SetupStatusResponse['state'] | null>(null);
  const location = useLocation();
  const [checkedPath, setCheckedPath] = useState('');
  const { status: authStatus } = useAuth();

  useEffect(() => {
    setChecking(true);
    setCheckedPath(location.pathname);

    let active = true;
    api.get<SetupStatusResponse>('/api/setup/status')
      .then((data) => {
        if (!active) return;
        setSetupState(data.state);
        setChecking(false);
      })
      .catch(() => {
        console.error('Setup status check failed');
        if (!active) return;
        setChecking(false);
      });
    return () => {
      active = false;
    };
  }, [location.pathname]);

  if (checking || checkedPath !== location.pathname) return <LoadingScreen />;
  if (setupState && setupState !== 'ready' && location.pathname !== '/setup') {
    return <Navigate to="/setup" replace />;
  }
  if (setupState === 'ready' && location.pathname === '/setup') {
    if (authStatus === 'checking') return <LoadingScreen />;
    const target = authStatus === 'authenticated' ? '/' : '/login';
    return <Navigate to={target} replace />;
  }
  return <>{children}</>;
}
