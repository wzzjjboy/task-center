# TaskCenter Go SDK API 文档

## 概述

TaskCenter Go SDK 是官方提供的 Go 语言客户端库，用于与 TaskCenter 任务调度系统进行集成。SDK 提供了完整的任务管理功能，包括任务的创建、查询、更新、删除以及回调处理等。

## 特性

- **完整的任务管理**：支持任务的全生命周期管理
- **认证管理**：基于 API Key 的安全认证
- **回调处理**：内置回调服务器和签名验证
- **错误处理**：完善的错误处理和重试机制
- **链式调用**：支持流畅的链式API调用
- **批量操作**：支持批量创建和管理任务
- **类型安全**：完全的类型安全保证

## 安装

```bash
go get github.com/your-org/task-center/sdk
```

## 快速开始

### 1. 创建客户端

```go
package main

import (
    "context"
    "log"

    "github.com/your-org/task-center/sdk"
)

func main() {
    // 使用默认配置创建客户端
    client, err := sdk.NewClientWithDefaults(
        "http://localhost:8080", // TaskCenter 服务地址
        "your-api-key",          // API Key
        123,                     // 业务系统 ID
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
}
```

### 2. 创建任务

```go
// 创建简单任务
task := sdk.NewTask("order-123", "https://api.example.com/webhook")

// 创建任务
createdTask, err := client.Tasks().Create(context.Background(), task)
if err != nil {
    log.Fatal(err)
}

log.Printf("Task created: %d", createdTask.ID)
```

## API 参考

### 客户端配置

#### Config

客户端配置结构：

```go
type Config struct {
    BaseURL     string        // TaskCenter 服务基础URL
    APIKey      string        // API 密钥
    BusinessID  int64         // 业务系统ID
    Timeout     time.Duration // 请求超时时间
    RetryPolicy *RetryPolicy  // 重试策略
    UserAgent   string        // 用户代理字符串
}
```

#### RetryPolicy

重试策略配置：

```go
type RetryPolicy struct {
    MaxRetries      int           // 最大重试次数
    InitialInterval time.Duration // 初始重试间隔
    MaxInterval     time.Duration // 最大重试间隔
    Multiplier      float64       // 重试间隔倍数
    RetryableErrors []int         // 可重试的HTTP状态码
}
```

#### 客户端创建方法

```go
// 使用默认配置创建客户端
func NewClientWithDefaults(baseURL, apiKey string, businessID int64) (*Client, error)

// 使用自定义配置创建客户端
func NewClient(config *Config) (*Client, error)

// 获取默认配置
func DefaultConfig() *Config
```

### 任务管理

#### TaskService 接口

```go
type TaskService interface {
    Create(ctx context.Context, req *CreateTaskRequest) (*Task, error)
    Get(ctx context.Context, taskID int64) (*Task, error)
    GetByBusinessUniqueID(ctx context.Context, businessUniqueID string) (*Task, error)
    Update(ctx context.Context, taskID int64, req *UpdateTaskRequest) (*Task, error)
    Delete(ctx context.Context, taskID int64) error
    List(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error)
    Stats(ctx context.Context) (*TaskStatsResponse, error)
    BatchCreate(ctx context.Context, req *BatchCreateTasksRequest) (*BatchCreateTasksResponse, error)
    Cancel(ctx context.Context, taskID int64) error
    Retry(ctx context.Context, taskID int64) error
}
```

#### 任务数据结构

##### Task

```go
type Task struct {
    ID               int64             `json:"id,omitempty"`
    BusinessUniqueID string            `json:"business_unique_id"`
    CallbackURL      string            `json:"callback_url"`
    CallbackMethod   string            `json:"callback_method,omitempty"`
    CallbackHeaders  map[string]string `json:"callback_headers,omitempty"`
    CallbackBody     string            `json:"callback_body,omitempty"`
    RetryIntervals   []int             `json:"retry_intervals,omitempty"`
    MaxRetries       int               `json:"max_retries,omitempty"`
    CurrentRetry     int               `json:"current_retry,omitempty"`
    Status           TaskStatus        `json:"status,omitempty"`
    Priority         TaskPriority      `json:"priority,omitempty"`
    Tags             []string          `json:"tags,omitempty"`
    Timeout          int               `json:"timeout,omitempty"`
    ScheduledAt      time.Time         `json:"scheduled_at,omitempty"`
    NextExecuteAt    *time.Time        `json:"next_execute_at,omitempty"`
    ExecutedAt       *time.Time        `json:"executed_at,omitempty"`
    CompletedAt      *time.Time        `json:"completed_at,omitempty"`
    ErrorMessage     string            `json:"error_message,omitempty"`
    Metadata         map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt        time.Time         `json:"created_at,omitempty"`
    UpdatedAt        time.Time         `json:"updated_at,omitempty"`
}
```

