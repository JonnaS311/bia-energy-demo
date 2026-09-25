import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { setUnauthorizedHandler } from '@/api/client';
import { login as apiLogin } from '@/api/auth';
import { storage, type StoredUser } from '@/lib/storage';

interface AuthContextValue {
  token: string | null;
  user: StoredUser | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => storage.getToken());
  const [user, setUser] = useState<StoredUser | null>(() => storage.getUser());
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();

  // CB-F-01: 401 en cualquier petición → limpiar sesión, caché y redirigir una sola vez.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      setToken(null);
      setUser(null);
      queryClient.clear();
      const next = `${location.pathname}${location.search}`;
      navigate(`/login?expired=1&next=${encodeURIComponent(next)}`, { replace: true });
    });
    return () => setUnauthorizedHandler(null);
  }, [navigate, location.pathname, location.search, queryClient]);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await apiLogin(email, password);
      storage.setToken(res.token);
      storage.setUser(res.user);
      setToken(res.token);
      setUser(res.user);
    },
    [],
  );

  const logout = useCallback(() => {
    storage.clearSession();
    storage.clearAnalysisId();
    setToken(null);
    setUser(null);
    queryClient.clear();
    navigate('/login', { replace: true });
  }, [navigate, queryClient]);

  const value = useMemo<AuthContextValue>(
    () => ({ token, user, isAuthenticated: Boolean(token), login, logout }),
    [token, user, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth debe usarse dentro de AuthProvider');
  return ctx;
}
