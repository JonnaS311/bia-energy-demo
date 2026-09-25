// RF-F-14 — Skeleton, EmptyState, ErrorState reutilizables.
import { Inbox, RefreshCw, WifiOff, type LucideIcon } from 'lucide-react';
import { isApiError } from '@/api/client';

export function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`skeleton ${className}`} aria-hidden="true" />;
}

export function SkeletonRows({ rows, className = 'h-10' }: { rows: number; className?: string }) {
  return (
    <div className="flex flex-col gap-2" aria-busy="true" aria-label="Cargando">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className={className} />
      ))}
    </div>
  );
}

interface EmptyProps {
  text: string;
  icon?: LucideIcon;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
}

export function EmptyState({ text, icon: Icon = Inbox, actionLabel, onAction, className = '' }: EmptyProps) {
  return (
    <div className={`flex flex-col items-center justify-center gap-3 px-6 py-10 text-center ${className}`}>
      <Icon size={28} strokeWidth={1.5} className="text-ink-3" aria-hidden="true" />
      <p className="max-w-md text-sm text-ink-2">{text}</p>
      {actionLabel && onAction && (
        <button type="button" className="btn-secondary h-9" onClick={onAction}>
          {actionLabel}
        </button>
      )}
    </div>
  );
}

interface ErrorProps {
  error: unknown;
  text: string;
  onRetry?: () => void;
  className?: string;
}

/** SI code = UNAUTHORIZED → no renderiza nada (el cliente ya redirige). SI es error de red → texto de conexión. */
export function ErrorState({ error, text, onRetry, className = '' }: ErrorProps) {
  if (isApiError(error) && error.code === 'UNAUTHORIZED') return null;
  const network = isApiError(error) && error.isNetwork;
  const message = network ? 'No se pudo conectar con el servidor' : text;
  return (
    <div role="alert" className={`flex flex-col items-center justify-center gap-3 px-6 py-10 text-center ${className}`}>
      <WifiOff size={28} strokeWidth={1.5} className="text-red" aria-hidden="true" />
      <p className="text-sm text-ink">{message}</p>
      {onRetry && (
        <button type="button" className="btn-secondary h-9" onClick={onRetry}>
          <RefreshCw size={14} strokeWidth={1.5} aria-hidden="true" />
          Reintentar
        </button>
      )}
    </div>
  );
}
