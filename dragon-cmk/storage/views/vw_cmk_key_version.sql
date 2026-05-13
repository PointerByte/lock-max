CREATE OR REPLACE VIEW dragon_cmk.vw_cmk_key_version AS
SELECT
  v.id_cmk_key_version,
  v.id_cmk_key,
  v.version_number,
  v.size,
  v.status,
  v.kid,
  v.secret_wrapped,
  v.wrap_alg,
  v.id_cmk_wrapping_key_ref,
  v.aditional,
  v.created_at,
  v.activated_at,
  v.deactivated_at,
  v.retired_at,
  v.secret_checksum
FROM dragon_cmk.cmk_key_version v;

commit;