##### TaskStatus

任务状态枚举：

```go
type TaskStatus int

const (
    TaskStatusPending    TaskStatus = 0 // 待执行
    TaskStatusRunning    TaskStatus = 1 // 执行中
    TaskStatusSucceeded  TaskStatus = 2 // 成功
    TaskStatusFailed     TaskStatus = 3 // 失败
    TaskStatusCancelled  TaskStatus = 4 // 取消
    TaskStatusExpired    TaskStatus = 5 // 过期
)
```

##### TaskPriority

任务优先级：

```go
type TaskPriority int

const (
    TaskPriorityHighest TaskPriority = 1
    TaskPriorityHigh    TaskPriority = 3
    TaskPriorityNormal  TaskPriority = 5
    TaskPriorityLow     TaskPriority = 7
    TaskPriorityLowest  TaskPriority = 9
)
```

#### 创建任务

##### CreateTaskRequest

```go
type CreateTaskRequest struct {
    BusinessUniqueID string                 `json:"business_unique_id"`
    CallbackURL      string                 `json:"callback_url"`
    CallbackMethod   string                 `json:"callback_method,omitempty"`
    CallbackHeaders  map[string]string      `json:"callback_headers,omitempty"`
    CallbackBody     string                 `json:"callback_body,omitempty"`
    RetryIntervals   []int                  `json:"retry_intervals,omitempty"`
    MaxRetries       int                    `json:"max_retries,omitempty"`
    Priority         TaskPriority           `json:"priority,omitempty"`
    Tags             []string               `json:"tags,omitempty"`
    Timeout          int                    `json:"timeout,omitempty"`
    ScheduledAt      *time.Time             `json:"scheduled_at,omitempty"`
    Metadata         map[string]interface{} `json:"metadata,omitempty"`
}
```

##### 便捷构造函数

```go
// 创建基本任务
func NewTask(businessUniqueID, callbackURL string) *CreateTaskRequest

// 创建带回调配置的任务
func NewTaskWithCallback(businessUniqueID, callbackURL, method string, headers map[string]string, body string) *CreateTaskRequest

// 创建定时任务
func NewScheduledTask(businessUniqueID, callbackURL string, scheduledAt time.Time) *CreateTaskRequest
```

##### 链式调用方法

```go
// 设置优先级
func (req *CreateTaskRequest) WithPriority(priority TaskPriority) *CreateTaskRequest

// 设置超时时间
func (req *CreateTaskRequest) WithTimeout(timeout int) *CreateTaskRequest

// 设置重试配置
func (req *CreateTaskRequest) WithRetries(maxRetries int, intervals []int) *CreateTaskRequest

// 设置标签
func (req *CreateTaskRequest) WithTags(tags ...string) *CreateTaskRequest

// 设置元数据
func (req *CreateTaskRequest) WithMetadata(metadata map[string]interface{}) *CreateTaskRequest

// 设置请求头
func (req *CreateTaskRequest) WithHeaders(headers map[string]string) *CreateTaskRequest

// 设置请求体
func (req *CreateTaskRequest) WithBody(body string) *CreateTaskRequest

// 设置计划执行时间
func (req *CreateTaskRequest) WithSchedule(scheduledAt time.Time) *CreateTaskRequest
```

##### 示例

```go
// 创建高优先级任务，带重试机制
task := sdk.NewTask("payment-456", "https://api.example.com/payment/callback").
    WithPriority(sdk.TaskPriorityHigh).
    WithTimeout(60).
    WithRetries(3, sdk.StandardRetryIntervals).
    WithTags("payment", "important").
    WithHeaders(map[string]string{
        "Authorization": "Bearer token",
        "Content-Type":  "application/json",
    }).
    WithBody(`{"payment_id": "456", "status": "completed"}`)

createdTask, err := client.Tasks().Create(ctx, task)
```

#### 查询任务

##### 单个任务查询

```go
// 根据任务ID查询
task, err := client.Tasks().Get(ctx, taskID)

// 根据业务唯一ID查询
task, err := client.Tasks().GetByBusinessUniqueID(ctx, "order-123")
```

##### 任务列表查询

```go
// 基本查询
req := sdk.NewListTasksRequest().
    WithStatus(sdk.TaskStatusPending, sdk.TaskStatusRunning).
    WithTagsFilter("payment").
    WithPriorityFilter(sdk.TaskPriorityHigh).
    WithPagination(1, 20)

response, err := client.Tasks().List(ctx, req)
```

