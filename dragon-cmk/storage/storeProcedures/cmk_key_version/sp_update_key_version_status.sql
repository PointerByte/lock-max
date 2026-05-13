CREATE OR REPLACE PROCEDURE dragon_cmk.sp_update_key_version_status(
  p_id_cmk_key_version UUID,
  p_status dragon_cmk.key_version_status
)
LANGUAGE plpgsql
AS $$
DECLARE
  v_id_cmk_key UUID;
BEGIN
  IF p_status IS NULL THEN
    RAISE EXCEPTION 'status es requerido';
  END IF;

  IF p_status = 'pendingDeletion' THEN
    RAISE EXCEPTION 'pendingDeletion no puede usarse para actualizar una version especifica';
  END IF;

  SELECT id_cmk_key
  INTO v_id_cmk_key
  FROM dragon_cmk.cmk_key_version
  WHERE id_cmk_key_version = p_id_cmk_key_version;

  IF v_id_cmk_key IS NULL THEN
    RAISE EXCEPTION 'cmk_key_version no existe: %', p_id_cmk_key_version;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM dragon_cmk.cmk_key_current_version
    WHERE id_cmk_key = v_id_cmk_key
      AND id_cmk_key_version = p_id_cmk_key_version
  ) THEN
    RAISE EXCEPTION 'la version principal no puede ser actualizada: %', p_id_cmk_key_version;
  END IF;

  UPDATE dragon_cmk.cmk_key_version
  SET status = p_status,
      activated_at = CASE
        WHEN p_status = 'enabled' THEN COALESCE(activated_at, now())
        ELSE activated_at
      END,
      deactivated_at = CASE
        WHEN p_status <> 'enabled' AND status = 'enabled' THEN now()
        WHEN p_status = 'enabled' THEN NULL
        ELSE deactivated_at
      END,
      retired_at = CASE
        WHEN p_status = 'retired' THEN COALESCE(retired_at, now())
        WHEN p_status IN ('enabled', 'disabled') THEN NULL
        ELSE retired_at
      END
  WHERE id_cmk_key_version = p_id_cmk_key_version;
END;
$$;

commit;
