# TaskCenter Go SDK 测试指南

本文档描述了如何运行和编写 TaskCenter Go SDK 的测试。

## 测试类型

我们有三种类型的测试：

### 1. 单元测试
单元测试验证单个组件的功能，不依赖外部服务。

**运行单元测试**:
```bash
cd sdk
go test -v
```

**运行特定测试**:
```bash
go test -v -run TestClientCreation
```

**生成测试覆盖率报告**:
```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### 2. 集成测试
集成测试需要真实的 TaskCenter 服务实例。

**设置环境变量**:
```bash
export TASKCENTER_API_URL="http://localhost:8080"
export TASKCENTER_API_KEY="your-api-key"
export TASKCENTER_BUSINESS_ID="123"
```

**运行集成测试**:
```bash
go test -tags=integration -v
```

### 3. 示例测试
运行示例代码作为测试：

```bash
# 基础示例
cd examples/basic
go run main.go

# 高级示例
cd examples/advanced
go run main.go

# 回调服务器
cd examples/callback_server
export TASKCENTER_API_SECRET="your-secret"
go run main.go
```

## 测试文件结构

```
sdk/
├── client_test.go          # 客户端测试
├── types_test.go           # 数据类型测试
├── errors_test.go          # 错误处理测试
├── callback_test.go        # 回调处理测试
└── integration_test.go     # 集成测试
```

## 编写测试

### 单元测试示例

```go
func TestNewClient(t *testing.T) {
    config := &Config{
        BaseURL:    "http://example.com",
        APIKey:     "test-key",
        BusinessID: 123,
    }

    client, err := NewClient(config)
    if err != nil {
        t.Fatalf("NewClient() failed: %v", err)
    }
    defer client.Close()

    if client == nil {
        t.Error("NewClient() returned nil")
    }
}
```

### 使用 httptest 测试 HTTP 交互

```go
func TestClientRequest(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"success": true}`))
    }))
    defer server.Close()

    // 创建客户端
    client, err := NewClientWithDefaults(server.URL, "test-key", 123)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // 测试请求
    resp, err := client.doRequest(context.Background(), "GET", "/test", nil)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}
```

### 测试错误处理

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name       string
        statusCode int
        body       string
        wantError  string
    }{
        {
            name:       "validation error",
            statusCode: 400,
            body:       `{"code": "VALIDATION_ERROR", "message": "invalid request"}`,
            wantError:  "invalid request",
        },
        {
            name:       "server error",
            statusCode: 500,
            body:       "",
            wantError:  "internal server error",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ParseHTTPError(tt.statusCode, []byte(tt.body))
            if err.Error() != tt.wantError {
                t.Errorf("Expected error %s, got %s", tt.wantError, err.Error())
            }
        })
    }
}
```

### 回调处理测试

```go
func TestCallbackServer(t *testing.T) {
    handler := &DefaultCallbackHandler{
        OnTaskCompleted: func(event *CallbackEvent) error {
            // 验证事件数据
            if event.TaskID == 0 {
                t.Error("TaskID should not be zero")
            }
            return nil
        },
    }

    server := NewCallbackServer("test-secret", handler)

    // 创建测试事件
    event := &CallbackEvent{
        EventType: "task.completed",
        TaskID:    123,
    }

    eventData, _ := json.Marshal(event)
    timestamp := strconv.FormatInt(time.Now().Unix(), 10)
    signature := calculateSignature("test-secret", timestamp, eventData)

    req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
    req.Header.Set("X-TaskCenter-Signature", signature)
    req.Header.Set("X-TaskCenter-Timestamp", timestamp)

    w := httptest.NewRecorder()
    server.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}
```

## 测试最佳实践

### 1. 测试组织

```go
func TestFeatureName(t *testing.T) {
    // 使用子测试组织相关测试
    t.Run("valid input", func(t *testing.T) {
        // 测试正常情况
    })

    t.Run("invalid input", func(t *testing.T) {
        // 测试异常情况
    })

    t.Run("edge cases", func(t *testing.T) {
        // 测试边界情况
    })
}
```

### 2. 表格驱动测试

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid-id", false},
        {"empty input", "", true},
        {"invalid format", "invalid id", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 3. 测试辅助函数

```go
// 测试辅助函数
func createTestClient(t *testing.T) *Client {
    client, err := NewClientWithDefaults("http://test.com", "test-key", 123)
    if err != nil {
        t.Fatalf("Failed to create test client: %v", err)
    }
    return client
}

