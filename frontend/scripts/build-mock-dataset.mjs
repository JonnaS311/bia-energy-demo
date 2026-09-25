// Genera src/mocks/data/dataset.json a partir de ../data/readings.csv y ../data/events.csv
// (o de la raíz del repo si aún no se movieron). Reproduce los cálculos del spec de data
// (baseline = primeros 7 días, perfil horario, outliers z ≥ 3, promedios eléctricos) para que
// los mocks del navegador muestren los datos reales del dataset. Solo desarrollo.
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, '..', '..');
const candidates = [resolve(root, 'data'), root];
const dataDir = candidates.find((d) => existsSync(resolve(d, 'readings.csv')));
if (!dataDir) {
  console.error('readings.csv no encontrado en data/ ni en la raíz del repositorio');
  process.exit(1);
}

const parseCsv = (path) => {
  const lines = readFileSync(path, 'utf8').replace(/^﻿/, '').split(/\r?\n/).filter(Boolean);
  const header = lines[0].split(',');
  return lines.slice(1).map((l) => {
    const cells = l.split(',');
    return Object.fromEntries(header.map((h, i) => [h.trim(), (cells[i] ?? '').trim()]));
  });
};

const readings = parseCsv(resolve(dataDir, 'readings.csv')).map((r) => ({
  meter_id: r.meter_id,
  timestamp: r.timestamp.replace(' ', 'T') + 'Z',
  consumption_kwh: Number(r.consumption_kwh),
  voltage_v: Number(r.voltage_v),
  current_a: Number(r.current_a),
  power_factor: Number(r.power_factor),
  status: r.status,
}));
const events = parseCsv(resolve(dataDir, 'events.csv')).map((e, i) => ({
  id: i + 1,
  meter_id: e.meter_id,
  timestamp: e.event_timestamp.replace(' ', 'T') + (e.event_timestamp.length === 16 ? ':00Z' : 'Z'),
  type: e.event_type,
  description: e.description,
}));

const r1 = (n) => Math.round(n * 10) / 10;
const r3 = (n) => Math.round(n * 1000) / 1000;
const avg = (a) => (a.length ? a.reduce((s, x) => s + x, 0) / a.length : 0);
const std = (a) => {
  if (a.length < 2) return 0;
  const m = avg(a);
  return Math.sqrt(a.reduce((s, x) => s + (x - m) ** 2, 0) / (a.length - 1));
};
const day = (ts) => ts.slice(0, 10);
const hour = (ts) => Number(ts.slice(11, 13));

const BASELINE_DAYS = 7;
const meterIds = [...new Set(readings.map((r) => r.meter_id))].sort();
const meters = {};
for (const id of meterIds) {
  const rows = readings.filter((r) => r.meter_id === id).sort((a, b) => (a.timestamp < b.timestamp ? -1 : 1));
  const days = [...new Set(rows.map((r) => day(r.timestamp)))].sort();
  const baselineDays = days.slice(0, BASELINE_DAYS);
  const base = rows.filter((r) => baselineDays.includes(day(r.timestamp)));
  const dailyTotals = days.map((d) => ({
    date: d,
    rows: rows.filter((r) => day(r.timestamp) === d),
  }));
  const baselineDaily = avg(baselineDays.map((d) => dailyTotals.find((x) => x.date === d).rows.reduce((s, r) => s + r.consumption_kwh, 0)));
  const profile = Array.from({ length: 24 }, (_, h) => {
    const v = base.filter((r) => hour(r.timestamp) === h).map((r) => r.consumption_kwh);
    return { hour: h, mean_kwh: r1(avg(v)), std_kwh: r1(std(v)), _mean: avg(v), _std: std(v) };
  });
  const lastDay = dailyTotals[dailyTotals.length - 1];
  const current = lastDay.rows.reduce((s, r) => s + r.consumption_kwh, 0);
  const elec = (key) => ({
    current_avg: key === 'power_factor' ? r3(avg(lastDay.rows.map((r) => r[key]))) : r1(avg(lastDay.rows.map((r) => r[key]))),
    baseline_avg: key === 'power_factor' ? r3(avg(base.map((r) => r[key]))) : r1(avg(base.map((r) => r[key]))),
  });
  meters[id] = {
    meter_id: id,
    name: `Medidor ${id}`,
    location: 'Planta principal',
    current_consumption_kwh: r1(current),
    baseline_daily_kwh: r1(baselineDaily),
    variation_pct: r1(((current - baselineDaily) / baselineDaily) * 100),
    period_consumption_kwh: r1(rows.reduce((s, r) => s + r.consumption_kwh, 0)),
    period: { start: days[0], end: days[days.length - 1], readings_count: rows.length },
    electrical: { voltage_v: elec('voltage_v'), current_a: elec('current_a'), power_factor: elec('power_factor') },
    events: events.filter((e) => e.meter_id === id),
    daily: dailyTotals.map((d) => {
      const total = d.rows.reduce((s, r) => s + r.consumption_kwh, 0);
      return {
        date: d.date,
        consumption_kwh: r1(total),
        deviation_pct: r1(((total - baselineDaily) / baselineDaily) * 100),
        voltage_avg: r1(avg(d.rows.map((r) => r.voltage_v))),
        current_avg: r1(avg(d.rows.map((r) => r.current_a))),
        power_factor_avg: r3(avg(d.rows.map((r) => r.power_factor))),
      };
    }),
    baseline_profile: profile.map(({ hour: h, mean_kwh, std_kwh }) => ({ hour: h, mean_kwh, std_kwh })),
    readings: rows.map((r) => {
      const p = profile[hour(r.timestamp)];
      const z = p._std > 0 ? Math.abs((r.consumption_kwh - p._mean) / p._std) : 0;
      return { ...r, is_outlier: z >= 3 };
    }),
  };
}

const dailyConsumption = [...new Set(readings.map((r) => day(r.timestamp)))]
  .sort()
  .map((d) => ({ date: d, consumption_kwh: r1(readings.filter((r) => day(r.timestamp) === d).reduce((s, r) => s + r.consumption_kwh, 0)) }));

const dataset = {
  generated_from: dataDir === root ? 'repo root' : 'data/',
  period: { start: dailyConsumption[0].date, end: dailyConsumption[dailyConsumption.length - 1].date },
  total_consumption_kwh: r1(readings.reduce((s, r) => s + r.consumption_kwh, 0)),
  daily_consumption: dailyConsumption,
  events,
  meters,
};

const out = resolve(here, '..', 'src', 'mocks', 'data');
mkdirSync(out, { recursive: true });
writeFileSync(resolve(out, 'dataset.json'), JSON.stringify(dataset));
console.log(`dataset.json: ${meterIds.length} medidores, ${readings.length} lecturas, ${events.length} eventos → ${resolve(out, 'dataset.json')}`);
