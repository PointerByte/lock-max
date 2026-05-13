-- Copyright 2026 PointerByte Contributors
-- SPDX-License-Identifier: Apache-2.0

DO $$ BEGIN
  CREATE TYPE dragon_cmk.event_type AS ENUM ('create_key', 'rotate_key');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE dragon_cmk.key_purpose AS ENUM ('sign', 'encrypt', 'wrap');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE dragon_cmk.key_type AS ENUM ('SYMMETRIC_DEFAULT', 'RSA_OAEP', 'RSA_PKCS1v15_SHA256', 'ECDH', 'EdDSA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE dragon_cmk.key_status AS ENUM ('enabled', 'disabled', 'pendingDeletion', 'pendingImport', 'Unavailable');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE dragon_cmk.key_version_status AS ENUM ('enabled', 'disabled', 'pendingDeletion', 'retired', 'Unavailable');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE dragon_cmk.queue_status AS ENUM ('pending', 'processing', 'processed', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

commit;

-- =========================================================
-- TABLES
-- =========================================================

-- dragon_cmk.cmk_key
CREATE TABLE IF NOT EXISTS dragon_cmk.cmk_key (
  id_cmk_key           UUID NOT NULL,                     -- ES: Identificador unico de la llave logica. EN: Unique identifier of the logical key.
  algorithm            dragon_cmk.key_type NOT NULL,      -- ES: Algoritmo asignado a la llave. EN: Algorithm assigned to the key.
  purpose              dragon_cmk.key_purpose NOT NULL,  -- ES: Proposito principal de uso. EN: Main usage purpose.
  status               dragon_cmk.key_status NOT NULL DEFAULT 'enabled', -- ES: Estado general de la llave. EN: Overall key status.
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(), -- ES: Fecha de creacion de la llave. EN: Key creation timestamp.
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()  -- ES: Fecha de ultima actualizacion. EN: Last update timestamp.
)
TABLESPACE ts_dragon_cmk;

-- dragon_cmk.cmk_wrapping_key_ref
CREATE TABLE IF NOT EXISTS dragon_cmk.cmk_wrapping_key_ref (
  id_cmk_wrapping_key_ref UUID NOT NULL,                  -- ES: Identificador unico de la referencia de wrapping. EN: Unique identifier of the wrapping reference.
  provider                TEXT NOT NULL,                  -- ES: Proveedor o sistema externo. EN: External provider or system.
  key_ref                 TEXT NOT NULL,                  -- ES: Referencia externa de la wrapping key. EN: External wrapping key reference.
  version                 TEXT NOT NULL,                  -- ES: Version de la wrapping key. EN: Wrapping key version.
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now() -- ES: Fecha de registro. EN: Registration timestamp.
)
TABLESPACE ts_dragon_cmk;

-- dragon_cmk.cmk_creation_key_queue
CREATE TABLE IF NOT EXISTS dragon_cmk.cmk_creation_key_queue (
  id_cmk_key_creation_queue UUID NOT NULL,                     -- ES: Identificador unico del evento en cola. EN: Unique identifier of the queued event.
  id_cmk_key                UUID NOT NULL,                     -- ES: Llave creada asociada al evento. EN: Created key associated with the event.
  event_type                dragon_cmk.event_type NOT NULL,    -- ES: Tipo de evento registrado. EN: Registered event type.
  status                    dragon_cmk.queue_status NOT NULL DEFAULT 'pending', -- ES: Estado de procesamiento del evento. EN: Event processing status.
  error_message             TEXT,                              -- ES: Error detectado al procesar. EN: Processing error message.
  queued_at                 TIMESTAMPTZ NOT NULL DEFAULT now(), -- ES: Fecha en que se encolo el evento. EN: Event queue timestamp.
  processed_at              TIMESTAMPTZ                        -- ES: Fecha de procesamiento final. EN: Final processing timestamp.
)
TABLESPACE ts_dragon_cmk;

-- dragon_cmk.cmk_key_version
CREATE TABLE IF NOT EXISTS dragon_cmk.cmk_key_version (
  id_cmk_key_version        UUID NOT NULL,                     -- ES: Identificador unico de la version. EN: Unique identifier of the key version.
  id_cmk_key                UUID NOT NULL,                     -- ES: Llave logica propietaria. EN: Owning logical key.
  version_number            INT NOT NULL,                      -- ES: Numero secuencial de version. EN: Sequential version number.
  size                      INT NOT NULL,                      -- ES: Tamano de la llave. EN: Key size.
  status                    dragon_cmk.key_version_status NOT NULL DEFAULT 'disabled', -- ES: Estado operativo de la version. EN: Operational version status.
  kid                       TEXT NOT NULL,                     -- ES: Identificador publico de la version. EN: Public identifier of the version.
  secret_wrapped            TEXT NOT NULL,                     -- ES: Secreto protegido con wrapping. EN: Secret protected with wrapping.
  wrap_alg                  TEXT NOT NULL,                     -- ES: Algoritmo usado para envolver el secreto. EN: Algorithm used to wrap the secret.
  id_cmk_wrapping_key_ref   UUID,                              -- ES: Referencia de wrapping utilizada. EN: Wrapping reference used.
  aditional                 TEXT,                              -- ES: Informacion adicional opcional. EN: Optional additional information.
  created_at                TIMESTAMPTZ NOT NULL DEFAULT now(), -- ES: Fecha de creacion de la version. EN: Version creation timestamp.
  activated_at              TIMESTAMPTZ,                       -- ES: Fecha de activacion. EN: Activation timestamp.
  deactivated_at            TIMESTAMPTZ,                       -- ES: Fecha de desactivacion. EN: Deactivation timestamp.
  retired_at                TIMESTAMPTZ,                       -- ES: Fecha de retiro. EN: Retirement timestamp.
  secret_checksum           TEXT                               -- ES: Checksum del secreto. EN: Secret checksum.
)
TABLESPACE ts_dragon_cmk;

-- dragon_cmk.cmk_key_current_version
CREATE TABLE IF NOT EXISTS dragon_cmk.cmk_key_current_version (
  id_cmk_key                UUID NOT NULL,                     -- ES: Llave logica. EN: Logical key.
  id_cmk_key_version        UUID NOT NULL,                     -- ES: Version actual de la llave. EN: Current key version.
  created_at                TIMESTAMPTZ NOT NULL DEFAULT now(), -- ES: Fecha de creacion del puntero. EN: Pointer creation timestamp.
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()  -- ES: Fecha de ultima actualizacion. EN: Last update timestamp.
)
TABLESPACE ts_dragon_cmk;

-- dragon_cmk.api_client
CREATE TABLE IF NOT EXISTS dragon_cmk.api_client (
  id_api_client      UUID NOT NULL,                     -- ES: Identificador unico del cliente API. EN: Unique API client identifier.
  client_id_hash     TEXT NOT NULL,                     -- ES: HMAC del client_id. EN: HMAC of the client_id.
  client_secret_hash TEXT NOT NULL,                     -- ES: HMAC del client_secret. EN: HMAC of the client_secret.
  description        TEXT NOT NULL DEFAULT '',          -- ES: Descripcion operativa del cliente. EN: Operational client description.
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(), -- ES: Fecha de creacion. EN: Creation timestamp.
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()  -- ES: Fecha de ultima actualizacion. EN: Last update timestamp.
)
TABLESPACE ts_dragon_cmk;

commit;

-- =========================================================
-- CONSTRAINTS / KEYS
-- =========================================================

-- Limpieza del puntero anterior en cmk_key para bases existentes.
ALTER TABLE dragon_cmk.cmk_key
  DROP CONSTRAINT IF EXISTS fk_cmk_key__active_version;

ALTER TABLE dragon_cmk.cmk_key
  DROP CONSTRAINT IF EXISTS uq_cmk_key__active_version_pointer;

ALTER TABLE dragon_cmk.cmk_key
  DROP COLUMN IF EXISTS id_cmk_key_version;

-- PKs
ALTER TABLE dragon_cmk.cmk_key
  ADD CONSTRAINT pk_cmk_key PRIMARY KEY (id_cmk_key);

ALTER TABLE dragon_cmk.cmk_wrapping_key_ref
  ADD CONSTRAINT pk_cmk_wrapping_key_ref PRIMARY KEY (id_cmk_wrapping_key_ref);

ALTER TABLE dragon_cmk.cmk_creation_key_queue
  ADD CONSTRAINT pk_cmk_key_creation_queue PRIMARY KEY (id_cmk_key_creation_queue);

ALTER TABLE dragon_cmk.cmk_key_version
  ADD CONSTRAINT pk_cmk_key_version PRIMARY KEY (id_cmk_key_version);

ALTER TABLE dragon_cmk.cmk_key_current_version
  ADD CONSTRAINT pk_cmk_key_current_version PRIMARY KEY (id_cmk_key);

DO $$ BEGIN
  ALTER TABLE dragon_cmk.api_client
    ADD CONSTRAINT pk_api_client PRIMARY KEY (id_api_client);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- FKs (nombres de columnas FK = nombre de PK origen)
ALTER TABLE dragon_cmk.cmk_key_version
  ADD CONSTRAINT fk_cmk_key_version__cmk_key
  FOREIGN KEY (id_cmk_key)
  REFERENCES dragon_cmk.cmk_key(id_cmk_key)
  ON DELETE CASCADE;

ALTER TABLE dragon_cmk.cmk_creation_key_queue
  ADD CONSTRAINT fk_cmk_key_creation_queue__cmk_key
  FOREIGN KEY (id_cmk_key)
  REFERENCES dragon_cmk.cmk_key(id_cmk_key)
  ON DELETE CASCADE;

ALTER TABLE dragon_cmk.cmk_key_version
  ADD CONSTRAINT fk_cmk_key_version__wrapping
  FOREIGN KEY (id_cmk_wrapping_key_ref)
  REFERENCES dragon_cmk.cmk_wrapping_key_ref(id_cmk_wrapping_key_ref);

commit;

-- =========================================================
-- UNIQUE
-- =========================================================

-- Uniques por key/version y key/kid
ALTER TABLE dragon_cmk.cmk_key_version
  ADD CONSTRAINT uq_cmk_key_version__key_version_number UNIQUE (id_cmk_key, version_number);

ALTER TABLE dragon_cmk.cmk_key_version
  ADD CONSTRAINT uq_cmk_key_version__key_kid UNIQUE (id_cmk_key, kid);

ALTER TABLE dragon_cmk.cmk_key_version
  ADD CONSTRAINT uq_cmk_key_version__key_version_id UNIQUE (id_cmk_key, id_cmk_key_version);

ALTER TABLE dragon_cmk.cmk_key_current_version
  ADD CONSTRAINT uq_cmk_key_current_version__version UNIQUE (id_cmk_key_version);

ALTER TABLE dragon_cmk.cmk_key_current_version
  ADD CONSTRAINT fk_cmk_key_current_version__cmk_key
  FOREIGN KEY (id_cmk_key)
  REFERENCES dragon_cmk.cmk_key(id_cmk_key)
  ON DELETE CASCADE;

ALTER TABLE dragon_cmk.cmk_key_current_version
  ADD CONSTRAINT fk_cmk_key_current_version__version
  FOREIGN KEY (id_cmk_key, id_cmk_key_version)
  REFERENCES dragon_cmk.cmk_key_version(id_cmk_key, id_cmk_key_version)
  ON DELETE CASCADE;

commit;

-- =========================================================
-- INDEXES
-- =========================================================

-- Unicidad por wrapping key externa.
DROP INDEX IF EXISTS dragon_cmk.ux_cmk_wrapping_key_ref_unique;

CREATE UNIQUE INDEX IF NOT EXISTS ux_cmk_wrapping_key_ref_unique
ON dragon_cmk.cmk_wrapping_key_ref (provider, key_ref, version)
TABLESPACE ts_dragon_cmk;

-- Solo una versión activa por key (índice parcial)
DROP INDEX IF EXISTS dragon_cmk.ux_one_active_version_per_key;

CREATE UNIQUE INDEX IF NOT EXISTS ux_one_enabled_version_per_key
ON dragon_cmk.cmk_key_version (id_cmk_key)
TABLESPACE ts_dragon_cmk
WHERE status = 'enabled';

-- Índices optimizados para las consultas usadas por CMK/entity.
-- Las PK/UNIQUE ya cubren:
-- - cmk_key(id_cmk_key)
-- - cmk_wrapping_key_ref(id_cmk_wrapping_key_ref)
-- - cmk_wrapping_key_ref(provider, key_ref, version)
-- - cmk_key_version(id_cmk_key_version)
-- - cmk_key_version(id_cmk_key, ...)
-- - cmk_key_current_version(id_cmk_key_version)
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_type;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_version_key;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_version_status;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_version_wrapping_ref;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_current_version_version;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_creation_queue_status;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_creation_queue_queued_at;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_creation_queue_key;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_creation_queue_status_queued_at;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_creation_queue_key_queued_at;
DROP INDEX IF EXISTS dragon_cmk.idx_cmk_key_creation_queue_queued_at_id;

CREATE UNIQUE INDEX IF NOT EXISTS ux_api_client_client_id_hash
ON dragon_cmk.api_client(client_id_hash)
TABLESPACE ts_dragon_cmk;

-- Rotación de WrapKey:
-- QueryCmkKeyVersionView("WHERE id_cmk_wrapping_key_ref = $1", ...)
CREATE INDEX IF NOT EXISTS idx_cmk_key_version_wrapping_ref
ON dragon_cmk.cmk_key_version(id_cmk_wrapping_key_ref, id_cmk_key)
TABLESPACE ts_dragon_cmk
WHERE id_cmk_wrapping_key_ref IS NOT NULL;

-- getQueueData busca por id_cmk_key; el otro lado del OR usa la PK id_cmk_key_creation_queue.
-- El listado paginado puede filtrar por id_cmk_key y ordenar por fecha de cola.
CREATE INDEX IF NOT EXISTS idx_cmk_key_creation_queue_key_queued_at
ON dragon_cmk.cmk_creation_key_queue(id_cmk_key, queued_at DESC, id_cmk_key_creation_queue DESC)
TABLESPACE ts_dragon_cmk;

-- Listado general paginado: ORDER BY queued_at DESC, id_cmk_key_creation_queue DESC.
CREATE INDEX IF NOT EXISTS idx_cmk_key_creation_queue_queued_at_id
ON dragon_cmk.cmk_creation_key_queue(queued_at DESC, id_cmk_key_creation_queue DESC)
TABLESPACE ts_dragon_cmk;

-- Listado paginado por status.
CREATE INDEX IF NOT EXISTS idx_cmk_key_creation_queue_status_queued_at
ON dragon_cmk.cmk_creation_key_queue(status, queued_at DESC, id_cmk_key_creation_queue DESC)
TABLESPACE ts_dragon_cmk;

commit;
