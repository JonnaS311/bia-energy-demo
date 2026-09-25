import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState, type ReactNode } from 'react';
import { isApiError } from '@/api/client';
import { ToastProvider } from '@/components/Toast';
import { AuthProvider } from '@/hooks/useAuth';
import { AnalysisProvider } from '@/hooks/useAnalysis';

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        // CB-F-02: reintenta 2 veces con 1 s ante error de red; nunca ante 4xx.
        retry: (count, error) => {
          if (isApiError(error) && error.status >= 400 && error.status < 500) return false;
          return count < 2;
        },
        retryDelay: 1000,
      },
    },
  });
}

/** Proveedores que dependen del router (AuthProvider usa useNavigate). */
export function RouterProviders({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <AnalysisProvider>{children}</AnalysisProvider>
    </AuthProvider>
  );
}

export function RootProviders({ children, client }: { children: ReactNode; client?: QueryClient }) {
  const [queryClient] = useState(() => client ?? createQueryClient());
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  );
}
