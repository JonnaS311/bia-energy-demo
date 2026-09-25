// RF-F-03 — Login.
import { useState, type FormEvent } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Zap } from 'lucide-react';
import { isApiError } from '@/api/client';
import { useAuth } from '@/hooks/useAuth';

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const expired = params.get('expired') === '1';
  const next = params.get('next');

  const disabled = !email || !password || submitting;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (disabled) return;
    setSubmitting(true);
    setError(null);
    try {
      await login(email, password);
      const target = next && next.startsWith('/') && !next.startsWith('/login') ? next : '/';
      navigate(target, { replace: true });
    } catch (err) {
      if (isApiError(err) && err.status === 401) setError('Credenciales inválidas');
      else if (isApiError(err) && err.isNetwork) setError('No se pudo conectar con el servidor');
      else setError('No se pudo iniciar sesión');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-canvas px-4">
      <div className="w-[400px]">
        {expired && (
          <p role="status" className="mb-3 rounded border border-amber/40 bg-amber/10 px-3 py-2 text-sm text-amber">
            Tu sesión expiró. Inicia sesión de nuevo.
          </p>
        )}
        <form onSubmit={onSubmit} className="panel px-8 py-8 shadow-panel" noValidate>
          <div className="flex items-center gap-2.5">
            <span className="flex h-8 w-8 items-center justify-center rounded bg-mint/15">
              <Zap size={18} strokeWidth={1.5} className="text-mint" aria-hidden="true" />
            </span>
            <h1 className="text-lg font-semibold tracking-tight text-ink">Energy Management</h1>
          </div>
          <p className="mt-1 text-sm text-ink-2">Inicia sesión para continuar</p>

          <label className="mt-6 block text-xs font-medium text-ink-2" htmlFor="email">
            Correo electrónico
          </label>
          <input id="email" name="email" type="email" autoComplete="username" className="field mt-1.5" placeholder="analista@energy.local" value={email} onChange={(e) => setEmail(e.target.value)} />

          <label className="mt-4 block text-xs font-medium text-ink-2" htmlFor="password">
            Contraseña
          </label>
          <input id="password" name="password" type="password" autoComplete="current-password" className="field mt-1.5" placeholder="Demo1234!" value={password} onChange={(e) => setPassword(e.target.value)} />

          <button type="submit" className="btn-primary mt-6 w-full" disabled={disabled}>
            {submitting ? 'Iniciando…' : 'Iniciar sesión'}
          </button>
          <p className="mt-3 min-h-[20px] text-sm text-red" role="alert" aria-live="assertive">
            {error}
          </p>
        </form>
      </div>
    </main>
  );
}
