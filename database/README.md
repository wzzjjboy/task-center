# Task Center - 数据库迁移完整指南

## 📋 概述

本项目使用 [golang-migrate](https://github.com/golang-migrate/migrate) 作为数据库迁移工具，提供企业级的数据库版本管理。经过完整的验证和测试，现已替代自制迁移脚本，为项目提供更稳定和专业的迁移管理。

**当前状态**: ✅ 完全就绪，4个核心表已成功迁移
**数据库版本**: 4
**最后更新**: 2025-09-15

## 🚀 快速开始 (3分钟上手)

### 前提条件
- ✅ MySQL 5.7+ 数据库
- ✅ Go 1.21+ 环境
- ✅ Docker (如果使用容器数据库)

### 步骤1：安装 golang-migrate

```bash
# 安装带 MySQL 支持的 migrate 工具
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 验证安装
migrate -version
# 预期输出: dev 或版本号
```

### 步骤2：环境配置

```bash
# 方式A: 直连数据库
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root123
export DB_NAME=task_center

# 方式B: 使用 Docker 容器 (推荐)
export DOCKER_CONTAINER=jcsk-mysql  # 你的 MySQL 容器名
```

### 步骤3：执行迁移 (推荐使用脚本)

```bash
# 进入项目目录
cd /path/to/task-center

# 使用项目脚本执行迁移 (推荐)
export DOCKER_CONTAINER=jcsk-mysql
./database/migrate.sh up

# 查看迁移状态
./database/migrate.sh status
```

### 验证安装

执行成功后应该看到：
- ✅ 5个表：business_systems, tasks, task_executions, task_locks, schema_migrations
- ✅ 3个外键约束正常工作
- ✅ 当前版本为 4

## 📁 目录结构

```
database/
├── migrations/                    # golang-migrate 迁移文件
│   ├── 000001_create_business_systems_table.up.sql
│   ├── 000001_create_business_systems_table.down.sql
│   ├── 000002_create_tasks_table.up.sql
│   ├── 000002_create_tasks_table.down.sql
│   ├── 000003_create_task_executions_table.up.sql
│   ├── 000003_create_task_executions_table.down.sql
│   ├── 000004_create_task_locks_table.up.sql
│   └── 000004_create_task_locks_table.down.sql
├── migrate.sh                     # 🔧 主要迁移管理脚本
├── integration.go                 # Go 代码集成接口
├── core_tables_no_fk.sql         # goctl 模型生成专用
└── README_GOLANG_MIGRATE.md      # 📖 本文档
```

## 🛠️ migrate.sh 脚本详细使用指南

### 脚本功能概览

`migrate.sh` 是项目提供的统一迁移管理工具，封装了 golang-migrate 的常用操作，提供更友好的使用体验。

### 环境变量配置

```bash
# 必需配置（二选一）
export DOCKER_CONTAINER=jcsk-mysql        # Docker 容器名（推荐）
# 或者
export DB_HOST=localhost                   # 直连数据库
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root123
export DB_NAME=task_center
```

### 📋 所有可用命令

#### 1. 迁移执行命令

```bash
# 执行所有待执行迁移
./database/migrate.sh up

# 执行指定数量的迁移
./database/migrate.sh up 2

# 迁移到指定版本
./database/migrate.sh goto 3
```

#### 2. 回滚命令

```bash
# 回滚最新的1个迁移
./database/migrate.sh down 1

# 回滚最新的2个迁移
./database/migrate.sh down 2

# 迁移到指定版本（支持向前或向后）
./database/migrate.sh goto 2
```

#### 3. 状态查看命令

```bash
# 查看详细迁移状态
./database/migrate.sh status

# 查看当前版本号
./database/migrate.sh version
```

#### 4. 开发命令

```bash
# 创建新迁移文件
./database/migrate.sh create add_user_table

# 验证迁移文件完整性
./database/migrate.sh validate
```

#### 5. 紧急修复命令

```bash
# 强制设置版本（修复脏状态）
./database/migrate.sh force 3

# 删除所有数据和迁移历史（危险操作）
./database/migrate.sh drop
```

#### 6. 帮助命令

```bash
# 显示完整帮助信息
./database/migrate.sh help
```

### 🎯 典型使用场景

#### 场景1：首次部署
```bash
# 1. 配置环境
export DOCKER_CONTAINER=your-mysql-container

# 2. 执行所有迁移
./database/migrate.sh up

# 3. 验证结果
./database/migrate.sh status
```

#### 场景2：开发新功能
```bash
# 1. 创建新迁移
./database/migrate.sh create add_user_permissions

# 2. 编辑生成的 up/down 文件
# 编辑 migrations/000005_add_user_permissions.up.sql
# 编辑 migrations/000005_add_user_permissions.down.sql

# 3. 执行新迁移
./database/migrate.sh up

# 4. 测试回滚
./database/migrate.sh down 1
./database/migrate.sh up
```

#### 场景3：生产环境部署
```bash
# 1. 备份数据库
mysqldump -u root -p task_center > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. 在测试环境验证
./database/migrate.sh up

# 3. 查看将要执行的迁移
./database/migrate.sh status

# 4. 生产环境执行（维护窗口）
./database/migrate.sh up

# 5. 验证结果
./database/migrate.sh status
```

#### 场景4：故障恢复
```bash
# 查看状态
./database/migrate.sh status

# 如果发现脏状态，强制修复
./database/migrate.sh force 3

# 重新执行迁移
./database/migrate.sh up
```

## 🎯 最佳实践和注意事项

### 1. 迁移文件编写规范

#### ✅ 推荐实践
```sql
-- 使用 IF NOT EXISTS 确保幂等性
CREATE TABLE IF NOT EXISTS users (
  id bigint NOT NULL AUTO_INCREMENT,
  name varchar(255) NOT NULL,
  email varchar(255) NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 索引添加
ALTER TABLE users ADD INDEX IF NOT EXISTS idx_created_at (created_at);
```

#### ❌ 避免的做法
```sql
-- 避免：不使用幂等性检查
CREATE TABLE users (...);  -- 重复执行会报错

-- 避免：中文注释（可能导致编码问题）
CREATE TABLE users (
  id bigint COMMENT '用户ID'  -- 可能出现编码问题
);

-- 避免：复杂的数据迁移在结构迁移中
INSERT INTO users SELECT * FROM old_users;  -- 应该分开处理
```

### 2. 命名约定

```bash
# 好的迁移名称
./database/migrate.sh create create_users_table
./database/migrate.sh create add_email_index_to_users
./database/migrate.sh create update_users_add_phone_column
./database/migrate.sh create remove_deprecated_status_column

# 避免的名称
./database/migrate.sh create fix_bug        # 不够具体
./database/migrate.sh create 修复用户表      # 中文字符
./database/migrate.sh create temp_changes   # 临时更改应该避免
```

### 3. 回滚策略

每个 `.up.sql` 都必须有对应的 `.down.sql`：

```sql
-- 000005_add_user_status.up.sql
ALTER TABLE users ADD COLUMN status tinyint NOT NULL DEFAULT 1;
ALTER TABLE users ADD INDEX idx_status (status);

-- 000005_add_user_status.down.sql
ALTER TABLE users DROP INDEX idx_status;
ALTER TABLE users DROP COLUMN status;
```

### 4. 生产环境部署流程

#### 标准部署流程
```bash
# 步骤1: 备份数据库
mysqldump -u root -p task_center > backup_$(date +%Y%m%d_%H%M%S).sql

# 步骤2: 测试环境验证
export DOCKER_CONTAINER=test-mysql
./database/migrate.sh up
./database/migrate.sh status

# 步骤3: 生产环境执行（维护窗口）
export DOCKER_CONTAINER=prod-mysql
./database/migrate.sh status  # 查看当前状态
./database/migrate.sh up      # 执行迁移
./database/migrate.sh status  # 验证结果

# 步骤4: 应用重启和验证
```

#### 大表迁移策略
```bash
# 对于大表，考虑分步执行
./database/migrate.sh up 1    # 执行一个迁移
# 观察性能影响
./database/migrate.sh up 1    # 继续下一个
```

## 🔧 与 go-zero 集成

### 1. 项目依赖

在 `go.mod` 中添加依赖：

```go
module task-center

go 1.21

require (
    github.com/golang-migrate/migrate/v4 v4.19.0
    github.com/go-sql-driver/mysql v1.5.0
    github.com/zeromicro/go-zero v1.5.0  // go-zero 框架
)
```

### 2. 启动时自动迁移（推荐用于开发环境）

```go
// main.go 或 初始化代码
package main

import (
    "task-center/database"
    "database/sql"
    "log"
)

func main() {
    // 1. 创建数据库连接
    db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/task_center?parseTime=true")
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()

    // 2. 执行数据库迁移
    config := database.DefaultMigrationConfig()
    migrator := database.NewMigrationManager(db, config)

    if err := migrator.RunMigrations(); err != nil {
        log.Fatal("Migration failed:", err)
    }

    log.Println("Database migration completed successfully")

    // 3. 启动 go-zero 服务
    // ... 你的服务启动代码
}
```

### 3. 生成 goctl 模型代码

数据库迁移完成后，生成 go-zero 模型：

```bash
# 使用项目提供的无外键版本生成模型
cd database
goctl model mysql ddl -src="core_tables_no_fk.sql" -dir="../model" -c

# 检查生成的文件
ls -la ../model/
```

### 4. CI/CD 集成示例

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Setup Database
        run: |
          # 启动测试数据库
          docker run -d --name mysql-test \
            -e MYSQL_ROOT_PASSWORD=test123 \
            -e MYSQL_DATABASE=task_center \
            -p 3306:3306 mysql:8.0

          # 等待数据库启动
          sleep 30

      - name: Run Migrations
        run: |
          # 安装 golang-migrate
          go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

          # 执行迁移
          export DOCKER_CONTAINER=mysql-test
          ./database/migrate.sh up

      - name: Verify Migration
        run: |
          export DOCKER_CONTAINER=mysql-test
          ./database/migrate.sh status
```

## 🚨 故障排除

### 常见问题及解决方案

#### 1. 脏数据库状态
```bash
# 错误信息：Dirty database version X. Fix and force version.
# 解决方案：强制设置版本
./database/migrate.sh force 3

# 或者使用原生命令
migrate -database "mysql://user:pass@tcp(host:port)/db" -path database/migrations force 3
```

#### 2. 权限问题
```bash
# 确保数据库用户有足够权限
mysql -u root -p -e "
GRANT ALL PRIVILEGES ON task_center.* TO 'user'@'%';
FLUSH PRIVILEGES;
"
```

#### 3. 连接问题
```bash
# 测试数据库连接
./database/migrate.sh version

# 如果失败，检查环境变量
echo $DOCKER_CONTAINER
echo $DB_HOST $DB_PORT $DB_USER

# 手动测试连接
docker exec $DOCKER_CONTAINER mysql -u root -p$DB_PASSWORD -e "SELECT 1;"
```

#### 4. 迁移文件编码问题
```bash
# 检查文件编码
file database/migrations/*.sql

# 转换编码（如果需要）
iconv -f GB2312 -t UTF-8 file.sql > file_utf8.sql
```

#### 5. 外键约束问题
```bash
# 如果遇到外键约束错误，检查数据完整性
./database/migrate.sh status

# 强制禁用外键检查（谨慎使用）
mysql -u root -p task_center -e "SET FOREIGN_KEY_CHECKS=0;"
```

### 🆘 紧急恢复流程

如果迁移出现严重问题：

```bash
# 1. 立即停止应用
systemctl stop your-app

# 2. 从备份恢复数据库
mysql -u root -p -e "DROP DATABASE task_center;"
mysql -u root -p -e "CREATE DATABASE task_center;"
mysql -u root -p task_center < backup_20250915_120000.sql

# 3. 重置迁移状态
./database/migrate.sh force 0

# 4. 重新执行迁移
./database/migrate.sh up

# 5. 验证数据完整性
./database/migrate.sh status
```

## 📊 当前迁移状态

| 版本 | 名称 | 描述 | 状态 | 执行时间 |
|------|------|------|------|----------|
| 001 | create_business_systems_table | 业务系统表 | ✅ 已完成 | ~26ms |
| 002 | create_tasks_table | 任务表 | ✅ 已完成 | ~69ms |
| 003 | create_task_executions_table | 执行历史表 | ✅ 已完成 | ~105ms |
| 004 | create_task_locks_table | 任务锁表 | ✅ 已完成 | ~132ms |

**总执行时间**: < 350ms
**数据库表数**: 5 (含 schema_migrations)
**外键约束**: 3个

## ✅ 项目完成状态

### 🎉 已完成功能
- ✅ **golang-migrate 工具集成** - 企业级迁移管理
- ✅ **4个核心表迁移** - 完整的任务调度表结构
- ✅ **统一管理脚本** - migrate.sh 提供友好的操作界面
- ✅ **完整回滚支持** - 所有迁移都支持安全回滚
- ✅ **生产环境就绪** - 经过完整测试和验证
- ✅ **go-zero 集成** - 与框架无缝集成
- ✅ **详细文档** - 完整的使用指南和最佳实践

### 🎯 下一步行动
1. **开始 Issue #4** - 使用 goctl 生成模型代码
2. **API 服务开发** - 基于 task-center.api 生成服务
3. **业务逻辑实现** - 任务调度核心功能开发

## 🔗 相关资源

- 📖 [golang-migrate 官方文档](https://github.com/golang-migrate/migrate)
- 🔧 [MySQL 迁移最佳实践](https://github.com/golang-migrate/migrate/tree/master/database/mysql)
- 🚀 [go-zero 模型生成指南](https://go-zero.dev/docs/tutorials)
- 🏗️ [Task Center API 协议文档](../task-center.api)

---

> 💡 **提示**:
> - 生产环境建议在维护窗口执行迁移
> - 大表迁移前务必进行性能测试
> - 建议在 CI/CD 中集成迁移验证步骤

*最后更新: 2025-09-15 | golang-migrate v4.19.0*