CREATE OR REPLACE PROCEDURE dragon_cmk.sp_delete_cmk_key(
  p_id_cmk_key UUID
)
LANGUAGE plpgsql
AS $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM dragon_cmk.cmk_key
    WHERE id_cmk_key = p_id_cmk_key
  ) THEN
    RAISE EXCEPTION 'cmk_key no existe: %', p_id_cmk_key;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM dragon_cmk.cmk_key_version
    WHERE id_cmk_key = p_id_cmk_key
  ) THEN
    RAISE EXCEPTION 'cmk_key % tiene versiones asociadas y no puede eliminarse', p_id_cmk_key;
  END IF;

  DELETE FROM dragon_cmk.cmk_key
  WHERE id_cmk_key = p_id_cmk_key;
END;
$$;

commit;
