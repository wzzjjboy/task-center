---
issue: 8
title: HTTP 回调执行系统实现
analyzed: 2025-09-15T10:33:48Z
estimated_hours: 28
parallelization_factor: 3.5
---

# 并行工作分析: Issue #8

## 概述
实现高效可靠的HTTP回调执行系统，构建task-center的核心回调引擎。负责向业务系统发送定时回调请求，包括HTTP客户端配置、请求重试机制、响应处理、错误恢复和性能优化，确保回调的可靠性和及时性。与Issue #7的任务调度引擎配合，形成完整的任务执行闭环。

## 并行工作流

### Stream A: HTTP客户端核心实现
**范围**: 实现HTTP客户端核心功能，包括连接池管理、请求配置、超时控制和基础请求发送能力
**文件**:
- internal/callback/http_client.go
- internal/callback/connection_pool.go
- internal/config/config.go (HTTP配置扩展)
- internal/svc/servicecontext.go (HTTP客户端集成)
- etc/taskcenter.yaml (HTTP配置)
**Agent类型**: general-purpose
**可开始时间**: 立即开始
**估计工时**: 8小时
**依赖**: 无（基础框架已就绪）

### Stream B: 回调执行引擎实现
**范围**: 实现回调执行核心逻辑，包括任务处理器、回调请求发送、响应处理和结果存储
**文件**:
- internal/logic/callback/callback_logic.go
- internal/handler/callback/callback_handler.go
- internal/callback/task_processor.go
- internal/types/callback_types.go
**Agent类型**: general-purpose
**可开始时间**: Stream A完成HTTP客户端后
**估计工时**: 9小时
**依赖**: Stream A (需要HTTP客户端)

### Stream C: 重试机制和错误处理
**范围**: 实现智能重试策略、错误分类处理、熔断保护和故障恢复机制
**文件**:
- internal/callback/retry_handler.go
- internal/callback/error_handler.go
- internal/callback/circuit_breaker.go
- internal/types/retry_callback_types.go
**Agent类型**: general-purpose
**可开始时间**: Stream A完成后与Stream B并行
**估计工时**: 7小时
**依赖**: Stream A (需要HTTP客户端基础)

### Stream D: 性能监控和统计
**范围**: 实现回调性能监控、统计指标收集、健康检查和管理接口
**文件**:
- internal/callback/monitor.go
- internal/logic/callback/stats_logic.go
- internal/handler/callback/stats_handler.go
- internal/types/monitor_types.go
**Agent类型**: general-purpose
**可开始时间**: Stream B完成基础执行逻辑后
**估计工时**: 4小时
**依赖**: Stream B (需要回调执行逻辑)

## 协调要点

### 共享文件
以下文件需要多个Stream修改，需要协调:
- `internal/config/config.go` - Stream A (HTTP配置)
- `internal/svc/servicecontext.go` - Stream A (客户端集成)
- `internal/types/types.go` - Stream B, C, D (类型定义)
- `etc/taskcenter.yaml` - Stream A (配置文件)

### 顺序要求
必须按以下顺序执行:
1. Stream A: HTTP客户端核心 (建立基础设施)
2. Stream B & C: 并行执行 (回调逻辑 + 重试错误处理)
3. Stream D: 监控统计 (依赖回调执行逻辑)

## 冲突风险评估
- **低风险**: 大部分文件在不同目录，冲突风险较低
- **中风险**: types文件需要协调，但可通过模块化避免冲突
- **低风险**: 配置文件修改在不同阶段进行

## 并行化策略

**推荐方法**: 分层并行

**第一层**: Stream A 独立执行 (HTTP客户端核心)
**第二层**: Stream B & C 并行执行 (回调逻辑 + 重试处理)
**第三层**: Stream D 执行 (监控统计)

## 预期时间线

**并行执行**:
- 墙钟时间: 21小时 (8h + max(9h, 7h) + 4h)
- 总工作量: 28小时
- 效率提升: 25%

**顺序执行**:
- 墙钟时间: 28小时

## 技术实施要点

### HTTP客户端架构
- 使用Go标准库net/http包构建
- 连接池: MaxIdleConns=100, MaxIdleConnsPerHost=20
- 超时配置: ConnectTimeout=10s, RequestTimeout=30s
- Keep-Alive: 启用，保持时间90秒
- 支持HTTP/1.1和HTTP/2

### 回调执行设计
- 与Issue #7的任务调度引擎集成
- 支持GET/POST/PUT/PATCH/DELETE方法
- 请求体支持JSON/XML/Form格式
- Header管理: 自定义Header + 认证Header
- 回调上下文: business_id, task_id, trace_id传递

### 重试策略配置
- 指数退避: 初始1秒，最大60秒，倍数1.5
- 最大重试: 默认3次，可按业务配置
- 重试条件: 5xx状态码、网络错误、超时
- 熔断保护: 连续失败阈值、恢复检查机制
- 死信处理: 超过重试次数的回调记录

### 响应处理机制
- 状态码分类: 2xx(成功), 3xx(重定向), 4xx(客户端错误), 5xx(服务端错误)
- 响应体限制: 最大1MB，超出截断
- 响应时间记录: 精确到毫秒
- 成功判断: 可配置的成功状态码范围
- 数据持久化: 执行记录存储到task_executions表

### 性能监控指标
- QPS统计: 请求每秒数
- 延迟指标: P50, P90, P95, P99响应时间
- 成功率: 按状态码分类的成功率统计
- 错误分布: 网络错误、超时、4xx、5xx分类
- 连接池状态: 活跃连接、空闲连接数量

### 并发控制和优化
- 工作协程池: 默认50个并发worker
- 请求限流: 每个目标主机的并发限制
- 内存优化: 对象池复用、及时GC
- 批量处理: 支持批量回调任务处理
- 优雅关闭: 等待进行中请求完成

## 与Issue #7的集成

### 任务调度集成
- 复用Asynq任务队列系统
- 注册回调任务处理器到任务调度引擎
- 使用相同的重试和错误处理机制
- 共享Redis连接和配置

### 任务类型扩展
- 新增CallbackTask任务类型
- 支持延时回调和定时回调
- 任务优先级和队列配置
- 任务取消和状态跟踪

## 测试策略

### 单元测试覆盖
- HTTP客户端连接和请求测试
- 重试机制和错误处理测试
- 响应解析和数据存储测试
- 监控统计功能测试

### 集成测试场景
- 端到端回调执行流程
- 异常场景处理 (超时、网络错误、服务不可用)
- 并发回调性能测试
- 故障恢复和重试验证

### 性能测试要求
- 支持1000+ 并发回调
- 平均响应时间: < 100ms
- P99响应时间: < 500ms
- 内存使用: < 200MB (1000个活跃任务)
- CPU使用率: < 50% (正常负载)

## 安全考虑

### 数据安全
- 敏感数据脱敏记录
- HTTPS支持和证书验证
- 请求体大小限制
- 防SSRF攻击保护

### 访问控制
- 目标URL白名单机制
- 请求Header过滤
- 速率限制和防滥用
- 审计日志记录

## 注意事项
- HTTP客户端要考虑目标服务的负载能力
- 重试机制要避免对目标服务造成压力
- 监控指标要及时清理历史数据
- 错误日志要包含足够的调试信息
- 考虑网络分区和服务降级场景
- 支持动态配置更新，无需重启服务