CREATE OR REPLACE VIEW dragon_cmk.vw_cmk_key AS
SELECT
  k.id_cmk_key,
  k.algorithm,
  k.purpose,
  k.status,
  cv.id_cmk_key_version,
  k.created_at,
  k.updated_at
FROM dragon_cmk.cmk_key k
LEFT JOIN dragon_cmk.cmk_key_current_version cv
  ON cv.id_cmk_key = k.id_cmk_key;

commit;
