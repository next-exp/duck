CREATE TABLE IF NOT EXISTS rateSamples (
  run INT NOT NULL,
  gdc VARCHAR(100) NOT NULL,
  sample_time DATETIME(6) NOT NULL COMMENT 'UTC source message timestamp',
  events BIGINT UNSIGNED NOT NULL,
  bytes BIGINT UNSIGNED NOT NULL,
  current_trigger_rate DOUBLE NOT NULL,
  current_byte_rate DOUBLE NOT NULL,
  avg_trigger_rate DOUBLE NOT NULL,
  avg_byte_rate DOUBLE NOT NULL,
  counter_reset BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (run, gdc, sample_time),
  INDEX rateSamples_run_time (run, sample_time)
) ENGINE=InnoDB;
