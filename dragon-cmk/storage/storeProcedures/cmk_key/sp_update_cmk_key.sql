CREATE OR REPLACE PROCEDURE dragon_cmk.sp_update_cmk_key(
  p_id_cmk_key UUID,
  p_status dragon_cmk.key_status DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE dragon_cmk.cmk_key
  SET
    status = COALESCE(p_status, status),
    updated_at = now()
  WHERE id_cmk_key = p_id_cmk_key;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'cmk_key no existe: %', p_id_cmk_key;
  END IF;

  IF p_status IS NULL OR p_status = 'pendingImport' THEN
    RETURN;
  END IF;

  IF p_status = 'enabled' THEN
    UPDATE dragon_cmk.cmk_key_version
    SET status = 'disabled',
        deactivated_at = COALESCE(deactivated_at, now())
    WHERE id_cmk_key = p_id_cmk_key
      AND status <> 'retired';

    UPDATE dragon_cmk.cmk_key_version v
    SET status = 'enabled',
        activated_at = COALESCE(v.activated_at, now()),
        deactivated_at = NULL,
        retired_at = NULL
    FROM dragon_cmk.cmk_key_current_version cv
    WHERE cv.id_cmk_key = p_id_cmk_key
      AND cv.id_cmk_key_version = v.id_cmk_key_version;

    IF NOT FOUND THEN
      RAISE EXCEPTION 'cmk_key % no tiene version actual para habilitar', p_id_cmk_key;
    END IF;

    RETURN;
  END IF;

  UPDATE dragon_cmk.cmk_key_version
  SET status = p_status::TEXT::dragon_cmk.key_version_status,
      deactivated_at = CASE
        WHEN p_status = 'disabled' AND status = 'enabled' THEN now()
        ELSE deactivated_at
      END,
      retired_at = CASE
        WHEN p_status = 'disabled' THEN NULL
        ELSE retired_at
      END
  WHERE id_cmk_key = p_id_cmk_key;
END;
$$;

commit;
