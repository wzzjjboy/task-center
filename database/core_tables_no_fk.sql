-- ========================================
-- Task Center 核心表结构 DDL (无外键版本)
-- 创建时间: 2025-09-15
-- 描述: 任务中心核心表结构，用于 goctl model 代码生成，移除外键约束
-- ========================================

-- 业务系统表
CREATE TABLE `business_systems` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID，自增',
  `business_code` varchar(64) NOT NULL COMMENT '业务系统唯一标识码，如：user-service、order-service',
  `business_name` varchar(128) NOT NULL COMMENT '业务系统名称，如：用户服务、订单服务',
  `api_key` varchar(128) NOT NULL COMMENT 'API访问密钥，用于系统认证',
  `api_secret` varchar(256) NOT NULL COMMENT 'API密钥对应的秘钥，加密存储',
  `rate_limit` int(11) NOT NULL DEFAULT '1000' COMMENT '速率限制，每分钟最大请求数',
  `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '系统状态：0-禁用，1-启用，2-维护中',
  `description` text COMMENT '业务系统描述信息',
  `contact_info` varchar(256) DEFAULT NULL COMMENT '联系人信息，JSON格式存储',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_business_code` (`business_code`),
  UNIQUE KEY `uk_api_key` (`api_key`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='业务系统表，管理接入任务中心的各个业务系统';

-- 任务表
CREATE TABLE `tasks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID，自增',
  `business_id` bigint(20) NOT NULL COMMENT '业务系统ID，关联 business_systems.id',
  `business_unique_id` varchar(128) NOT NULL COMMENT '业务系统内的唯一ID，由业务方定义',
  `callback_url` varchar(512) NOT NULL COMMENT '回调地址，任务执行时的HTTP回调URL',
  `callback_method` varchar(10) NOT NULL DEFAULT 'POST' COMMENT 'HTTP回调方法：GET、POST、PUT、DELETE等',
  `callback_headers` text COMMENT '回调请求头，JSON格式存储',
  `callback_body` text COMMENT '回调请求体，支持模板变量',
  `retry_intervals` varchar(256) NOT NULL DEFAULT '[60,300,900]' COMMENT '重试间隔配置，JSON数组，单位秒，如：[60,300,900]',
  `max_retries` int(11) NOT NULL DEFAULT '3' COMMENT '最大重试次数',
  `current_retry` int(11) NOT NULL DEFAULT '0' COMMENT '当前重试次数',
  `status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '任务状态：0-待执行，1-执行中，2-成功，3-失败，4-取消，5-过期',
  `priority` tinyint(4) NOT NULL DEFAULT '5' COMMENT '任务优先级，1-9，数字越小优先级越高',
  `tags` varchar(512) DEFAULT NULL COMMENT '任务标签，JSON数组格式，用于分类和查询',
  `timeout` int(11) NOT NULL DEFAULT '30' COMMENT '任务超时时间，单位秒',
  `scheduled_at` timestamp NOT NULL COMMENT '计划执行时间',
  `next_execute_at` timestamp NULL DEFAULT NULL COMMENT '下次执行时间，用于延时和重试',
  `executed_at` timestamp NULL DEFAULT NULL COMMENT '实际执行时间',
  `completed_at` timestamp NULL DEFAULT NULL COMMENT '完成时间',
  `error_message` text COMMENT '最新的错误信息',
  `metadata` text COMMENT '扩展元数据，JSON格式存储',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_business_task` (`business_id`, `business_unique_id`),
  KEY `idx_status` (`status`),
  KEY `idx_priority` (`priority`),
  KEY `idx_next_execute_at` (`next_execute_at`),
  KEY `idx_scheduled_at` (`scheduled_at`),
  KEY `idx_business_id_status` (`business_id`, `status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='任务表，存储所有待执行的任务信息';

-- 任务执行历史表
CREATE TABLE `task_executions` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID，自增',
  `task_id` bigint(20) NOT NULL COMMENT '任务ID，关联 tasks.id',
  `execution_sequence` int(11) NOT NULL COMMENT '执行序号，从1开始，表示第几次执行',
  `execution_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '执行开始时间',
  `duration` int(11) DEFAULT NULL COMMENT '执行耗时，单位毫秒',
  `http_status` int(11) DEFAULT NULL COMMENT 'HTTP响应状态码',
  `response_headers` text COMMENT 'HTTP响应头，JSON格式存储',
  `response_data` text COMMENT 'HTTP响应体数据',
  `error_message` text COMMENT '执行错误信息',
  `retry_after` timestamp NULL DEFAULT NULL COMMENT '下次重试时间',
  `execution_node` varchar(64) DEFAULT NULL COMMENT '执行节点标识，用于分布式环境',
  `trace_id` varchar(128) DEFAULT NULL COMMENT '链路追踪ID',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_execution_time` (`execution_time`),
  KEY `idx_task_sequence` (`task_id`, `execution_sequence`),
  KEY `idx_http_status` (`http_status`),
  KEY `idx_trace_id` (`trace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='任务执行历史表，记录每次任务执行的详细信息';

-- 任务锁表
CREATE TABLE `task_locks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID，自增',
  `task_id` bigint(20) NOT NULL COMMENT '任务ID，关联 tasks.id',
  `lock_key` varchar(128) NOT NULL COMMENT '锁的唯一标识',
  `node_id` varchar(64) NOT NULL COMMENT '持有锁的节点ID',
  `locked_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加锁时间',
  `expires_at` timestamp NOT NULL COMMENT '锁过期时间',
  `version` int(11) NOT NULL DEFAULT '1' COMMENT '版本号，用于乐观锁',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_lock_key` (`lock_key`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='任务锁表，用于分布式环境下的任务执行锁';

-- 迁移状态跟踪表
CREATE TABLE `migrations` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID，自增',
  `version` varchar(20) NOT NULL COMMENT '迁移版本号，格式：001、002、003',
  `name` varchar(255) NOT NULL COMMENT '迁移名称，描述本次迁移的内容',
  `filename` varchar(255) NOT NULL COMMENT '迁移文件名',
  `checksum` varchar(64) NOT NULL COMMENT '迁移文件内容的SHA256校验和',
  `applied_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '迁移执行时间',
  `execution_time` int(11) DEFAULT NULL COMMENT '迁移执行耗时，单位毫秒',
  `status` enum('SUCCESS','FAILED','RUNNING') NOT NULL DEFAULT 'SUCCESS' COMMENT '迁移状态',
  `error_message` text COMMENT '错误信息（如果迁移失败）',
  `rollback_available` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否支持回滚：1-是，0-否',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_version` (`version`),
  UNIQUE KEY `uk_filename` (`filename`),
  KEY `idx_applied_at` (`applied_at`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='数据库迁移版本跟踪表，记录每次迁移的执行状态和详情';

-- 迁移锁表
CREATE TABLE `migration_locks` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `lock_name` varchar(64) NOT NULL COMMENT '锁名称',
  `locked_by` varchar(128) NOT NULL COMMENT '持有锁的进程标识',
  `locked_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加锁时间',
  `expires_at` timestamp NOT NULL COMMENT '锁过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_lock_name` (`lock_name`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='迁移执行锁表，防止并发执行迁移导致数据不一致';