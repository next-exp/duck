CREATE USER IF NOT EXISTS 'duck'@'%' IDENTIFIED BY 'dummy_password';
GRANT ALL ON duck.* TO 'duck'@'%';
GRANT ALL ON NEXT100DB.* TO 'duck'@'%';
FLUSH PRIVILEGES;
