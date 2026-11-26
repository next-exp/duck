-- name: InsertRateSample :exec
INSERT INTO rateSamples (
  run, gdc, sample_time, events, bytes, current_trigger_rate,
  current_byte_rate, avg_trigger_rate, avg_byte_rate, counter_reset
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE sample_time = rateSamples.sample_time;
