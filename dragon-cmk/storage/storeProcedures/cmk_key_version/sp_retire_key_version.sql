CREATE OR REPLACE PROCEDURE dragon_cmk.sp_retire_key_version(
  p_id_cmk_key_version UUID
)
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE dragon_cmk.cmk_key_version
  SET status = 'retired',
      retired_at = now()
  WHERE id_cmk_key_version = p_id_cmk_key_version;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'cmk_key_version no existe: %', p_id_cmk_key_version;
  END IF;
END;
$$;
commit;
