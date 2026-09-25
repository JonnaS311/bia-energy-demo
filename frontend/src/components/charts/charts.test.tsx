// RF-F-10 — toda gráfica sin datos muestra "Sin datos para mostrar".
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DailyBarsChart } from './DailyBarsChart';
import { DailyConsumptionChart } from './DailyConsumptionChart';
import { ElectricalChart } from './ElectricalChart';
import { HourlyConsumptionChart } from './HourlyConsumptionChart';
import { niceScale } from './chartTheme';

describe('charts vacías', () => {
  it('DailyConsumptionChart', () => {
    render(<DailyConsumptionChart data={[]} />);
    expect(screen.getByText('Sin datos para mostrar')).toBeInTheDocument();
  });
  it('DailyBarsChart', () => {
    render(<DailyBarsChart data={[]} baselineDailyKwh={null} ariaLabel="x" />);
    expect(screen.getByText('Sin datos para mostrar')).toBeInTheDocument();
  });
  it('HourlyConsumptionChart', () => {
    render(<HourlyConsumptionChart items={[]} baselineProfile={[]} events={[]} meterId="M-101" />);
    expect(screen.getByText('Sin datos para mostrar')).toBeInTheDocument();
  });
  it('ElectricalChart', () => {
    render(<ElectricalChart items={[]} metric="voltage_v" baselineAvg={220} meterId="M-101" />);
    expect(screen.getByText('Sin datos para mostrar')).toBeInTheDocument();
  });
});

describe('niceScale', () => {
  it('produce ticks limpios', () => {
    expect(niceScale(12503).ticks).toEqual([0, 5000, 10000, 15000]);
    expect(niceScale(2207).ticks).toEqual([0, 1000, 2000, 3000]);
    expect(niceScale(0)).toEqual({ domain: [0, 1], ticks: [0, 1] });
  });
});
