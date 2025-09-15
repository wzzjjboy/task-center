CREATE TABLE task_locks (
  id bigint(20) NOT NULL AUTO_INCREMENT,
  task_id bigint(20) NOT NULL,
  lock_key varchar(128) NOT NULL,
  node_id varchar(64) NOT NULL,
  locked_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at timestamp NOT NULL,
  version int(11) NOT NULL DEFAULT 1,
  PRIMARY KEY (id),
  UNIQUE KEY uk_lock_key (lock_key),
  KEY idx_task_id (task_id),
  KEY idx_expires_at (expires_at),
  CONSTRAINT fk_locks_task_id FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;