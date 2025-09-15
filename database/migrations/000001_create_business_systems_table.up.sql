CREATE TABLE business_systems (
  id bigint(20) NOT NULL AUTO_INCREMENT,
  business_code varchar(64) NOT NULL,
  business_name varchar(128) NOT NULL,
  api_key varchar(128) NOT NULL,
  api_secret varchar(256) NOT NULL,
  rate_limit int(11) NOT NULL DEFAULT 1000,
  status tinyint(4) NOT NULL DEFAULT 1,
  description text,
  contact_info varchar(256) DEFAULT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_business_code (business_code),
  UNIQUE KEY uk_api_key (api_key),
  KEY idx_status (status),
  KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;