CREATE OR REPLACE PROCEDURE dragon_cmk.sp_delete_creation_key_queue(
  p_id_cmk_key_creation_queue UUID
)
LANGUAGE plpgsql
AS $$
BEGIN
  DELETE FROM dragon_cmk.cmk_creation_key_queue
  WHERE id_cmk_key_creation_queue = p_id_cmk_key_creation_queue;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'cmk_creation_key_queue no existe: %', p_id_cmk_key_creation_queue;
  END IF;
END;
$$;

commit;
