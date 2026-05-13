CREATE OR REPLACE PROCEDURE dragon_cmk.sp_create_key_version(
  p_id_cmk_key_version UUID,
  p_id_cmk_key UUID,
  p_version_number INT,
  p_size INT,
  p_status dragon_cmk.key_version_status,
  p_kid TEXT,
  p_secret_wrapped TEXT,
  p_wrap_alg TEXT,
  p_id_cmk_wrapping_key_ref UUID DEFAULT NULL,
  p_aditional TEXT DEFAULT NULL,
  p_secret_checksum TEXT DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
DECLARE
  v_status dragon_cmk.key_version_status := COALESCE(p_status, 'disabled');
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM dragon_cmk.cmk_key
    WHERE id_cmk_key = p_id_cmk_key
  ) THEN
    RAISE EXCEPTION 'cmk_key no existe: %', p_id_cmk_key;
  END IF;

  IF v_status = 'enabled' THEN
    UPDATE dragon_cmk.cmk_key_version
    SET status = 'disabled',
        deactivated_at = now()
    WHERE id_cmk_key = p_id_cmk_key
      AND status = 'enabled';
  END IF;

  INSERT INTO dragon_cmk.cmk_key_version (
    id_cmk_key_version,
    id_cmk_key,
    version_number,
    size,
    status,
    kid,
    secret_wrapped,
    wrap_alg,
    id_cmk_wrapping_key_ref,
    aditional,
    created_at,
    activated_at,
    deactivated_at,
    retired_at,
    secret_checksum
  )
  VALUES (
    p_id_cmk_key_version,
    p_id_cmk_key,
    p_version_number,
    p_size,
    v_status,
    p_kid,
    p_secret_wrapped,
    p_wrap_alg,
    p_id_cmk_wrapping_key_ref,
    p_aditional,
    now(),
    CASE WHEN v_status = 'enabled' THEN now() ELSE NULL END,
    NULL,
    NULL,
    p_secret_checksum
  );

  IF v_status = 'enabled' THEN
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

    UPDATE dragon_cmk.cmk_key
    SET updated_at = now()
    WHERE id_cmk_key = p_id_cmk_key;
  END IF;

EXCEPTION
  WHEN unique_violation THEN
    RAISE EXCEPTION
      'key_version duplicada (id o (key,version_number) o (key,kid)). id=%, key=%, vnum=%, kid=%',
      p_id_cmk_key_version, p_id_cmk_key, p_version_number, p_kid;
END;
$$;

commit;