##### ListTasksRequest

```go
type ListTasksRequest struct {
    Status      []TaskStatus  `json:"status,omitempty"`
    Tags        []string      `json:"tags,omitempty"`
    Priority    *TaskPriority `json:"priority,omitempty"`
    CreatedFrom *time.Time    `json:"created_from,omitempty"`
    CreatedTo   *time.Time    `json:"created_to,omitempty"`
    Page        int           `json:"page,omitempty"`
    PageSize    int           `json:"page_size,omitempty"`
}
```

##### ListTasksResponse

```go
type ListTasksResponse struct {
    Tasks      []Task `json:"tasks"`
    Total      int    `json:"total"`
    Page       int    `json:"page"`
    PageSize   int    `json:"page_size"`
    TotalPages int    `json:"total_pages"`
}
```

#### 更新任务

```go
updateReq := &sdk.UpdateTaskRequest{
    Priority: &sdk.TaskPriorityHigh,
    Timeout:  &newTimeout,
}

updatedTask, err := client.Tasks().Update(ctx, taskID, updateReq)
```

#### 批量操作

##### 批量创建

```go
batchReq := &sdk.BatchCreateTasksRequest{
    Tasks: []sdk.CreateTaskRequest{
        *sdk.NewTask("batch-1", "https://api.example.com/webhook1"),
        *sdk.NewTask("batch-2", "https://api.example.com/webhook2"),
    },
}

response, err := client.Tasks().BatchCreate(ctx, batchReq)
```

##### BatchCreateTasksResponse

```go
type BatchCreateTasksResponse struct {
    Succeeded []Task           `json:"succeeded"`
    Failed    []BatchTaskError `json:"failed"`
}
```

#### 任务控制

```go
// 取消任务
err := client.Tasks().Cancel(ctx, taskID)

// 重试任务
err := client.Tasks().Retry(ctx, taskID)

// 删除任务
err := client.Tasks().Delete(ctx, taskID)
```

#### 统计信息

```go
stats, err := client.Tasks().Stats(ctx)

type TaskStatsResponse struct {
    TotalTasks     int                    `json:"total_tasks"`
    StatusCounts   map[TaskStatus]int     `json:"status_counts"`
    PriorityCounts map[TaskPriority]int   `json:"priority_counts"`
    TagCounts      map[string]int         `json:"tag_counts"`
}
```

### 回调处理

#### CallbackServer

用于接收和处理 TaskCenter 的回调通知：

```go
// 创建回调处理器
handler := &sdk.DefaultCallbackHandler{
    OnTaskCreated: func(event *sdk.CallbackEvent) error {
        log.Printf("Task created: %d", event.TaskID)
        return nil
    },
    OnTaskCompleted: func(event *sdk.CallbackEvent) error {
        log.Printf("Task completed: %d", event.TaskID)
        return nil
    },
    OnTaskFailed: func(event *sdk.CallbackEvent) error {
        log.Printf("Task failed: %d, error: %s", event.TaskID, event.Task.ErrorMessage)
        return nil
    },
}

// 创建回调服务器
server := sdk.NewCallbackServer("your-api-secret", handler)

// 启动服务器
log.Fatal(http.ListenAndServe(":8080", server))
```

#### CallbackEvent

```go
type CallbackEvent struct {
    EventType   string    `json:"event_type"`   // task.created, task.started, task.completed, task.failed
    EventTime   time.Time `json:"event_time"`
    TaskID      int64     `json:"task_id"`
    BusinessID  int64     `json:"business_id"`
    Task        Task      `json:"task"`
    Signature   string    `json:"signature"`    // 回调签名，用于验证
}
```

#### 中间件支持

```go
// 日志中间件
loggingMiddleware := &sdk.LoggingMiddleware{
    Logger: func(level, message string, fields map[string]interface{}) {
        log.Printf("[%s] %s: %+v", level, message, fields)
    },
}

// 指标中间件
metricsMiddleware := &sdk.MetricsMiddleware{
    IncCounter: func(name string, labels map[string]string) {
        // 更新指标
    },
}

// 创建带中间件的服务器
server := sdk.NewCallbackServer(
    "your-api-secret",
    handler,
    sdk.WithCallbackMiddleware(loggingMiddleware),
    sdk.WithCallbackMiddleware(metricsMiddleware),
)
```

### 错误处理

#### 错误类型

```go
type Error interface {
    error
    Code() string
    StatusCode() int
    Details() interface{}
}
```

#### 错误代码

