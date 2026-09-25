// Tests de render de cada página con los mocks (RF-F-03 … RF-F-09) y casos borde clave.
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { API_URL } from '@/api/client';
import { storage } from '@/lib/storage';
import { server } from '@/mocks/server';
import { ANOMALY_SEEDS } from '@/mocks/anomalies';
import { loginAsDemo, renderRoute } from '@/test/render';
import { LoginPage } from './LoginPage';
import { DashboardPage } from './DashboardPage';
import { MetersPage } from './MetersPage';
import { MeterDetailPage } from './MeterDetailPage';
import { AnomaliesPage } from './AnomaliesPage';
import { AnomalyDetailPage } from './AnomalyDetailPage';
import { NotFoundPage } from './NotFoundPage';

const ALL_ROUTES = [
  { path: '/login', element: <LoginPage /> },
  { path: '/', element: <DashboardPage /> },
  { path: '/meters', element: <MetersPage /> },
  { path: '/meters/:meterId', element: <MeterDetailPage /> },
  { path: '/anomalies', element: <AnomaliesPage /> },
  { path: '/anomalies/:id', element: <AnomalyDetailPage /> },
  { path: '*', element: <NotFoundPage /> },
];

/** Ejecuta un análisis completo contra los mocks (POST + espera a COMPLETED). */
async function runMockAnalysis(): Promise<void> {
  const res = await fetch(`${API_URL}/ai/analyze`, { method: 'POST', headers: { Authorization: `Bearer ${storage.getToken()}` } });
  const { analysis_id } = (await res.json()) as { analysis_id: string };
  await waitFor(
    async () => {
      const r = await fetch(`${API_URL}/ai/analysis/${analysis_id}`, { headers: { Authorization: `Bearer ${storage.getToken()}` } });
      const a = (await r.json()) as { status: string };
      expect(a.status).toBe('COMPLETED');
    },
    { timeout: 10_000, interval: 200 },
  );
}

describe('LoginPage (RF-F-03)', () => {
  it('botón deshabilitado con campos vacíos; 401 muestra "Credenciales inválidas"', async () => {
    const user = userEvent.setup();
    renderRoute('/login', ALL_ROUTES);
    const button = screen.getByRole('button', { name: 'Iniciar sesión' });
    expect(button).toBeDisabled();
    expect(screen.getByPlaceholderText('analista@energy.local')).toHaveValue('');
    await user.type(screen.getByLabelText('Correo electrónico'), 'analista@energy.local');
    await user.type(screen.getByLabelText('Contraseña'), 'incorrecta');
    await user.click(button);
    expect(await screen.findByText('Credenciales inválidas')).toBeInTheDocument();
    expect(storage.getToken()).toBeNull();
  });

  it('login correcto guarda el token y navega al dashboard', async () => {
    const user = userEvent.setup();
    renderRoute('/login', ALL_ROUTES);
    await user.type(screen.getByLabelText('Correo electrónico'), 'analista@energy.local');
    await user.type(screen.getByLabelText('Contraseña'), 'Demo1234!');
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }));
    await waitFor(() => expect(storage.getToken()).not.toBeNull());
    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument();
  });

  it('CB-F-01: ?expired=1 muestra el aviso de sesión expirada', () => {
    renderRoute('/login?expired=1', ALL_ROUTES);
    expect(screen.getByText('Tu sesión expiró. Inicia sesión de nuevo.')).toBeInTheDocument();
  });
});

