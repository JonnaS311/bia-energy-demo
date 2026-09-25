// Recorre el guion de demo contra el dev server (mocks) y captura cada pantalla a 1440 px.
// Uso: node scripts/screenshots.mjs [outDir] — requiere playwright-core y Edge o Chrome instalados.
import { chromium } from 'playwright-core';
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const BASE = process.env.BASE_URL || 'http://localhost:5173';
const out = resolve(process.argv[2] || '../.impeccable/review');
mkdirSync(out, { recursive: true });

const channel = process.env.BROWSER_CHANNEL || 'msedge';
const browser = await chromium.launch({ channel, headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1, locale: 'es-CO', timezoneId: 'America/Bogota' });
const page = await context.newPage();
const consoleIssues = [];
page.on('console', (msg) => {
  if (msg.type() === 'error' || msg.type() === 'warning') consoleIssues.push(`[${msg.type()}] ${msg.text()}`);
});
page.on('pageerror', (err) => consoleIssues.push(`[pageerror] ${err.message}`));

const shot = async (name, fullPage = true) => {
  await page.waitForTimeout(400);
  await page.screenshot({ path: resolve(out, `${name}.png`), fullPage });
  console.log(`ok ${name}.png`);
};

const strip = () => page.getByRole('heading', { name: 'Estado de la red' }).waitFor();

await page.goto(`${BASE}/`, { waitUntil: 'networkidle' });
await page.waitForURL(/\/login/);
await shot('01-login', false);

await page.locator('#email').fill('analista@energy.local');
await page.locator('#password').fill('Demo1234!');
await page.getByRole('button', { name: 'Iniciar sesión' }).click();
await page.waitForURL(`${BASE}/`);
await strip();
await page.waitForTimeout(800);
await shot('02-dashboard-sin-analisis');

await page.getByRole('link', { name: 'Medidores' }).click();
await page.waitForURL(/\/meters$/);
await page.getByText('12 medidores').waitFor();
await shot('03-medidores');

await page.getByRole('button', { name: 'Críticas' }).click();
await page.getByText('Ningún medidor coincide con los filtros').waitFor();
await shot('03b-medidores-criticas-vacio', false);
await page.getByRole('button', { name: 'Todos' }).click();

await page.getByRole('cell', { name: /^M-109/ }).first().click();
await page.waitForURL(/\/meters\/M-109/);
await page.getByText('Eventos del medidor').waitFor();
await page.waitForTimeout(1000);
await shot('04-detalle-M-109');

await page.getByRole('button', { name: 'Run AI Analysis' }).click();
await page.getByRole('dialog').waitFor();
await page.waitForTimeout(1500);
await shot('05-analisis-en-curso', false);
await page.getByRole('button', { name: 'Ver anomalías' }).waitFor({ timeout: 60_000 });
await shot('05b-analisis-completado', false);
await page.getByRole('button', { name: 'Ver anomalías' }).click();
await page.waitForURL(/\/anomalies$/);
await page.getByRole('cell', { name: 'M-109' }).waitFor();
await shot('06-anomalias');

await page.getByRole('cell', { name: 'M-109' }).click();
await page.waitForURL(/\/anomalies\//);
await page.getByRole('heading', { name: 'Evidencia' }).waitFor();
await page.waitForTimeout(1000);
await shot('07-investigacion-M-109');

await page.getByRole('button', { name: 'Marcar en investigación' }).click();
await page.getByText('Estado actualizado a En investigación').waitFor();
await shot('08-accion', false);

await page.getByRole('link', { name: 'Dashboard' }).click();
await page.waitForURL(`${BASE}/`);
await strip();
const toastClose = page.getByRole('button', { name: 'Cerrar notificación' });
if (await toastClose.count()) await toastClose.first().click();
await page.waitForTimeout(1500);
await shot('desktop');

await page.getByRole('link', { name: 'Medidores' }).click();
await page.getByText('12 medidores').waitFor();
await shot('09-medidores-post-analisis');

await page.goto(`${BASE}/ruta-inexistente`, { waitUntil: 'networkidle' });
await shot('10-404', false);

await page.setViewportSize({ width: 390, height: 844 });
await page.goto(`${BASE}/`, { waitUntil: 'networkidle' });
await page.waitForTimeout(800);
await shot('mobile', false);

console.log(`\nConsola: ${consoleIssues.length} incidencias`);
consoleIssues.forEach((c) => console.log('  ' + c));
await browser.close();
