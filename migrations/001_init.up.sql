CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin', 'staff')),
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE counters (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE queues (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  queue_number TEXT NOT NULL,
  service_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'waiting'
    CHECK (status IN ('waiting','calling','serving','completed','skipped')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  called_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  counter_id INTEGER REFERENCES counters(id),
  user_id UUID REFERENCES users(id),
  rating INTEGER CHECK (rating BETWEEN 1 AND 5),
  feedback TEXT
);
CREATE INDEX idx_queues_status_service_created ON queues(status, service_type, created_at);
CREATE INDEX idx_queues_created_at ON queues(created_at);

CREATE TABLE videos (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  target_screen TEXT NOT NULL CHECK (target_screen IN ('kiosk','display','both')),
  duration_seconds INTEGER,
  display_order INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT true,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  uploaded_by UUID REFERENCES users(id)
);
