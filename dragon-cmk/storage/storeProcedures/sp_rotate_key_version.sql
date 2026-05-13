CREATE OR REPLACE PROCEDURE dragon_cmk.sp_rotate_key_version (
  p_id_cmk_key UUID,
  p_id_cmk_key_version UUID
)
LANGUAGE plpgsql
AS $$
BEGIN
  -- 1) Verificar que la versión pertenece al key
  IF NOT EXISTS (
    SELECT 1
    FROM dragon_cmk.cmk_key_version
    WHERE id_cmk_key_version = p_id_cmk_key_version
      AND id_cmk_key = p_id_cmk_key
  ) THEN
    RAISE EXCEPTION
      'La version % no pertenece al key %',
      p_id_cmk_key_version, p_id_cmk_key;
  END IF;

  -- 2) Desactivar versión activa actual (si existe)
  UPDATE dragon_cmk.cmk_key_version
  SET status = 'disabled',
      deactivated_at = now()
  WHERE id_cmk_key = p_id_cmk_key
    AND id_cmk_key_version <> p_id_cmk_key_version
    AND status = 'enabled';

  -- 3) Activar la nueva versión
  UPDATE dragon_cmk.cmk_key_version
  SET status = 'enabled',
      activated_at = COALESCE(activated_at, now()),
      deactivated_at = NULL
  WHERE id_cmk_key_version = p_id_cmk_key_version;

  -- 4) Actualizar puntero actual en tabla intermedia
  INSERT INTO dragon_cmk.cmk_key_current_version (
    id_cmk_key,
    id_cmk_key_version,
    created_at,
    updated_at
  )
  VALUES (
    p_id_cmk_key,
    p_id_cmk_key_version,
    now(),
    now()
  )
  ON CONFLICT (id_cmk_key) DO UPDATE
  SET id_cmk_key_version = EXCLUDED.id_cmk_key_version,
      updated_at = now();

  -- 5) Actualizar updated_at del key
  UPDATE dragon_cmk.cmk_key
  SET updated_at = now()
  WHERE id_cmk_key = p_id_cmk_key;
END;
$$;

commit;
