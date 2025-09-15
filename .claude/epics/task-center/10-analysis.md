---
issue: 10
epic: task-center
analyzed: 2025-09-15T13:55:55Z
streams: 4
parallel_ready: 3
---

# Issue #10 Analysis: 监控告警系统集成

## Work Streams

### Stream A: Prometheus 指标暴露和业务指标收集
- **Agent**: general-purpose
- **Scope**: 实现 Prometheus metrics 端点，收集任务执行、回调、业务系统等核心业务指标
- **Files**: `internal/metrics/*.go`, `internal/handler/metrics.go`, `api/metrics.api`
- **Dependencies**: None
- **Ready**: Yes

### Stream B: 系统性能监控和健康检查
- **Agent**: general-purpose
- **Scope**: 实现系统性能指标收集(HTTP QPS、数据库连接池、内存CPU)和健康检查机制
- **Files**: `internal/health/*.go`, `internal/middleware/metrics.go`, `internal/handler/health.go`
- **Dependencies**: None
- **Ready**: Yes

### Stream C: Asynq 任务队列监控集成
- **Agent**: general-purpose
- **Scope**: 集成 Asynq 任务队列监控指标，包括队列长度、Worker 处理速度、重试次数统计
- **Files**: `internal/queue/metrics.go`, `internal/worker/metrics.go`
- **Dependencies**: Stream A (共享 metrics registry)
- **Ready**: No

### Stream D: 告警配置和可视化监控
- **Agent**: general-purpose
- **Scope**: 配置 Prometheus 告警规则、Grafana 仪表板模板和结构化日志输出
- **Files**: `deploy/prometheus/*.yml`, `deploy/grafana/*.json`, `internal/logger/structured.go`
- **Dependencies**: Stream A, Stream B (需要指标定义完成)
- **Ready**: No

## Coordination Notes
- Stream A 需要首先建立 metrics registry 和基础架构，为其他 streams 提供基础
- Stream B 可以独立开发健康检查，但性能指标需要复用 Stream A 的 registry
- Stream C 的 Asynq 监控需要等待 Stream A 的指标基础设施完成
- Stream D 的告警规则和仪表板配置需要等待前面 streams 定义的指标名称和标签
- 所有 streams 共享相同的日志格式规范，需要在开始前协调统一
- 考虑使用 go-zero 框架内置的 metrics 中间件来减少重复工作