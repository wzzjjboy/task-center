# TaskCenter Go SDK 示例

本目录包含了 TaskCenter Go SDK 的完整使用示例，展示了从基础功能到复杂工作流的各种使用场景。

## 📁 示例目录

### 1. [基础示例](basic/)
- **文件**: `basic/main.go`
- **内容**: SDK 的基本使用方法
- **功能**:
  - 客户端创建和配置
  - 简单任务创建
  - 任务查询
  - 定时任务
  - 任务列表

**运行方式**:
```bash
cd examples/basic
go run main.go
```

### 2. [高级示例](advanced/)
- **文件**: `advanced/main.go`
- **内容**: SDK 的高级功能演示
- **功能**:
  - 复杂任务配置
  - 批量操作
  - 任务管理
  - 错误处理
  - 统计信息

**运行方式**:
```bash
cd examples/advanced
go run main.go
```

### 3. [回调服务器](callback_server/)
- **文件**: `callback_server/main.go`
- **内容**: 完整的回调处理服务器
- **功能**:
  - 回调事件处理
  - 签名验证
  - 中间件使用
  - 优雅关闭
  - 业务逻辑集成

**运行方式**:
```bash
cd examples/callback_server
export TASKCENTER_API_SECRET="your-secret-key"
go run main.go
```

### 4. [完整工作流](complete_workflow/)
- **文件**: `complete_workflow/main.go`
- **内容**: 电商订单处理的完整工作流
- **功能**:
  - 多步骤任务编排
  - 任务依赖管理
  - 批量后续任务
  - 工作流监控

**运行方式**:
```bash
cd examples/complete_workflow
go run main.go
```

## 🔧 环境变量配置

所有示例都支持通过环境变量进行配置。创建 `.env` 文件或设置以下变量：

```bash
# TaskCenter 服务配置
export TASKCENTER_API_URL="http://localhost:8080"
export TASKCENTER_API_KEY="your-api-key"
export TASKCENTER_BUSINESS_ID="123"

# 回调服务器配置
export TASKCENTER_API_SECRET="your-webhook-secret"
export CALLBACK_PORT="8080"

# 可选的外部服务配置
export WEBHOOK_TOKEN="your-webhook-token"
export PAYMENT_API_TOKEN="your-payment-token"
```

## 🚀 快速开始

1. **安装依赖**:
   ```bash
   go mod init your-project
   go get github.com/your-org/task-center/sdk
   ```

2. **设置环境变量**:
   ```bash
   export TASKCENTER_API_URL="http://localhost:8080"
   export TASKCENTER_API_KEY="your-api-key"
   export TASKCENTER_BUSINESS_ID="123"
   ```

3. **运行基础示例**:
   ```bash
   cd examples/basic
   go run main.go
   ```

## 📖 示例说明

### 基础示例 (basic/)

这是最简单的入门示例，展示了：

```go
// 创建客户端
client, err := sdk.NewClientWithDefaults(apiURL, apiKey, businessID)

// 创建任务
task := sdk.NewTask("my-task", "https://api.example.com/webhook")
createdTask, err := client.Tasks().Create(ctx, task)

// 查询任务
task, err := client.Tasks().Get(ctx, taskID)
```

### 高级示例 (advanced/)

展示了更复杂的使用场景：

```go
// 复杂任务配置
task := sdk.NewTask("payment-task", "https://api.example.com/payment").
    WithPriority(sdk.TaskPriorityHigh).
    WithTimeout(30).
    WithRetries(3, sdk.FastRetryIntervals).
    WithTags("payment", "critical").
    WithHeaders(map[string]string{
        "Authorization": "Bearer token",
    }).
    WithBody(`{"amount": 100}`)

// 批量创建任务
batchReq := &sdk.BatchCreateTasksRequest{Tasks: tasks}
response, err := client.Tasks().BatchCreate(ctx, batchReq)

// 错误处理
if sdk.IsValidationError(err) {
    // 处理验证错误
}
```

### 回调服务器 (callback_server/)

完整的 webhook 服务器实现：

