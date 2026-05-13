CREATE OR REPLACE PROCEDURE dragon_cmk.sp_create_wrapping_key_ref(
  p_id_cmk_wrapping_key_ref UUID,
  p_provider TEXT,
  p_key_ref TEXT,
  p_version TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO dragon_cmk.cmk_wrapping_key_ref (
    id_cmk_wrapping_key_ref, provider, key_ref, version, created_at
  )
  VALUES (
    p_id_cmk_wrapping_key_ref, p_provider, p_key_ref, p_version, now()
  );
EXCEPTION
  WHEN unique_violation THEN
    RAISE EXCEPTION 'wrapping_key_ref ya existe (unique index provider+key_ref). provider=%, key_ref=%',
      p_provider, p_key_ref;
END;
$$;

commit;
