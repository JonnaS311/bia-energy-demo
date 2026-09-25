-- DDL del maestro §8.1 con dos ajustes necesarios para PostgreSQL y la ingesta:
--  1. "window" va entre comillas: WINDOW es palabra reservada en PostgreSQL.
--  2. events tiene UNIQUE (meter_id, timestamp, type): lo exige el UPSERT de la ingesta (spec de data RF-D-05).

CREATE TABLE meters (
  id            bigserial PRIMARY KEY,
  meter_id      text NOT NULL UNIQUE,
  name          text NOT NULL,
  location      text NOT NULL,
  status        text NOT NULL DEFAULT 'OK' CHECK (status IN ('OK','ALERT','CRITICAL')),
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE readings (
  id              bigserial PRIMARY KEY,
  meter_id        text NOT NULL REFERENCES meters(meter_id),
  timestamp       timestamptz NOT NULL,
  consumption_kwh numeric(10,3) NOT NULL,
  voltage_v       numeric(8,3)  NOT NULL,
  current_a       numeric(8,3)  NOT NULL,
  power_factor    numeric(5,3)  NOT NULL,
  status          text NOT NULL DEFAULT 'OK',
  UNIQUE (meter_id, timestamp)
);
CREATE INDEX readings_meter_ts_idx ON readings (meter_id, timestamp);

CREATE TABLE events (
  id          bigserial PRIMARY KEY,
  meter_id    text NOT NULL REFERENCES meters(meter_id),
  timestamp   timestamptz NOT NULL,
  type        text NOT NULL,
  description text NOT NULL DEFAULT '',
  UNIQUE (meter_id, timestamp, type)
);

CREATE TABLE analyses (
  id             uuid PRIMARY KEY,
  status         text NOT NULL CHECK (status IN ('QUEUED','RUNNING','COMPLETED','FAILED')),
  steps          jsonb NOT NULL,
  summary        jsonb,
  error_message  text,
  started_at     timestamptz,
  finished_at    timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE anomalies (
  id                 uuid PRIMARY KEY,
  analysis_id        uuid NOT NULL REFERENCES analyses(id),
  meter_id           text NOT NULL REFERENCES meters(meter_id),
  detected_at        timestamptz NOT NULL,
  window_start       date NOT NULL,
  window_end         date NOT NULL,
  type               text NOT NULL CHECK (type IN ('REAL_ANOMALY','EXPLAINABLE_ANOMALY','FALSE_POSITIVE','DATA_QUALITY')),
  severity           text NOT NULL CHECK (severity IN ('HIGH','MEDIUM','LOW')),
  confidence         numeric(4,2) NOT NULL CHECK (confidence BETWEEN 0.50 AND 0.98),
  priority_rank      integer NOT NULL,
  reason             text NOT NULL,
  recommended_action text NOT NULL,
  explanation_points jsonb NOT NULL,
  explanation_source text NOT NULL CHECK (explanation_source IN ('llm','template')),
  related_event_id   bigint REFERENCES events(id),
  status             text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','INVESTIGATING','RESOLVED','DISMISSED')),
  superseded         boolean NOT NULL DEFAULT false,
  UNIQUE (analysis_id, meter_id)
);

CREATE TABLE anomaly_evidence (
  id          bigserial PRIMARY KEY,
  anomaly_id  uuid NOT NULL REFERENCES anomalies(id) ON DELETE CASCADE,
  position    integer NOT NULL,
  metric      text NOT NULL,
  observed    numeric(12,3),
  baseline    numeric(12,3),
  delta_pct   numeric(8,1),
  unit        text NOT NULL,
  "window"    text NOT NULL,
  detail      text NOT NULL
);

CREATE TABLE users (
  id            bigserial PRIMARY KEY,
  email         text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  name          text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now()
);