func TestWithHelper(t *testing.T) {
    client := createTestClient(t)
    defer client.Close()

    // 使用客户端进行测试...
}
```

### 4. 清理资源

```go
func TestResourceCleanup(t *testing.T) {
    client := createTestClient(t)
    defer client.Close() // 确保清理

    server := httptest.NewServer(handler)
    defer server.Close() // 确保关闭测试服务器

    // 测试逻辑...
}
```

### 5. 并发测试

```go
func TestConcurrentAccess(t *testing.T) {
    client := createTestClient(t)
    defer client.Close()

    const goroutines = 10
    errors := make(chan error, goroutines)

    for i := 0; i < goroutines; i++ {
        go func(id int) {
            task := NewTask(fmt.Sprintf("concurrent-%d", id), "https://example.com")
            _, err := client.Tasks().Create(context.Background(), task)
            errors <- err
        }(i)
    }

    for i := 0; i < goroutines; i++ {
        if err := <-errors; err != nil {
            t.Errorf("Goroutine %d failed: %v", i, err)
        }
    }
}
```

## 模拟和存根

### HTTP 模拟

```go
func TestWithMockServer(t *testing.T) {
    // 创建模拟服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/api/v1/tasks":
            w.WriteHeader(http.StatusCreated)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "success": true,
                "data": map[string]interface{}{
                    "id": 123,
                    "business_unique_id": "test-task",
                    "status": 0,
                },
            })
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
    defer server.Close()

    client, _ := NewClientWithDefaults(server.URL, "test-key", 123)
    defer client.Close()

    // 测试任务创建
    task := NewTask("test-task", "https://example.com")
    createdTask, err := client.Tasks().Create(context.Background(), task)
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }

    if createdTask.ID != 123 {
        t.Errorf("Expected ID 123, got %d", createdTask.ID)
    }
}
```

## 持续集成

### GitHub Actions 配置

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2

    - name: Set up Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.19

    - name: Run unit tests
      run: |
        cd sdk
        go test -v -coverprofile=coverage.out

    - name: Upload coverage
      uses: codecov/codecov-action@v1
      with:
        file: ./sdk/coverage.out

    - name: Run integration tests
      env:
        TASKCENTER_API_URL: http://localhost:8080
        TASKCENTER_API_KEY: test-key
        TASKCENTER_BUSINESS_ID: 1
      run: |
        # 启动测试服务器
        docker-compose up -d taskcenter

        # 等待服务启动
        sleep 30

        # 运行集成测试
        cd sdk
        go test -tags=integration -v
```

## 基准测试

```go
func BenchmarkTaskCreation(b *testing.B) {
    client := createTestClient(b)
    defer client.Close()

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        task := NewTask(fmt.Sprintf("bench-%d", i), "https://example.com")
        _, err := client.Tasks().Create(context.Background(), task)
        if err != nil {
            b.Fatalf("Create failed: %v", err)
        }
    }
}

// 运行基准测试
// go test -bench=.
// go test -bench=BenchmarkTaskCreation
```

## 测试命令汇总

```bash
# 运行所有单元测试
go test -v

# 运行特定测试
go test -v -run TestClientCreation

# 运行集成测试
go test -tags=integration -v

# 生成覆盖率报告
go test -coverprofile=coverage.out
go tool cover -html=coverage.out

# 运行基准测试
go test -bench=.

# 运行竞态检测
go test -race

# 详细输出
go test -v -count=1

# 运行示例
cd examples/basic && go run main.go
cd examples/advanced && go run main.go
cd examples/callback_server && go run main.go
cd examples/complete_workflow && go run main.go
```

## 故障排除

### 常见测试问题

1. **时间相关测试不稳定**
   ```go
   // 不要依赖精确的时间比较
   // Bad
   if time.Since(start) != 5*time.Second { ... }

   // Good
   duration := time.Since(start)
   if duration < 4*time.Second || duration > 6*time.Second {
       t.Errorf("Unexpected duration: %v", duration)
   }
   ```

2. **网络超时**
   ```go
   // 在测试中使用较短的超时
   config := DefaultConfig()
   config.Timeout = 5 * time.Second
   ```

3. **并发问题**
   ```go
   // 使用 -race 标志检测竞态条件
   // go test -race
   ```

4. **资源泄漏**
   ```go
   // 确保所有资源都被正确清理
   defer client.Close()
   defer server.Close()
   defer resp.Body.Close()
   ```

## 测试覆盖率目标

我们的目标是达到以下测试覆盖率：

- **单元测试**: > 85%
- **集成测试**: > 70%
- **关键路径**: 100%

通过运行以下命令检查覆盖率：

```bash
go test -coverprofile=coverage.out
go tool cover -func=coverage.out
```

好的测试是代码质量的保障，请在添加新功能时同步编写相应的测试！