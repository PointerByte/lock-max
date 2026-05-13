CREATE OR REPLACE VIEW dragon_cmk.vw_cmk_creation_key_queue AS
SELECT
  q.id_cmk_key_creation_queue,
  q.id_cmk_key,
  q.event_type,
  q.status,
  q.error_message,
  q.queued_at,
  q.processed_at
FROM dragon_cmk.cmk_creation_key_queue q;

commit;
