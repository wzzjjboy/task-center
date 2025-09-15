# TaskCenter Go SDK 快速开始指南

本指南将帮助您快速开始使用 TaskCenter Go SDK 集成任务调度功能。

## 前置条件

- Go 1.18 或更高版本
- TaskCenter 服务实例
- 有效的 API Key 和业务系统 ID

## 安装

```bash
go get github.com/your-org/task-center/sdk
```

## 基础设置

### 1. 导入包

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/your-org/task-center/sdk"
)
```

### 2. 配置环境变量

创建 `.env` 文件或设置环境变量：

```bash
export TASKCENTER_API_URL="http://localhost:8080"
export TASKCENTER_API_KEY="your-api-key-here"
export TASKCENTER_BUSINESS_ID="123"
export TASKCENTER_API_SECRET="your-webhook-secret-here"
```

### 3. 创建客户端

```go
func createClient() (*sdk.Client, error) {
    // 从环境变量读取配置
    apiURL := os.Getenv("TASKCENTER_API_URL")
    apiKey := os.Getenv("TASKCENTER_API_KEY")
    businessIDStr := os.Getenv("TASKCENTER_BUSINESS_ID")

    if apiURL == "" || apiKey == "" || businessIDStr == "" {
        return nil, fmt.Errorf("missing required environment variables")
    }

    businessID, err := strconv.ParseInt(businessIDStr, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("invalid business ID: %w", err)
    }

    // 创建客户端
    client, err := sdk.NewClientWithDefaults(apiURL, apiKey, businessID)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }

    return client, nil
}
```

## 基本使用

### 创建简单任务

```go
func createSimpleTask() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 创建任务
    task := sdk.NewTask(
        "order-12345",                           // 业务唯一ID
        "https://api.yourapp.com/webhook/task",  // 回调URL
    )

    ctx := context.Background()
    createdTask, err := client.Tasks().Create(ctx, task)
    if err != nil {
        log.Fatalf("Failed to create task: %v", err)
    }

    fmt.Printf("Task created successfully! ID: %d\n", createdTask.ID)
}
```

### 创建复杂任务

```go
func createAdvancedTask() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 创建带完整配置的任务
    task := sdk.NewTask("payment-67890", "https://api.yourapp.com/webhook/payment").
        WithPriority(sdk.TaskPriorityHigh).                    // 高优先级
        WithTimeout(300).                                      // 5分钟超时
        WithRetries(3, sdk.StandardRetryIntervals).           // 标准重试策略
        WithTags("payment", "critical", "user-premium").      // 添加标签
        WithHeaders(map[string]string{                        // 回调请求头
            "Authorization": "Bearer " + os.Getenv("WEBHOOK_TOKEN"),
            "Content-Type":  "application/json",
            "X-Source":      "taskcenter",
        }).
        WithBody(`{
            "payment_id": "67890",
            "amount": 99.99,
            "currency": "USD",
            "status": "completed"
        }`).                                                  // 回调请求体
        WithMetadata(map[string]interface{}{                  // 元数据
            "user_id":    12345,
            "order_type": "premium_subscription",
            "region":     "us-east-1",
        })

    ctx := context.Background()
    createdTask, err := client.Tasks().Create(ctx, task)
    if err != nil {
        log.Fatalf("Failed to create task: %v", err)
    }

    fmt.Printf("Advanced task created! ID: %d, Status: %s\n",
        createdTask.ID, createdTask.Status.String())
}
```

### 创建定时任务

```go
func createScheduledTask() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 创建1小时后执行的任务
    scheduledTime := time.Now().Add(1 * time.Hour)

    task := sdk.NewScheduledTask(
        "reminder-newsletter",
        "https://api.yourapp.com/webhook/newsletter",
        scheduledTime,
    ).
        WithTags("newsletter", "marketing").
        WithMetadata(map[string]interface{}{
            "campaign_id": "weekly-digest-2023",
            "send_time":   scheduledTime.Format(time.RFC3339),
        })

    ctx := context.Background()
    createdTask, err := client.Tasks().Create(ctx, task)
    if err != nil {
        log.Fatalf("Failed to create scheduled task: %v", err)
    }

    fmt.Printf("Scheduled task created! ID: %d, will execute at: %s\n",
        createdTask.ID, createdTask.ScheduledAt.Format(time.RFC3339))
}
```

## 任务管理

### 查询任务

```go
func queryTasks() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // 根据ID查询单个任务
    task, err := client.Tasks().Get(ctx, 123)
    if err != nil {
        if sdk.IsNotFoundError(err) {
            fmt.Println("Task not found")
            return
        }
        log.Fatalf("Failed to get task: %v", err)
    }

    fmt.Printf("Task ID: %d, Status: %s\n", task.ID, task.Status.String())

    // 根据业务ID查询
    task, err = client.Tasks().GetByBusinessUniqueID(ctx, "order-12345")
    if err != nil {
        log.Printf("Failed to get task by business ID: %v", err)
        return
    }

    fmt.Printf("Found task: %d for business ID: %s\n", task.ID, task.BusinessUniqueID)
}
```

### 查询任务列表

```go
func listTasks() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 构建查询条件
    req := sdk.NewListTasksRequest().
        WithStatus(sdk.TaskStatusPending, sdk.TaskStatusRunning).  // 活跃任务
        WithTagsFilter("payment").                                 // 包含payment标签
        WithPriorityFilter(sdk.TaskPriorityHigh).                 // 高优先级
        WithDateRange(                                            // 最近24小时创建
            time.Now().Add(-24*time.Hour),
            time.Now(),
        ).
        WithPagination(1, 20)                                     // 第一页，每页20条

    ctx := context.Background()
    response, err := client.Tasks().List(ctx, req)
    if err != nil {
        log.Fatalf("Failed to list tasks: %v", err)
    }

    fmt.Printf("Found %d tasks (total: %d)\n", len(response.Tasks), response.Total)
    for _, task := range response.Tasks {
        fmt.Printf("- Task %d: %s (%s) - %v\n",
            task.ID,
            task.BusinessUniqueID,
            task.Status.String(),
            task.Tags)
    }
}
```

### 批量创建任务

```go
func batchCreateTasks() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 创建多个任务
    tasks := []sdk.CreateTaskRequest{
        *sdk.NewTask("batch-order-1", "https://api.yourapp.com/webhook/order").
            WithTags("order", "batch"),
        *sdk.NewTask("batch-order-2", "https://api.yourapp.com/webhook/order").
            WithTags("order", "batch"),
        *sdk.NewTask("batch-order-3", "https://api.yourapp.com/webhook/order").
            WithTags("order", "batch"),
    }

    batchReq := &sdk.BatchCreateTasksRequest{
        Tasks: tasks,
    }

    ctx := context.Background()
    response, err := client.Tasks().BatchCreate(ctx, batchReq)
    if err != nil {
        log.Fatalf("Failed to batch create tasks: %v", err)
    }

    fmt.Printf("Batch creation completed:\n")
    fmt.Printf("- Succeeded: %d tasks\n", len(response.Succeeded))
    fmt.Printf("- Failed: %d tasks\n", len(response.Failed))

    for _, task := range response.Succeeded {
        fmt.Printf("  ✓ Task %d: %s\n", task.ID, task.BusinessUniqueID)
    }

    for _, failure := range response.Failed {
        fmt.Printf("  ✗ Index %d: %s\n", failure.Index, failure.Error)
    }
}
```

### 任务控制

```go
func taskControl() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    taskID := int64(123)

    // 取消任务
    err = client.Tasks().Cancel(ctx, taskID)
    if err != nil {
        if sdk.IsNotFoundError(err) {
            fmt.Println("Task not found")
        } else {
            log.Printf("Failed to cancel task: %v", err)
        }
        return
    }
    fmt.Printf("Task %d cancelled successfully\n", taskID)

    // 重试任务
    err = client.Tasks().Retry(ctx, taskID)
    if err != nil {
        log.Printf("Failed to retry task: %v", err)
        return
    }
    fmt.Printf("Task %d retried successfully\n", taskID)
}
```

## 回调处理

### 创建回调服务器

```go
func setupCallbackServer() {
    // 获取API密钥用于签名验证
    apiSecret := os.Getenv("TASKCENTER_API_SECRET")
    if apiSecret == "" {
        log.Fatal("TASKCENTER_API_SECRET environment variable is required")
    }

    // 创建回调处理器
    handler := &sdk.DefaultCallbackHandler{
        OnTaskCreated: func(event *sdk.CallbackEvent) error {
            fmt.Printf("📝 Task created: %d (%s)\n",
                event.TaskID, event.Task.BusinessUniqueID)
            // 这里可以添加业务逻辑
            return nil
        },
        OnTaskStarted: func(event *sdk.CallbackEvent) error {
            fmt.Printf("🚀 Task started: %d (%s)\n",
                event.TaskID, event.Task.BusinessUniqueID)
            // 这里可以添加业务逻辑
            return nil
        },
        OnTaskCompleted: func(event *sdk.CallbackEvent) error {
            fmt.Printf("✅ Task completed: %d (%s)\n",
                event.TaskID, event.Task.BusinessUniqueID)

            // 根据任务类型处理业务逻辑
            switch {
            case contains(event.Task.Tags, "payment"):
                return handlePaymentCompletion(event.Task)
            case contains(event.Task.Tags, "order"):
                return handleOrderCompletion(event.Task)
            default:
                return handleGenericCompletion(event.Task)
            }
        },
        OnTaskFailed: func(event *sdk.CallbackEvent) error {
            fmt.Printf("❌ Task failed: %d (%s) - Error: %s\n",
                event.TaskID, event.Task.BusinessUniqueID, event.Task.ErrorMessage)

            // 发送告警通知
            return sendFailureAlert(event.Task)
        },
    }

    // 创建带中间件的回调服务器
    server := sdk.NewCallbackServer(
        apiSecret,
        handler,
        sdk.WithCallbackMiddleware(&sdk.LoggingMiddleware{
            Logger: func(level, message string, fields map[string]interface{}) {
                log.Printf("[%s] %s: %+v", level, message, fields)
            },
        }),
    )

    // 启动HTTP服务器
    fmt.Println("Starting callback server on :8080")
    log.Fatal(http.ListenAndServe(":8080", server))
}

