CREATE OR REPLACE PROCEDURE dragon_cmk.sp_update_creation_key_queue(
  p_id_cmk_key_creation_queue UUID,
  p_status dragon_cmk.queue_status,
  p_error_message TEXT DEFAULT NULL,
  p_processed_at TIMESTAMPTZ DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
BEGIN
  UPDATE dragon_cmk.cmk_creation_key_queue
  SET
    status = p_status,
    error_message = COALESCE(p_error_message, error_message),
    processed_at = COALESCE(
      p_processed_at,
      CASE
        WHEN p_status IN ('processed', 'failed') THEN now()
        ELSE processed_at
      END
    )
  WHERE id_cmk_key_creation_queue = p_id_cmk_key_creation_queue;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'cmk_creation_key_queue no existe: %', p_id_cmk_key_creation_queue;
  END IF;
END;
$$;

commit;
