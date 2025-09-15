# 🗄️ Task Center 数据库设置指南

## 📋 快速开始 (5分钟)

Task Center 使用 **golang-migrate** 进行数据库版本管理，提供企业级的迁移体验。

### 🚀 一键设置

```bash
# 1. 确保 MySQL 运行中
docker ps | grep mysql

# 2. 设置环境变量
export DOCKER_CONTAINER=your-mysql-container-name

# 3. 执行数据库迁移
./database/migrate.sh up

# 4. 验证结果
./database/migrate.sh status
```

成功后应该看到：
- ✅ **4个核心表**: business_systems, tasks, task_executions, task_locks
- ✅ **外键约束**: 3个数据一致性保护
- ✅ **当前版本**: 4

## 📁 项目结构

```
task-center/
├── database/
│   ├── migrations/           # 🔄 迁移文件 (golang-migrate)
│   ├── migrate.sh           # 🛠️ 统一管理脚本
│   ├── integration.go       # 🔌 Go 代码集成
│   ├── core_tables_no_fk.sql  # 📝 goctl 模型生成
│   └── README_GOLANG_MIGRATE.md  # 📖 详细文档
├── task-center.api          # 🌐 API 协议定义
└── model/                   # 🏗️ 生成的 Go 模型 (待生成)
```

## 🛠️ 常用操作

### 开发环境
```bash
# 执行迁移
./database/migrate.sh up

# 查看状态
./database/migrate.sh status

# 创建新迁移
./database/migrate.sh create add_new_feature
```

### 生产环境
```bash
# 备份数据库
mysqldump -u root -p task_center > backup_$(date +%Y%m%d_%H%M%S).sql

# 执行迁移
./database/migrate.sh up

# 验证结果
./database/migrate.sh status
```

## 🔧 与 go-zero 集成

### 生成模型代码
```bash
cd database
goctl model mysql ddl -src="core_tables_no_fk.sql" -dir="../model" -c
```

### 代码中使用
```go
import "task-center/database"

// 自动迁移（开发环境）
migrator := database.NewMigrationManager(db, database.DefaultMigrationConfig())
err := migrator.RunMigrations()
```

## 📊 数据库架构

### 核心表关系
```
business_systems (业务系统)
    ↓ (1:N)
tasks (任务)
    ↓ (1:N)
task_executions (执行历史)

tasks (任务)
    ↓ (1:N)
task_locks (分布式锁)
```

### 表概览
| 表名 | 用途 | 主要字段 |
|------|------|----------|
| business_systems | 业务系统管理 | business_code, api_key, rate_limit |
| tasks | 任务信息 | callback_url, status, priority, scheduled_at |
| task_executions | 执行历史 | task_id, http_status, duration, execution_time |
| task_locks | 分布式锁 | task_id, lock_key, node_id, expires_at |

## ⚠️ 重要提醒

### 生产环境注意事项
- 🔒 **备份优先**: 迁移前必须备份数据库
- 🕐 **维护窗口**: 建议在低峰期执行
- 📊 **性能测试**: 大表迁移前进行性能评估
- 🔄 **回滚准备**: 确保回滚脚本可用

### 开发最佳实践
- ✅ **幂等设计**: 使用 `IF NOT EXISTS` 确保可重复执行
- 📝 **命名规范**: 使用语义化的迁移名称
- 🔄 **完整回滚**: 每个 up 迁移都要有对应的 down
- 🧪 **测试验证**: 在测试环境验证迁移

## 🆘 故障排除

### 常见问题
```bash
# 脏状态修复
./database/migrate.sh force 3

# 权限问题
GRANT ALL PRIVILEGES ON task_center.* TO 'user'@'%';

# 连接测试
./database/migrate.sh version
```

### 紧急恢复
```bash
# 1. 停止应用
systemctl stop task-center

# 2. 恢复备份
mysql -u root -p task_center < backup_20250915.sql

# 3. 重置迁移
./database/migrate.sh force 0
./database/migrate.sh up
```

## 📚 更多资源

- 📖 **详细文档**: [database/README_GOLANG_MIGRATE.md](database/README_GOLANG_MIGRATE.md)
- 🔧 **API 协议**: [task-center.api](task-center.api)
- 🏗️ **go-zero 文档**: https://go-zero.dev/docs/tutorials

---

> 💡 **提示**: 如遇问题，请查看详细文档或联系开发团队。数据库迁移功能已经过完整测试，可放心使用。

*Task Center Database v4 | 基于 golang-migrate v4.19.0*