```go
const (
    CodeValidationError     = "VALIDATION_ERROR"
    CodeAuthenticationError = "AUTHENTICATION_ERROR"
    CodeAuthorizationError  = "AUTHORIZATION_ERROR"
    CodeNotFoundError       = "NOT_FOUND_ERROR"
    CodeConflictError       = "CONFLICT_ERROR"
    CodeRateLimitError      = "RATE_LIMIT_ERROR"
    CodeServerError         = "SERVER_ERROR"
    CodeNetworkError        = "NETWORK_ERROR"
    CodeTimeoutError        = "TIMEOUT_ERROR"
    CodeUnknownError        = "UNKNOWN_ERROR"
)
```

#### 错误检查函数

```go
// 检查错误类型
if sdk.IsValidationError(err) {
    // 处理验证错误
}

if sdk.IsAuthenticationError(err) {
    // 处理认证错误
}

if sdk.IsRetryableError(err) {
    // 可以重试的错误
}
```

#### 错误处理示例

```go
task, err := client.Tasks().Create(ctx, req)
if err != nil {
    switch {
    case sdk.IsValidationError(err):
        log.Printf("Validation error: %s", err.Error())
        // 修复请求数据
    case sdk.IsAuthenticationError(err):
        log.Printf("Authentication failed: %s", err.Error())
        // 检查API密钥
    case sdk.IsRateLimitError(err):
        log.Printf("Rate limit exceeded: %s", err.Error())
        // 等待后重试
        time.Sleep(time.Minute)
    default:
        log.Printf("Unexpected error: %s", err.Error())
    }
    return
}
```

### 工具函数

#### 任务状态检查

```go
// 检查任务是否处于活跃状态
if sdk.IsTaskActive(task.Status) {
    // 任务正在运行或等待执行
}

// 检查任务是否已完成
if sdk.IsTaskCompleted(task.Status) {
    // 任务已完成（成功、失败、取消或过期）
}

// 检查任务是否成功
if sdk.IsTaskSuccessful(task.Status) {
    // 任务成功完成
}
```

#### 重试计算

```go
// 计算重试延迟
delay := sdk.CalculateRetryDelay(task.CurrentRetry, task.RetryIntervals)
```

#### 预定义配置

```go
// 预定义的重试间隔
sdk.StandardRetryIntervals  // [60, 300, 900] (1分钟, 5分钟, 15分钟)
sdk.FastRetryIntervals      // [10, 30, 60] (10秒, 30秒, 60秒)
sdk.SlowRetryIntervals      // [300, 1800, 7200] (5分钟, 30分钟, 2小时)
sdk.ExponentialRetryIntervals // [60, 120, 240, 480, 960] (指数增长)
```

## 最佳实践

### 1. 客户端配置

```go
// 推荐的生产环境配置
config := sdk.DefaultConfig()
config.BaseURL = "https://taskcenter.example.com"
config.APIKey = os.Getenv("TASKCENTER_API_KEY")
config.BusinessID = 123
config.Timeout = 30 * time.Second
config.RetryPolicy.MaxRetries = 3

client, err := sdk.NewClient(config)
```

### 2. 错误处理

```go
// 始终检查和处理错误
task, err := client.Tasks().Create(ctx, req)
if err != nil {
    // 根据错误类型采取不同的处理策略
    if sdk.IsRetryableError(err) {
        // 记录日志并重试
        return handleRetryableError(err)
    }
    // 记录错误并返回
    return fmt.Errorf("failed to create task: %w", err)
}
```

### 3. 资源管理

```go
// 确保正确关闭客户端
client, err := sdk.NewClient(config)
if err != nil {
    return err
}
defer client.Close()
```

### 4. 回调安全

```go
// 使用强密钥进行签名验证
apiSecret := os.Getenv("TASKCENTER_API_SECRET")
if len(apiSecret) < 32 {
    log.Fatal("API secret must be at least 32 characters")
}

server := sdk.NewCallbackServer(apiSecret, handler)
```

### 5. 监控和日志

```go
// 添加日志和监控中间件
server := sdk.NewCallbackServer(
    apiSecret,
    handler,
    sdk.WithCallbackMiddleware(&sdk.LoggingMiddleware{
        Logger: yourLogger,
    }),
    sdk.WithCallbackMiddleware(&sdk.MetricsMiddleware{
        IncCounter: yourMetrics.Counter,
    }),
)
```

## 版本信息

- **当前版本**: 1.0.0
- **Go 版本要求**: Go 1.18+
- **兼容性**: TaskCenter API v1

## 更新日志

### v1.0.0
- 初始发布
- 完整的任务管理功能
- 回调处理支持
- 错误处理和重试机制
- 批量操作支持

## 许可证

MIT License