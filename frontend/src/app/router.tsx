import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom';
import { RouterProviders } from './providers';
import { useAuth } from '@/hooks/useAuth';
import { LoginPage } from '@/pages/LoginPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { MetersPage } from '@/pages/MetersPage';
import { MeterDetailPage } from '@/pages/MeterDetailPage';
import { AnomaliesPage } from '@/pages/AnomaliesPage';
import { AnomalyDetailPage } from '@/pages/AnomalyDetailPage';
import { NotFoundPage } from '@/pages/NotFoundPage';

function RequireAuth() {
  const { isAuthenticated } = useAuth();
  const location = useLocation();
  if (!isAuthenticated) {
    const next = `${location.pathname}${location.search}`;
    return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />;
  }
  return <Outlet />;
}

function RedirectIfAuth() {
  const { isAuthenticated } = useAuth();
  if (isAuthenticated) return <Navigate to="/" replace />;
  return <Outlet />;
}

export function AppRoutes() {
  return (
    <RouterProviders>
      <Routes>
        <Route element={<RedirectIfAuth />}>
          <Route path="/login" element={<LoginPage />} />
        </Route>
        <Route element={<RequireAuth />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/meters" element={<MetersPage />} />
          <Route path="/meters/:meterId" element={<MeterDetailPage />} />
          <Route path="/anomalies" element={<AnomaliesPage />} />
          <Route path="/anomalies/:id" element={<AnomalyDetailPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </RouterProviders>
  );
}
