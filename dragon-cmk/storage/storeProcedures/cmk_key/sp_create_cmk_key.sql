CREATE OR REPLACE PROCEDURE dragon_cmk.sp_create_cmk_key(
  p_id_cmk_key UUID,
  p_algorithm TEXT,
  p_purpose dragon_cmk.key_purpose,
  p_status dragon_cmk.key_status DEFAULT 'enabled'
)
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO dragon_cmk.cmk_key (
    id_cmk_key, algorithm, purpose, status,
    created_at, updated_at
  )
  VALUES (
    p_id_cmk_key, p_algorithm::dragon_cmk.key_type, p_purpose, COALESCE(p_status, 'enabled'),
    now(), now()
  );
  
EXCEPTION
  WHEN invalid_text_representation THEN
    RAISE EXCEPTION 'Algoritmo no permitido: %', p_algorithm;
  WHEN unique_violation THEN
    RAISE EXCEPTION 'cmk_key ya existe. id=%', p_id_cmk_key;
END;
$$;

commit;
