---
issue: 5
title: 使用 goctl 生成数据模型代码
analyzed: 2025-09-15T07:31:18Z
estimated_hours: 8
parallelization_factor: 2.5
---

# 并行工作分析: Issue #5

## 概述
基于已完成的数据库DDL脚本（Issue #3），使用goctl工具生成完整的数据模型代码。生成所有表的Model结构体、基础CRUD操作、自定义查询方法和数据库连接管理，确保数据访问层符合go-zero开发规范。

## 并行工作流

### Stream A: 模型代码生成与结构建立
**范围**: 使用goctl工具生成核心数据模型，建立基础Model结构体和CRUD方法
**文件**:
- internal/model/business_systems_model.go
- internal/model/tasks_model.go
- internal/model/task_executions_model.go
- internal/model/task_locks_model.go
- internal/model/vars.go
**Agent类型**: general-purpose
**可开始时间**: 立即开始
**估计工时**: 3小时
**依赖**: 无（DDL文件已就绪）

### Stream B: 数据库连接池与配置优化
**范围**: 配置数据库连接池、超时设置、事务管理和ServiceContext集成
**文件**:
- internal/svc/serviceContext.go (扩展)
- internal/config/config.go (数据库配置)
- etc/taskcenter.yaml (数据源配置)
**Agent类型**: general-purpose
**可开始时间**: Stream A完成模型生成后
**估计工时**: 2.5小时
**依赖**: Stream A (需要模型代码)

### Stream C: 单元测试与验证
**范围**: 为生成的模型编写单元测试，验证CRUD操作和数据访问方法
**文件**:
- internal/model/business_systems_model_test.go
- internal/model/tasks_model_test.go
- internal/model/task_executions_model_test.go
- internal/model/task_locks_model_test.go
**Agent类型**: test-runner
**可开始时间**: Stream A完成后
**估计工时**: 2.5小时
**依赖**: Stream A (需要模型代码)

## 协调要点

### 共享文件
以下文件需要多个Stream修改，需要协调:
- `internal/svc/serviceContext.go` - Stream A & B (模型集成与连接池配置)
- `internal/config/config.go` - Stream B (数据库连接配置)

### 顺序要求
必须按以下顺序执行:
1. Stream A: 模型代码生成 (goctl工具生成基础结构)
2. Stream B & C: 并行执行 (B负责连接配置，C负责测试验证)

## 冲突风险评估
- **低风险**: 大部分文件为goctl工具生成，不会冲突
- **中风险**: serviceContext.go需要协调模型集成和配置集成
- **低风险**: 测试文件独立，不会产生冲突

## 并行化策略

**推荐方法**: 混合模式

阶段1: Stream A独立执行 (生成模型代码)
阶段2: Stream B & C并行执行 (B配置连接，C编写测试)

## 预期时间线

**并行执行**:
- 墙钟时间: 5.5小时 (3h + max(2.5h, 2.5h))
- 总工作量: 8小时
- 效率提升: 31%

**顺序执行**:
- 墙钟时间: 8小时

## 技术实施要点

### goctl工具使用
- 使用DDL文件生成: `goctl model mysql ddl -src="database/core_tables_no_fk.sql" -dir="./internal/model" -c`
- 确保使用无外键版本的DDL文件（与Issue #3的双版本策略配合）
- 验证生成代码符合go-zero规范

### 数据库连接优化
- 连接池大小: MaxOpenConns=20, MaxIdleConns=10
- 连接超时: ConnMaxLifetime=30分钟
- 查询超时: 10秒
- 支持读写分离配置（为后续扩展预留）

### 测试覆盖策略
- 单元测试覆盖所有CRUD操作
- 包含数据验证和边界测试
- 使用测试数据库，不影响开发环境
- 集成go-zero的测试最佳实践

## 注意事项
- 严格使用goctl工具生成，不手工创建模型文件
- 保持生成代码原样，业务扩展在logic层实现
- 数据库配置必须与Issue #4的配置保持一致
- 测试使用独立的测试数据库，避免污染开发数据