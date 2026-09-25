// "Estado de la red": la franja de 12 celdas de estado, críticas primero (interacción firma de la consola).
import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { AlertTriangle, Check, Octagon } from 'lucide-react';
import { STROKE, meterStatusColor, palette, tint, variationColor } from '@/lib/colors';
import { formatKwh, formatPct } from '@/lib/format';
import { statusLabel } from '@/lib/labels';
import type { MeterListItem, MeterStatus } from '@/types/api';
import { SkeletonRows } from './States';

interface Props {
  meters: MeterListItem[] | undefined;
  isLoading: boolean;
  /** Marca de tiempo del último análisis completado; dispara el encendido secuencial. */
  igniteKey: number | null;
}

const ICON: Record<MeterStatus, typeof Check> = { OK: Check, ALERT: AlertTriangle, CRITICAL: Octagon };
// Una sola luz: solo la celda crítica recibe halo; alerta es borde ámbar sobre el plano.
const GLOW: Record<MeterStatus, string> = {
  OK: '',
  ALERT: '',
  CRITICAL: 'shadow-glow-red',
};
const IGNITE_DELAY_MS = 120;

export function NetworkStrip({ meters, isLoading, igniteKey }: Props) {
  const [igniting, setIgniting] = useState<Set<string>>(new Set());
  const previous = useRef<Map<string, MeterStatus>>(new Map());

  // Al completarse un análisis, las celdas cuyo estado cambió se encienden en secuencia y luego reposan.
  useEffect(() => {
    if (!meters) return;
    const changed = meters.filter((m) => previous.current.size > 0 && previous.current.get(m.meter_id) !== m.status).map((m) => m.meter_id);
    previous.current = new Map(meters.map((m) => [m.meter_id, m.status]));
    if (!igniteKey || changed.length === 0) return;
    const timers: ReturnType<typeof setTimeout>[] = [];
    changed.forEach((id, i) => {
      timers.push(setTimeout(() => setIgniting((s) => new Set(s).add(id)), i * IGNITE_DELAY_MS));
      timers.push(
        setTimeout(() => setIgniting((s) => {
          const n = new Set(s);
          n.delete(id);
          return n;
        }), i * IGNITE_DELAY_MS + 1000),
      );
    });
    return () => timers.forEach(clearTimeout);
  }, [meters, igniteKey]);

  const description = useMemo(() => {
    if (!meters) return '';
    const critical = meters.filter((m) => m.status === 'CRITICAL').map((m) => m.meter_id);
    const alert = meters.filter((m) => m.status === 'ALERT').map((m) => m.meter_id);
    return `Estado de la red: ${meters.length} medidores; ${critical.length} críticos${critical.length ? ` (${critical.join(', ')})` : ''}; ${alert.length} en alerta${alert.length ? ` (${alert.join(', ')})` : ''}.`;
  }, [meters]);

  if (isLoading || !meters) {
    return (
      <div className="grid grid-cols-12 gap-2" aria-busy="true">
        {Array.from({ length: 12 }).map((_, i) => (
          <SkeletonRows key={i} rows={1} className="h-[76px]" />
        ))}
      </div>
    );
  }

  return (
    <div>
      <p className="sr-only">{description}</p>
      <ul className="grid grid-cols-12 gap-2" aria-label="Estado de la red">
        {meters.map((m) => {
          const color = meterStatusColor[m.status];
          const Icon = ICON[m.status];
          const ignite = igniting.has(m.meter_id);
          return (
            <li key={m.meter_id}>
              <Link
                to={`/meters/${m.meter_id}`}
                className={`block rounded-md border bg-panel px-2.5 py-2 transition-[background-color,border-color] duration-150 hover:bg-hover ${GLOW[m.status]} ${ignite ? 'cell-ignite' : ''}`}
                style={{
                  borderColor: m.status === 'OK' ? palette.line : color,
                  ['--ignite' as string]: color,
                  ['--ignite-glow' as string]: tint(color, 0.55),
                  ['--ignite-rest' as string]: tint(color, m.status === 'OK' ? 0 : 0.25),
                }}
                aria-label={`${m.meter_id}: ${statusLabel(m.status)}, ${formatKwh(m.current_consumption_kwh)} kWh, variación ${formatPct(m.variation_pct)}`}
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-ink">{m.meter_id}</span>
                  <Icon size={12} strokeWidth={STROKE} style={{ color }} aria-hidden="true" />
                </div>
                <div className="tnum mt-1.5 text-sm font-semibold leading-none text-ink">{formatKwh(m.current_consumption_kwh)}</div>
                <div className="tnum mt-1 text-xs leading-none" style={{ color: variationColor(m.variation_pct) }}>
                  {formatPct(m.variation_pct)}
                </div>
              </Link>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