describe('DashboardPage (RF-F-04)', () => {
  it('sin análisis: tiles en 0/—, texto de vacío y chip "Sin análisis"', async () => {
    loginAsDemo();
    renderRoute('/', ALL_ROUTES);
    expect(await screen.findByText('155.250,9')).toBeInTheDocument();
    expect(screen.getByText('Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías.')).toBeInTheDocument();
    expect(screen.getAllByText('Sin análisis').length).toBeGreaterThan(0);
    expect(await screen.findByRole('list', { name: 'Estado de la red' })).toBeInTheDocument();
  });

  it('con análisis: 4 anomalías, 2 alta prioridad en rojo, 93 % y top 3 con M-109 primero', async () => {
    loginAsDemo();
    await runMockAnalysis();
    renderRoute('/', ALL_ROUTES);
    expect(await screen.findByText('93 %')).toBeInTheDocument();
    const high = screen.getByText('2', { selector: 'div' });
    expect(high).toHaveStyle({ color: 'rgb(239, 68, 68)' });
    const rows = screen.getAllByRole('row').filter((r) => within(r).queryByRole('link'));
    expect(within(rows[0]).getByRole('link')).toHaveTextContent('M-109');
  });

  it('error de red muestra "No se pudo cargar el dashboard"', async () => {
    loginAsDemo();
    server.use(http.get(`${API_URL}/dashboard/summary`, () => HttpResponse.json({ error: { code: 'INTERNAL_ERROR', message: 'x', details: null } }, { status: 500 })));
    renderRoute('/', ALL_ROUTES);
    expect(await screen.findByText('No se pudo cargar el dashboard')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeInTheDocument();
  });
});

describe('MetersPage (RF-F-05)', () => {
  it('renderiza 12 medidores con formato es-CO y filtra por Críticas', async () => {
    const user = userEvent.setup();
    loginAsDemo();
    renderRoute('/meters', ALL_ROUTES);
    expect(await screen.findByText('12 medidores')).toBeInTheDocument();
    expect(screen.getByText('2.207,6')).toBeInTheDocument();
    expect(screen.getByText('+110,5 %')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Críticas' }));
    expect(await screen.findByText('Ningún medidor coincide con los filtros')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Limpiar filtros' })).toBeInTheDocument();
  });

  it('tras el análisis M-109 es CRÍTICO y va primero', async () => {
    loginAsDemo();
    await runMockAnalysis();
    renderRoute('/meters', ALL_ROUTES);
    await screen.findByText('12 medidores');
    const firstRow = screen.getAllByRole('row')[1];
    expect(within(firstRow).getByText('M-109')).toBeInTheDocument();
    expect(within(firstRow).getByText('CRÍTICO')).toBeInTheDocument();
  });
});

describe('MeterDetailPage (RF-F-06)', () => {
  it('muestra las 4 cards de M-109 y los eventos', async () => {
    loginAsDemo();
    renderRoute('/meters/M-109', ALL_ROUTES);
    expect(await screen.findByText('2.207,6 kWh')).toBeInTheDocument();
    expect(screen.getByText('1.048,8 kWh/día')).toBeInTheDocument();
    expect(screen.getByText('+110,5 %')).toBeInTheDocument();
    expect(await screen.findByText('No operational event reported')).toBeInTheDocument();
    expect(screen.getByText('Sin anomalía vigente')).toBeInTheDocument();
  });

  it('404 muestra "El medidor M-999 no existe"', async () => {
    loginAsDemo();
    renderRoute('/meters/M-999', ALL_ROUTES);
    expect(await screen.findByText('El medidor M-999 no existe')).toBeInTheDocument();
  });
});

describe('AnomaliesPage (RF-F-07)', () => {
  it('sin análisis muestra el texto de vacío', async () => {
    loginAsDemo();
    renderRoute('/anomalies', ALL_ROUTES);
    expect(await screen.findByText('Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías.')).toBeInTheDocument();
  });

  it('orden M-109, M-112, M-104, M-106 y filtro por tipo', async () => {
    const user = userEvent.setup();
    loginAsDemo();
    await runMockAnalysis();
    renderRoute('/anomalies', ALL_ROUTES);
    await screen.findByText('M-109');
    const ids = screen.getAllByRole('row').slice(1).map((r) => within(r).getAllByRole('cell')[1].textContent);
    expect(ids).toEqual(['M-109', 'M-112', 'M-104', 'M-106']);
    const firstRow = screen.getAllByRole('row')[1];
    expect(within(firstRow).getByText('Anomalía real')).toBeInTheDocument();
    expect(within(firstRow).getByText('Alta')).toBeInTheDocument();
    expect(screen.getAllByText('98 %').length).toBe(2);
    await user.selectOptions(screen.getByLabelText('Tipo'), 'DATA_QUALITY');
    await waitFor(() => expect(screen.queryByText('M-109')).not.toBeInTheDocument());
    expect(screen.getByText('M-112')).toBeInTheDocument();
  });
});

describe('AnomalyDetailPage (RF-F-08)', () => {
  it('7 encabezados en orden, comparación, sin eventos, badge IA y cambio de estado', async () => {
    const user = userEvent.setup();
    loginAsDemo();
    await runMockAnalysis();
    renderRoute(`/anomalies/${ANOMALY_SEEDS[0].id}`, ALL_ROUTES);
    await screen.findByRole('heading', { name: 'Evidencia' });
    const h2 = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent);
    expect(h2).toEqual(['Qué encontró la IA', 'Variables que cambiaron', 'Comparación contra baseline', 'Eventos relacionados', 'Severidad y confianza', 'Acción recomendada', 'Evidencia']);
    const corriente = screen.getByText('Corriente', { selector: 'div' }).parentElement as HTMLElement;
    expect(within(corriente).getByText('420,5 A')).toBeInTheDocument();
    expect(within(corriente).getByText('+109,7 %')).toBeInTheDocument();
    expect(within(corriente).getByText('base 200,5 A')).toBeInTheDocument();
    const pf = screen.getByText('Factor de potencia', { selector: 'div' }).parentElement as HTMLElement;
    expect(within(pf).getByText('0,744')).toBeInTheDocument();
    expect(within(pf).getByText('−20,8 %')).toBeInTheDocument();
    expect(within(pf).getByText('base 0,939')).toBeInTheDocument();
    expect(screen.getByText('Sin eventos relacionados')).toBeInTheDocument();
    expect(screen.getByText('Explicación generada por IA')).toBeInTheDocument();
    expect(screen.getAllByRole('row').length).toBeGreaterThanOrEqual(7);

    await user.click(screen.getByRole('button', { name: 'Marcar en investigación' }));
    expect(await screen.findByText('Estado actualizado a En investigación')).toBeInTheDocument();
    expect(screen.getByText('Estado: En investigación')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Marcar en investigación' })).toBeDisabled();
  });

  it('anomalía inexistente muestra "La anomalía no existe"', async () => {
    loginAsDemo();
    renderRoute('/anomalies/no-existe', ALL_ROUTES);
    expect(await screen.findByText('La anomalía no existe')).toBeInTheDocument();
  });
});

describe('Rutas (RF-F-02)', () => {
  it('ruta desconocida muestra "Página no encontrada"', async () => {
    loginAsDemo();
    renderRoute('/ruta-inexistente', ALL_ROUTES);
    expect(await screen.findByRole('heading', { name: 'Página no encontrada' })).toBeInTheDocument();
    expect(screen.getByText('Página no encontrada', { selector: 'p' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Volver al dashboard' })).toHaveAttribute('href', '/');
  });
});
