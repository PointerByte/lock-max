-- Copyright 2026 PointerByte Contributors
-- SPDX-License-Identifier: Apache-2.0

CREATE OR REPLACE VIEW dragon_cmk.vw_api_client AS
SELECT
  id_api_client,
  client_id_hash,
  description,
  created_at,
  updated_at
FROM dragon_cmk.api_client;

COMMIT;
