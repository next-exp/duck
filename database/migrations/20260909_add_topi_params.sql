CREATE TABLE IF NOT EXISTS topiParams (
  id INT PRIMARY KEY NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  daemon_url VARCHAR(512) NOT NULL DEFAULT '',
  api_token TEXT NOT NULL,
  rabbitmq_address VARCHAR(255) NOT NULL DEFAULT '',
  rabbitmq_port INT NOT NULL DEFAULT 5672,
  rabbitmq_user VARCHAR(255) NOT NULL DEFAULT '',
  rabbitmq_password TEXT NOT NULL,
  rabbitmq_vhost VARCHAR(255) NOT NULL DEFAULT '/',
  exchange_name VARCHAR(255) NOT NULL DEFAULT 'production',
  control_queue VARCHAR(255) NOT NULL DEFAULT 'topi_daemon_control',
  selected_configuration VARCHAR(255) NOT NULL DEFAULT ''
);
