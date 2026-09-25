# Despliegue gratuito

Orden: base de datos → backend → frontend. Cada paso necesita el dato del anterior.

| Capa | Proveedor | Plan | Límite relevante |
|---|---|---|---|
| PostgreSQL 16 | Neon | Free | 0,5 GB; se suspende tras 5 min sin uso y despierta en < 1 s |
| Backend Go | Render | Free (Docker) | 512 MB; duerme tras 15 min sin tráfico y tarda ~1 min en despertar |
| Frontend | Vercel | Hobby | Sitio estático; `VITE_API_URL` se fija en el build |

Ningún secreto se escribe en el repositorio: todos se cargan en el dashboard de cada proveedor.

## 1. Base de datos (Neon)

1. Crear cuenta en <https://console.neon.tech> (login con GitHub).
2. **New project** → nombre `energy`, Postgres 16, región **AWS US West 2 (Oregon)** (la misma que Render en `render.yaml`), base de datos `energy`.
3. En **Connect**, desactivar **Connection pooling** y copiar la cadena **directa** (host sin `-pooler`):

   ```
   postgresql://<usuario>:<password>@ep-xxxx.us-west-2.aws.neon.tech/energy?sslmode=require&channel_binding=require
   ```

   - Se usa la conexión directa porque `golang-migrate` toma un advisory lock de sesión, que no funciona detrás de PgBouncer en modo transacción.
   - `channel_binding=require` puede quedarse: el backend lo elimina antes de conectar, porque pgx 5.7 no lo soporta. `sslmode=require` se conserva y garantiza TLS.

No hay que crear tablas: el backend aplica las migraciones y carga los CSV al arrancar.

Alternativa por CLI:

```powershell
npx neonctl auth
npx neonctl projects create --name energy --region-id aws-us-west-2
npx neonctl connection-string --database-name energy
```

## 2. Backend (Render)

1. Crear cuenta en <https://dashboard.render.com> con GitHub y dar acceso al repositorio.
2. **New → Blueprint** → elegir el repositorio. Render lee `render.yaml` y crea el servicio `energy-api`.
   - La imagen se construye con `backend/Dockerfile.deploy`, con contexto en la raíz.
   - La imagen incluye solo `data/readings.csv` y `data/events.csv`.
3. Completar las variables que pide el Blueprint:

   | Variable | Valor |
   |---|---|
   | `DATABASE_URL` | Cadena directa de Neon del paso 1 |
   | `DEMO_USER_PASSWORD` | Contraseña nueva para la demo; no usar la de `docker-compose.yml`. Se comparte con el evaluador por privado. |
   | `DEEPSEEK_API_KEY` | Opcional. Sin ella, las explicaciones salen de plantillas. |
   | `CORS_ALLOWED_ORIGINS` | Por ahora `http://localhost:5173`; se corrige en el paso 3.4 |

   - `JWT_SECRET` lo genera Render solo.
   - `DEMO_USER_EMAIL` y `DEEPSEEK_MODEL` ya vienen fijados en el Blueprint.
4. Esperar el primer deploy y verificar:

   ```powershell
   Invoke-RestMethod https://energy-api-xxxx.onrender.com/health
   ```

   La respuesta esperada es `status = ok`, `db = ok` e `ingest.status = COMPLETED` con 4032 lecturas.

## 3. Frontend (Vercel)

1. Crear cuenta en <https://vercel.com> con GitHub.
2. **Add New → Project** → importar el repositorio → **Root Directory: `frontend`**. `frontend/vercel.json` ya define el build de Vite, la salida `dist/`, el fallback SPA a `index.html` y la caché de `/assets`.
3. En **Environment Variables**, agregar `VITE_API_URL` con la URL de Render del paso 2, sin barra final. Luego pulsar **Deploy**.
4. Copiar la URL final de Vercel, por ejemplo `https://bia-energy-demo.vercel.app`. En Render, cambiar `CORS_ALLOWED_ORIGINS` a esa URL. Render redespliega solo.

Alternativa por CLI, desde `frontend/`:

```powershell
npx vercel link
npx vercel env add VITE_API_URL production
npx vercel --prod
```

## Verificación final

1. Abrir la URL de Vercel. Si el backend estaba dormido, el primer login tarda ~1 min.
2. Recorrer el guion de la demo: Login → Dashboard → M-109 → Run AI Analysis → Anomalía → Explicación → Acción.
3. El orden de prioridad esperado es M-109, M-112, M-104 y M-106.

Antes de presentar, abrir `/health` para despertar el backend. Esto evita el arranque en frío de Render durante la demo.

## Problemas conocidos

| Síntoma | Causa | Solución |
|---|---|---|
| "No se pudo conectar con el servidor" con el backend arriba | `CORS_ALLOWED_ORIGINS` no coincide con la URL de Vercel | Poner la URL exacta, con `https://` y sin ruta |
| El login pasa a `localhost:8080` | Faltaba `VITE_API_URL` en el build | Agregarla en Vercel y redesplegar |
| `CONFIG_INVALID` en los logs de Render | Variable secreta vacía | Completarla en **Environment** |
| Render marca el deploy como fallido tras varios minutos | `DATABASE_URL` apunta al host `-pooler` o es incorrecta | Usar la cadena directa de Neon |
