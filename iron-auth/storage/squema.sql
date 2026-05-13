-- Copyright 2026 PointerByte Contributors
-- SPDX-License-Identifier: Apache-2.0

-- 1) Crear rol (usuario)
SELECT format('CREATE ROLE iron_auth_user LOGIN PASSWORD %L', :'iron_auth_user_password')
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_roles
  WHERE rolname = 'iron_auth_user'
)\gexec

ALTER ROLE iron_auth_user PASSWORD :'iron_auth_user_password';

-- 2) Crear schema con ese usuario como owner
CREATE SCHEMA IF NOT EXISTS iron_auth AUTHORIZATION iron_auth_user;

-- 3) Evitar que cualquiera use / cree en public (recomendado)
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE USAGE ON SCHEMA public FROM PUBLIC;

-- 4) Asegurar permisos del usuario en su schema
GRANT USAGE, CREATE ON SCHEMA iron_auth TO iron_auth_user;

-- 5) Hacer que ese usuario use ese schema por defecto
ALTER ROLE iron_auth_user SET search_path = iron_auth;

-- 6) Si ya existieran objetos en el schema (por si acaso)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA iron_auth TO iron_auth_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA iron_auth TO iron_auth_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA iron_auth TO iron_auth_user;

-- 7) Para que TODO lo nuevo quede con permisos correctos (por si otro rol crea algo)
ALTER DEFAULT PRIVILEGES IN SCHEMA iron_auth
GRANT ALL ON TABLES TO iron_auth_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA iron_auth
GRANT ALL ON SEQUENCES TO iron_auth_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA iron_auth
GRANT ALL ON FUNCTIONS TO iron_auth_user;

-- 8) Crear tablespace 
SELECT format('CREATE TABLESPACE %I LOCATION %L', 'ts_iron_auth', '/tablespaces/iron_auth')
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_tablespace
  WHERE spcname = 'ts_iron_auth'
)\gexec

commit;