```go
// 创建回调处理器
handler := &sdk.DefaultCallbackHandler{
    OnTaskCompleted: func(event *sdk.CallbackEvent) error {
        fmt.Printf("Task %d completed\n", event.TaskID)
        return nil
    },
}

// 创建服务器
server := sdk.NewCallbackServer(apiSecret, handler)

// 启动服务器
http.ListenAndServe(":8080", server)
```

### 完整工作流 (complete_workflow/)

电商订单处理的端到端示例：

```go
// 1. 支付任务
paymentTask := sdk.NewTask(orderID+"-payment", paymentWebhook).
    WithPriority(sdk.TaskPriorityHighest)

// 2. 库存任务
inventoryTask := sdk.NewTask(orderID+"-inventory", inventoryWebhook).
    WithPriority(sdk.TaskPriorityHigh)

// 3. 定时配送任务
shippingTask := sdk.NewScheduledTask(orderID+"-shipping",
    shippingWebhook, time.Now().Add(1*time.Hour))

// 4. 批量营销任务
followUpTasks := createMarketingTasks(orderID)
batchResponse, err := client.Tasks().BatchCreate(ctx, batchReq)
```

## 🔄 常见使用模式

### 1. 重试策略

```go
// 快速重试（适合实时任务）
task.WithRetries(3, sdk.FastRetryIntervals)  // [10s, 30s, 60s]

// 标准重试（适合一般任务）
task.WithRetries(3, sdk.StandardRetryIntervals)  // [1m, 5m, 15m]

// 慢速重试（适合后台任务）
task.WithRetries(2, sdk.SlowRetryIntervals)  // [5m, 30m, 2h]
```

### 2. 任务优先级

```go
// 关键业务任务
task.WithPriority(sdk.TaskPriorityHighest)  // 支付、安全相关

// 重要任务
task.WithPriority(sdk.TaskPriorityHigh)     // 订单处理、通知

// 普通任务
task.WithPriority(sdk.TaskPriorityNormal)   // 一般业务逻辑

// 后台任务
task.WithPriority(sdk.TaskPriorityLow)      // 分析、清理
```

### 3. 标签使用

```go
// 按业务分类
task.WithTags("payment", "order", "user-premium")

// 按紧急程度
task.WithTags("critical", "real-time")

// 按处理类型
task.WithTags("webhook", "email", "sms")
```

### 4. 元数据

```go
task.WithMetadata(map[string]interface{}{
    "user_id":      12345,
    "order_id":     "ORD-123",
    "amount":       99.99,
    "currency":     "USD",
    "region":       "us-east-1",
    "retry_count":  0,
})
```

## 🛠 故障排除

### 常见错误

1. **认证失败**:
   ```
   Error: authentication failed
   ```
   检查 `TASKCENTER_API_KEY` 是否正确设置。

2. **验证错误**:
   ```
   Error: business_unique_id is required
   ```
   确保任务的 `BusinessUniqueID` 不为空且唯一。

3. **网络错误**:
   ```
   Error: network error
   ```
   检查 `TASKCENTER_API_URL` 是否可访问。

4. **回调签名错误**:
   ```
   Error: invalid signature
   ```
   检查 `TASKCENTER_API_SECRET` 是否与服务端配置一致。

### 调试技巧

1. **启用详细日志**:
   ```go
   config := sdk.DefaultConfig()
   config.RetryPolicy.MaxRetries = 0  // 禁用重试以查看原始错误
   ```

2. **检查任务状态**:
   ```go
   task, err := client.Tasks().Get(ctx, taskID)
   if err == nil {
       fmt.Printf("Status: %s, Error: %s\n",
           task.Status.String(), task.ErrorMessage)
   }
   ```

3. **使用测试回调**:
   使用 `https://httpbin.org/post` 作为测试回调 URL。

## 📚 更多资源

- [API 文档](../docs/API.md)
- [快速开始指南](../GETTING_STARTED.md)
- [最佳实践](../docs/BEST_PRACTICES.md)
- [FAQ](../docs/FAQ.md)

## 🤝 贡献

欢迎提交新的示例和改进建议！请遵循以下格式：

1. 在相应目录创建新的示例文件
2. 添加清晰的注释和说明
3. 更新本 README 文档
4. 提交 Pull Request

## 📄 许可证

MIT License