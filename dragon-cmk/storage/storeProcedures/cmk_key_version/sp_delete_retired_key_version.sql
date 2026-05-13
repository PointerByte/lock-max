CREATE OR REPLACE PROCEDURE dragon_cmk.sp_delete_retired_key_version(
  p_id_cmk_key_version UUID
)
LANGUAGE plpgsql
AS $$
DECLARE
  v_id_cmk_key UUID;
BEGIN
  SELECT id_cmk_key
  INTO v_id_cmk_key
  FROM dragon_cmk.cmk_key_version
  WHERE id_cmk_key_version = p_id_cmk_key_version
    AND status IN ('retired', 'pendingDeletion');

  IF v_id_cmk_key IS NULL THEN
    IF EXISTS (
      SELECT 1
      FROM dragon_cmk.cmk_key_version
      WHERE id_cmk_key_version = p_id_cmk_key_version
    ) THEN
      RAISE EXCEPTION 'cmk_key_version % no esta en status retired o pendingDeletion', p_id_cmk_key_version;
    END IF;

    RAISE EXCEPTION 'cmk_key_version no existe: %', p_id_cmk_key_version;
  END IF;

  DELETE FROM dragon_cmk.cmk_key_version
  WHERE id_cmk_key_version = p_id_cmk_key_version
    AND status IN ('retired', 'pendingDeletion');

  IF NOT FOUND THEN
    RAISE EXCEPTION 'cmk_key no existe para cmk_key_version: %', p_id_cmk_key_version;
  END IF;
END;
$$;

commit;
