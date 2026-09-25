import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react';
import { X } from 'lucide-react';

interface ToastItem {
  id: number;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}

interface ToastContextValue {
  show: (message: string, action?: { label: string; onClick: () => void }) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);
const TOAST_MS = 6000;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const counter = useRef(0);

  const dismiss = useCallback((id: number) => setItems((list) => list.filter((t) => t.id !== id)), []);

  const show = useCallback(
    (message: string, action?: { label: string; onClick: () => void }) => {
      const id = ++counter.current;
      setItems((list) => [...list, { id, message, actionLabel: action?.label, onAction: action?.onClick }]);
      setTimeout(() => dismiss(id), TOAST_MS);
    },
    [dismiss],
  );

  const value = useMemo(() => ({ show }), [show]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed bottom-6 right-6 z-50 flex w-[360px] flex-col gap-2" aria-live="polite" role="status">
        {items.map((t) => (
          <div key={t.id} className="toast-in pointer-events-auto flex items-start gap-3 rounded-md border border-line-strong bg-raised px-4 py-3 text-sm text-ink shadow-panel">
            <span className="flex-1">{t.message}</span>
            {t.actionLabel && (
              <button
                type="button"
                className="font-semibold text-mint hover:text-mint-bright"
                onClick={() => {
                  t.onAction?.();
                  dismiss(t.id);
                }}
              >
                {t.actionLabel}
              </button>
            )}
            <button type="button" aria-label="Cerrar notificación" className="text-ink-2 hover:text-ink" onClick={() => dismiss(t.id)}>
              <X size={16} strokeWidth={1.5} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast debe usarse dentro de ToastProvider');
  return ctx;
}
