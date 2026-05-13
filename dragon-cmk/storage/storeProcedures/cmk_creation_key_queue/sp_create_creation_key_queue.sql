CREATE OR REPLACE PROCEDURE dragon_cmk.sp_create_creation_key_queue(
  p_id_cmk_key_creation_queue UUID,
  p_id_cmk_key UUID,
  p_event_type dragon_cmk.event_type,
  p_status dragon_cmk.queue_status DEFAULT 'pending',
  p_processed_at TIMESTAMPTZ DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM dragon_cmk.cmk_key
    WHERE id_cmk_key = p_id_cmk_key
  ) THEN
    RAISE EXCEPTION 'cmk_key no existe: %', p_id_cmk_key;
  END IF;

  INSERT INTO dragon_cmk.cmk_creation_key_queue (
    id_cmk_key_creation_queue,
    id_cmk_key,
    event_type,
    status,
    error_message,
    queued_at,
    processed_at
  )
  VALUES (
    p_id_cmk_key_creation_queue,
    p_id_cmk_key,
    p_event_type,
    p_status,
    NULL,
    now(),
    p_processed_at
  );

EXCEPTION
  WHEN unique_violation THEN
    RAISE EXCEPTION
      'cmk_creation_key_queue duplicado. id_queue=%, id_key=%, event_type=%',
      p_id_cmk_key_creation_queue, p_id_cmk_key, p_event_type;
END;
$$;

commit;
