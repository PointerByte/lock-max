CREATE OR REPLACE PROCEDURE dragon_cmk.sp_delete_wrapping_key_ref(
  p_id_cmk_wrapping_key_ref UUID
)
LANGUAGE plpgsql
AS $$
BEGIN
  DELETE FROM dragon_cmk.cmk_wrapping_key_ref
  WHERE id_cmk_wrapping_key_ref = p_id_cmk_wrapping_key_ref;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'wrapping_key_ref no existe: %', p_id_cmk_wrapping_key_ref;
  END IF;
END;
$$;

commit;
