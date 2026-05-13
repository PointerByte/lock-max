CREATE OR REPLACE VIEW dragon_cmk.vw_cmk_wrapping_key_ref AS
SELECT
  w.id_cmk_wrapping_key_ref,
  w.provider,
  w.key_ref,
  w.version,
  w.created_at
FROM dragon_cmk.cmk_wrapping_key_ref w;

commit;
