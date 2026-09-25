import { NavLink } from 'react-router-dom';
import { Activity, LayoutDashboard, LogOut, Radar, Zap } from 'lucide-react';
import { useAuth } from '@/hooks/useAuth';

const NAV = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/meters', label: 'Medidores', icon: Activity, end: false },
  { to: '/anomalies', label: 'Anomalías IA', icon: Radar, end: false },
];

export function Sidebar() {
  const { user, logout } = useAuth();
  return (
    <aside className="fixed inset-y-0 left-0 flex w-[240px] flex-col border-r border-line bg-app text-[#e4e4e7]">
      <div className="flex h-16 items-center gap-2.5 px-5">
        <span className="flex h-7 w-7 items-center justify-center rounded bg-mint/15">
          <Zap size={16} strokeWidth={1.5} className="text-mint" aria-hidden="true" />
        </span>
        <span className="text-md font-semibold tracking-tight text-ink">Energy Management</span>
      </div>
      <nav className="mt-2 flex flex-col gap-0.5 px-3" aria-label="Navegación principal">
        {NAV.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              `relative flex h-10 items-center gap-3 rounded px-3 text-sm font-medium transition-colors duration-150 ${
                isActive ? 'bg-raised text-ink' : 'text-ink-2 hover:bg-hover hover:text-ink'
              }`
            }
          >
            {({ isActive }) => (
              <>
                {isActive && <span className="absolute inset-y-2 left-0 w-[3px] rounded-r bg-mint" aria-hidden="true" />}
                <Icon size={16} strokeWidth={1.5} aria-hidden="true" className={isActive ? 'text-mint' : ''} />
                {label}
              </>
            )}
          </NavLink>
        ))}
      </nav>
      <div className="mt-auto border-t border-line px-5 py-4">
        <div className="text-sm font-medium text-ink">{user?.name ?? 'Analista'}</div>
        <div className="truncate text-xs text-ink-2">{user?.email}</div>
        <button type="button" onClick={logout} className="mt-3 inline-flex items-center gap-2 text-sm text-ink-2 hover:text-ink">
          <LogOut size={14} strokeWidth={1.5} aria-hidden="true" />
          Cerrar sesión
        </button>
      </div>
    </aside>
  );
}
