CREATE OR REPLACE PROCEDURE dragon_cmk.sp_update_key_version_metadata(
  p_id_cmk_key_version UUID,
  p_id_cmk_wrapping_key_ref UUID DEFAULT NULL,
  p_secret_wrapped TEXT DEFAULT NULL,
  p_secret_checksum TEXT DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE dragon_cmk.cmk_key_version
  SET
    id_cmk_wrapping_key_ref = COALESCE(p_id_cmk_wrapping_key_ref, id_cmk_wrapping_key_ref),
    secret_wrapped = COALESCE(p_secret_wrapped, secret_wrapped),
    secret_checksum = COALESCE(p_secret_checksum, secret_checksum)
  WHERE id_cmk_key_version = p_id_cmk_key_version;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'cmk_key_version no existe: %', p_id_cmk_key_version;
  END IF;
END;
$$;

commit;
