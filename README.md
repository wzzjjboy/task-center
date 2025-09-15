# TaskCenter Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/your-org/task-center/sdk.svg)](https://pkg.go.dev/github.com/your-org/task-center/sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/task-center/sdk)](https://goreportcard.com/report/github.com/your-org/task-center/sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

TaskCenter 官方 Go SDK，提供完整的任务调度和管理功能。

## 特性

- 🚀 **简单易用**: 直观的 API 设计，快速上手
- 🔒 **安全认证**: 基于 API Key 的安全认证机制
- 🔄 **智能重试**: 可配置的重试策略和错误处理
- 📊 **批量操作**: 支持批量创建和管理任务
- 🎯 **优先级管理**: 灵活的任务优先级控制
- 📞 **回调处理**: 内置回调服务器和签名验证
- 🏷️ **标签系统**: 基于标签的任务分类和查询
- ⏰ **定时任务**: 支持延时和定时任务执行
- 🔍 **完整监控**: 任务状态跟踪和统计信息
- 🧪 **测试友好**: 完整的测试套件和模拟工具

## 快速开始

### 安装

```bash
go get github.com/your-org/task-center/sdk
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/your-org/task-center/sdk"
)

func main() {
    // 创建客户端
    client, err := sdk.NewClientWithDefaults(
        "http://localhost:8080", // TaskCenter 服务地址
        "your-api-key",          // API Key
        123,                     // 业务系统 ID
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 创建任务
    task := sdk.NewTask("order-123", "https://api.example.com/webhook")

    createdTask, err := client.Tasks().Create(context.Background(), task)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Task created: %d\n", createdTask.ID)
}
```

### 高级用法

```go
// 创建复杂任务
task := sdk.NewTask("payment-456", "https://api.example.com/payment").
    WithPriority(sdk.TaskPriorityHigh).                    // 高优先级
    WithTimeout(30).                                       // 30秒超时
    WithRetries(3, sdk.StandardRetryIntervals).           // 标准重试策略
    WithTags("payment", "critical").                       // 添加标签
    WithHeaders(map[string]string{                         // 自定义请求头
        "Authorization": "Bearer token",
    }).
    WithBody(`{"amount": 100, "currency": "USD"}`).       // 请求体
    WithMetadata(map[string]interface{}{                   // 元数据
        "user_id": 12345,
        "order_id": "ORD-456",
    })

createdTask, err := client.Tasks().Create(ctx, task)
```

### 回调处理

```go
// 创建回调处理器
handler := &sdk.DefaultCallbackHandler{
    OnTaskCompleted: func(event *sdk.CallbackEvent) error {
        fmt.Printf("Task %d completed\n", event.TaskID)
        return nil
    },
    OnTaskFailed: func(event *sdk.CallbackEvent) error {
        fmt.Printf("Task %d failed: %s\n", event.TaskID, event.Task.ErrorMessage)
        return nil
    },
}

// 创建回调服务器
server := sdk.NewCallbackServer("your-api-secret", handler)

// 启动服务器
log.Fatal(http.ListenAndServe(":8080", server))
```

## 文档

- 📚 [API 文档](docs/API.md) - 完整的 API 参考
- 🚀 [快速开始指南](GETTING_STARTED.md) - 详细的入门教程
- 💡 [示例代码](examples/) - 各种使用场景的示例
- 🧪 [测试指南](docs/TESTING.md) - 测试编写和运行指南

## 示例

我们提供了丰富的示例代码来帮助您快速上手：

- [基础示例](examples/basic/) - SDK 基本功能演示
- [高级示例](examples/advanced/) - 复杂场景和最佳实践
- [回调服务器](examples/callback_server/) - 完整的回调处理实现
- [完整工作流](examples/complete_workflow/) - 电商订单处理工作流

运行示例：

```bash
# 设置环境变量
export TASKCENTER_API_URL="http://localhost:8080"
export TASKCENTER_API_KEY="your-api-key"
export TASKCENTER_BUSINESS_ID="123"

# 运行基础示例
cd examples/basic
go run main.go

# 运行回调服务器
cd examples/callback_server
export TASKCENTER_API_SECRET="your-secret"
go run main.go
```

## API 概览

### 客户端管理

```go
// 创建客户端
client, err := sdk.NewClientWithDefaults(apiURL, apiKey, businessID)
client, err := sdk.NewClient(config)

// 获取默认配置
config := sdk.DefaultConfig()
```

### 任务操作

```go
// 创建任务
task := sdk.NewTask("business-id", "callback-url")
createdTask, err := client.Tasks().Create(ctx, task)

// 查询任务
task, err := client.Tasks().Get(ctx, taskID)
task, err := client.Tasks().GetByBusinessUniqueID(ctx, "business-id")

// 更新任务
updateReq := &sdk.UpdateTaskRequest{Priority: &sdk.TaskPriorityHigh}
updatedTask, err := client.Tasks().Update(ctx, taskID, updateReq)

// 列出任务
listReq := sdk.NewListTasksRequest().WithStatus(sdk.TaskStatusPending)
response, err := client.Tasks().List(ctx, listReq)

// 批量创建
batchReq := &sdk.BatchCreateTasksRequest{Tasks: tasks}
response, err := client.Tasks().BatchCreate(ctx, batchReq)

// 任务控制
err := client.Tasks().Cancel(ctx, taskID)
err := client.Tasks().Retry(ctx, taskID)
err := client.Tasks().Delete(ctx, taskID)

// 获取统计
stats, err := client.Tasks().Stats(ctx)
```

### 任务构建器