// 辅助函数
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func handlePaymentCompletion(task sdk.Task) error {
    fmt.Printf("Processing payment completion for task %d\n", task.ID)
    // 支付完成处理逻辑
    return nil
}

func handleOrderCompletion(task sdk.Task) error {
    fmt.Printf("Processing order completion for task %d\n", task.ID)
    // 订单完成处理逻辑
    return nil
}

func handleGenericCompletion(task sdk.Task) error {
    fmt.Printf("Processing generic completion for task %d\n", task.ID)
    // 通用完成处理逻辑
    return nil
}

func sendFailureAlert(task sdk.Task) error {
    fmt.Printf("Sending failure alert for task %d\n", task.ID)
    // 发送告警通知逻辑
    return nil
}
```

## 错误处理

### 全面的错误处理示例

```go
func handleErrors() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    task := sdk.NewTask("error-demo", "https://api.yourapp.com/webhook")

    ctx := context.Background()
    createdTask, err := client.Tasks().Create(ctx, task)
    if err != nil {
        // 根据错误类型进行处理
        switch {
        case sdk.IsValidationError(err):
            fmt.Printf("❌ Validation error: %s\n", err.Error())
            // 修复数据后重试

        case sdk.IsAuthenticationError(err):
            fmt.Printf("🔐 Authentication failed: %s\n", err.Error())
            // 检查API密钥配置

        case sdk.IsAuthorizationError(err):
            fmt.Printf("🚫 Access denied: %s\n", err.Error())
            // 检查权限配置

        case sdk.IsRateLimitError(err):
            fmt.Printf("⏱️ Rate limit exceeded: %s\n", err.Error())
            // 等待后重试
            time.Sleep(time.Minute)

        case sdk.IsConflictError(err):
            fmt.Printf("⚠️ Resource conflict: %s\n", err.Error())
            // 处理资源冲突

        case sdk.IsServerError(err):
            fmt.Printf("🔥 Server error: %s\n", err.Error())
            // 服务器错误，可以重试

        case sdk.IsNetworkError(err):
            fmt.Printf("🌐 Network error: %s\n", err.Error())
            // 网络错误，检查连接

        case sdk.IsTimeoutError(err):
            fmt.Printf("⏰ Request timeout: %s\n", err.Error())
            // 请求超时，可以重试

        default:
            fmt.Printf("❓ Unknown error: %s\n", err.Error())
        }
        return
    }

    fmt.Printf("✅ Task created successfully: %d\n", createdTask.ID)
}
```

### 重试机制示例

```go
func createTaskWithRetry() {
    client, err := createClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    task := sdk.NewTask("retry-demo", "https://api.yourapp.com/webhook")

    ctx := context.Background()
    maxAttempts := 3

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        fmt.Printf("Attempt %d/%d...\n", attempt, maxAttempts)

        createdTask, err := client.Tasks().Create(ctx, task)
        if err != nil {
            // 只有在可重试的错误时才重试
            if sdk.IsRetryableError(err) && attempt < maxAttempts {
                fmt.Printf("Retryable error: %s. Retrying in 5 seconds...\n", err.Error())
                time.Sleep(5 * time.Second)
                continue
            }

            log.Fatalf("Failed to create task after %d attempts: %v", attempt, err)
        }

        fmt.Printf("✅ Task created successfully: %d\n", createdTask.ID)
        break
    }
}
```

## 完整示例

将所有功能组合的完整示例：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "time"

    "github.com/your-org/task-center/sdk"
)

func main() {
    // 确保环境变量已设置
    if err := checkEnvironment(); err != nil {
        log.Fatal(err)
    }

    // 演示任务管理功能
    fmt.Println("🚀 Starting TaskCenter SDK Demo")

    // 1. 创建各种类型的任务
    createSimpleTask()
    createAdvancedTask()
    createScheduledTask()

    // 2. 演示批量操作
    batchCreateTasks()

    // 3. 查询和管理任务
    queryTasks()
    listTasks()

    // 4. 获取统计信息
    getTaskStats()

    // 5. 启动回调服务器（这会阻塞）
    fmt.Println("📡 Starting callback server...")
    go setupCallbackServer()

    // 保持程序运行
    select {}
}

func checkEnvironment() error {
    required := []string{
        "TASKCENTER_API_URL",
        "TASKCENTER_API_KEY",
        "TASKCENTER_BUSINESS_ID",
        "TASKCENTER_API_SECRET",
    }

    for _, env := range required {
        if os.Getenv(env) == "" {
            return fmt.Errorf("missing required environment variable: %s", env)
        }
    }

    return nil
}

func getTaskStats() {
    client, err := createClient()
    if err != nil {
        log.Printf("Failed to create client: %v", err)
        return
    }
    defer client.Close()

    ctx := context.Background()
    stats, err := client.Tasks().Stats(ctx)
    if err != nil {
        log.Printf("Failed to get stats: %v", err)
        return
    }

    fmt.Printf("📊 Task Statistics:\n")
    fmt.Printf("  Total tasks: %d\n", stats.TotalTasks)
    fmt.Printf("  Status distribution:\n")
    for status, count := range stats.StatusCounts {
        fmt.Printf("    %s: %d\n", status.String(), count)
    }
}
```

## 下一步

现在您已经了解了基本用法，建议您：

1. 阅读完整的 [API 文档](docs/API.md)
2. 查看更多 [示例代码](examples/)
3. 了解 [最佳实践](docs/BEST_PRACTICES.md)
4. 查看 [故障排除指南](docs/TROUBLESHOOTING.md)

## 获取帮助

如果您遇到问题或需要帮助：

1. 查看 [常见问题](docs/FAQ.md)
2. 提交 [GitHub Issue](https://github.com/your-org/task-center/issues)
3. 联系技术支持

Happy coding! 🎉