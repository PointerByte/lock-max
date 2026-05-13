-- Copyright 2026 PointerByte Contributors
-- SPDX-License-Identifier: Apache-2.0

-- 1) Crear rol (usuario)
SELECT format('CREATE ROLE dragon_cmk_user LOGIN PASSWORD %L', :'dragon_cmk_user_password')
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_roles
  WHERE rolname = 'dragon_cmk_user'
)\gexec

ALTER ROLE dragon_cmk_user PASSWORD :'dragon_cmk_user_password';

-- 2) Crear schema con ese usuario como owner
CREATE SCHEMA IF NOT EXISTS dragon_cmk AUTHORIZATION dragon_cmk_user;

-- 3) Evitar que cualquiera use / cree en public (recomendado)
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE USAGE ON SCHEMA public FROM PUBLIC;

-- 4) Asegurar permisos del usuario en su schema
GRANT USAGE, CREATE ON SCHEMA dragon_cmk TO dragon_cmk_user;

-- 5) Hacer que ese usuario use ese schema por defecto
ALTER ROLE dragon_cmk_user SET search_path = dragon_cmk;

-- 6) Si ya existieran objetos en el schema (por si acaso)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA dragon_cmk TO dragon_cmk_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA dragon_cmk TO dragon_cmk_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA dragon_cmk TO dragon_cmk_user;

-- 7) Para que TODO lo nuevo quede con permisos correctos (por si otro rol crea algo)
ALTER DEFAULT PRIVILEGES IN SCHEMA dragon_cmk
GRANT ALL ON TABLES TO dragon_cmk_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA dragon_cmk
GRANT ALL ON SEQUENCES TO dragon_cmk_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA dragon_cmk
GRANT ALL ON FUNCTIONS TO dragon_cmk_user;

-- 8) Crear tablespace 
SELECT format('CREATE TABLESPACE %I LOCATION %L', 'ts_dragon_cmk', '/tablespaces/dragon_cmk')
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_tablespace
  WHERE spcname = 'ts_dragon_cmk'
)\gexec

commit;