```go
// 基本任务
task := sdk.NewTask("id", "url")

// 带回调配置
task := sdk.NewTaskWithCallback("id", "url", "POST", headers, body)

// 定时任务
task := sdk.NewScheduledTask("id", "url", time.Now().Add(1*time.Hour))

// 链式配置
task := sdk.NewTask("id", "url").
    WithPriority(sdk.TaskPriorityHigh).
    WithTimeout(60).
    WithRetries(3, sdk.FastRetryIntervals).
    WithTags("tag1", "tag2").
    WithMetadata(metadata)
```

### 查询构建器

```go
// 任务列表查询
req := sdk.NewListTasksRequest().
    WithStatus(sdk.TaskStatusPending, sdk.TaskStatusRunning).
    WithTagsFilter("payment").
    WithPriorityFilter(sdk.TaskPriorityHigh).
    WithDateRange(from, to).
    WithPagination(1, 20)
```

### 错误处理

```go
_, err := client.Tasks().Create(ctx, task)
if err != nil {
    switch {
    case sdk.IsValidationError(err):
        // 处理验证错误
    case sdk.IsAuthenticationError(err):
        // 处理认证错误
    case sdk.IsRateLimitError(err):
        // 处理限流错误
    case sdk.IsRetryableError(err):
        // 可重试的错误
    default:
        // 其他错误
    }
}
```

## 配置

### 环境变量

```bash
# 必需配置
export TASKCENTER_API_URL="http://localhost:8080"
export TASKCENTER_API_KEY="your-api-key"
export TASKCENTER_BUSINESS_ID="123"

# 可选配置 - 回调服务器
export TASKCENTER_API_SECRET="your-webhook-secret"
export CALLBACK_PORT="8080"
```

### 客户端配置

```go
config := sdk.DefaultConfig()
config.BaseURL = "https://taskcenter.example.com"
config.APIKey = "your-api-key"
config.BusinessID = 123
config.Timeout = 60 * time.Second

// 自定义重试策略
config.RetryPolicy = &sdk.RetryPolicy{
    MaxRetries:      5,
    InitialInterval: 2 * time.Second,
    MaxInterval:     30 * time.Second,
    Multiplier:      2.0,
    RetryableErrors: []int{429, 500, 502, 503, 504},
}

client, err := sdk.NewClient(config)
```

## 测试

```bash
# 运行单元测试
cd sdk
go test -v

# 运行集成测试 (需要 TaskCenter 服务)
export TASKCENTER_API_URL="http://localhost:8080"
export TASKCENTER_API_KEY="test-key"
export TASKCENTER_BUSINESS_ID="1"
go test -tags=integration -v

# 生成覆盖率报告
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

更多测试信息请参考 [测试指南](docs/TESTING.md)。

## 最佳实践

### 1. 错误处理

```go
// 总是检查错误并根据类型处理
task, err := client.Tasks().Create(ctx, req)
if err != nil {
    if sdk.IsRetryableError(err) {
        // 可以重试
        return handleRetryableError(err)
    }
    return fmt.Errorf("failed to create task: %w", err)
}
```

### 2. 资源管理

```go
// 确保正确关闭客户端
client, err := sdk.NewClient(config)
if err != nil {
    return err
}
defer client.Close()
```

### 3. 回调安全

```go
// 使用强密钥
apiSecret := os.Getenv("TASKCENTER_API_SECRET")
if len(apiSecret) < 32 {
    log.Fatal("API secret must be at least 32 characters")
}
```

### 4. 任务设计

```go
// 使用合适的优先级
task.WithPriority(sdk.TaskPriorityHigh)    // 关键业务
task.WithPriority(sdk.TaskPriorityNormal)  // 一般任务
task.WithPriority(sdk.TaskPriorityLow)     // 后台任务

// 设置合理的超时时间
task.WithTimeout(30)   // 30秒，适合实时任务
task.WithTimeout(300)  // 5分钟，适合一般任务
task.WithTimeout(1800) // 30分钟，适合长时间任务

// 使用标签分类
task.WithTags("payment", "critical", "user-premium")
```

## 性能考虑

- **批量操作**: 对于大量任务使用批量创建接口
- **连接复用**: 客户端会自动复用HTTP连接
- **合理超时**: 根据业务需求设置合适的超时时间
- **错误重试**: 使用预定义的重试策略避免过度重试

## 版本兼容性

| SDK 版本 | Go 版本 | TaskCenter API |
|----------|---------|----------------|
| 1.0.x    | 1.18+   | v1             |

## 更新日志

### v1.0.0 (Latest)
- 🎉 初始发布
- ✅ 完整的任务管理功能
- ✅ 回调处理支持
- ✅ 错误处理和重试机制
- ✅ 批量操作支持
- ✅ 完整的测试套件
- ✅ 详细的文档和示例

## 贡献

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

在提交之前，请确保：

- [ ] 代码通过所有测试
- [ ] 添加了适当的测试用例
- [ ] 更新了相关文档
- [ ] 遵循现有的代码风格

## 支持

如果您遇到问题或需要帮助：

1. 📖 查看 [文档](docs/)
2. 🔍 搜索现有的 [Issues](https://github.com/your-org/task-center/issues)
3. 💬 创建新的 [Issue](https://github.com/your-org/task-center/issues/new)
4. 📧 联系技术支持: support@example.com

## 许可证

本项目使用 [MIT 许可证](LICENSE)。

## 致谢

感谢所有为 TaskCenter Go SDK 做出贡献的开发者！

---

<div align="center">
  <strong>Made with ❤️ by TaskCenter Team</strong>
</div>