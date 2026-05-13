-- Copyright 2026 PointerByte Contributors
-- SPDX-License-Identifier: Apache-2.0

CREATE OR REPLACE PROCEDURE dragon_cmk.sp_create_api_client(
  IN p_id_api_client UUID,
  IN p_client_id_hash TEXT,
  IN p_client_secret_hash TEXT,
  IN p_description TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO dragon_cmk.api_client (
    id_api_client,
    client_id_hash,
    client_secret_hash,
    description
  )
  VALUES (
    p_id_api_client,
    p_client_id_hash,
    p_client_secret_hash,
    COALESCE(p_description, '')
  );
END;
$$;

COMMIT;
