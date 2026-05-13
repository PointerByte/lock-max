CREATE OR REPLACE PROCEDURE dragon_cmk.sp_update_wrapping_key_ref(
  p_id_cmk_wrapping_key_ref UUID,
  p_provider TEXT DEFAULT NULL,
  p_key_ref TEXT DEFAULT NULL,
  p_version TEXT DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE dragon_cmk.cmk_wrapping_key_ref
  SET
    provider = COALESCE(p_provider, provider),
    key_ref = COALESCE(p_key_ref, key_ref),
    version = COALESCE(p_version, version)
  WHERE id_cmk_wrapping_key_ref = p_id_cmk_wrapping_key_ref;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'wrapping_key_ref no existe: %', p_id_cmk_wrapping_key_ref;
  END IF;
END;
$$;

commit;
