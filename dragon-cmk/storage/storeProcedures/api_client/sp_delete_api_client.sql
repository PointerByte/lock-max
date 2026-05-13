-- Copyright 2026 PointerByte Contributors
-- SPDX-License-Identifier: Apache-2.0

CREATE OR REPLACE PROCEDURE dragon_cmk.sp_delete_api_client(
  IN p_client_id_hash TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
  DELETE FROM dragon_cmk.api_client
  WHERE client_id_hash = p_client_id_hash;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'api client not found';
  END IF;
END;
$$;

COMMIT;
