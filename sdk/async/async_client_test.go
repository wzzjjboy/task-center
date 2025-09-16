package async

import (
	"context"
	"sync"
	"testing"
	"time"

	"task-center/sdk"
	"task-center/sdk/task"
)

// MockAsyncTaskClient 用于测试的模拟任务客户端
type MockAsyncTaskClient struct {
	createTaskFunc func(ctx context.Context, req *task.CreateRequest) (*task.Task, error)
	delay          time.Duration
	mu             sync.RWMutex
}

func (m *MockAsyncTaskClient) CreateTask(ctx context.Context, req *task.CreateRequest) (*task.Task, error) {
	// 模拟延迟
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.createTaskFunc != nil {
		return m.createTaskFunc(ctx, req)
	}

	// 返回默认的模拟任务
	return &task.Task{
		ID:        1,
		Name:      req.Name,
		Type:      req.Type,
		Status:    task.StatusPending,
		Priority:  req.Priority,
		Payload:   req.Payload,
		Tags:      req.Tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockAsyncTaskClient) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

func (m *MockAsyncTaskClient) SetCreateTaskFunc(fn func(ctx context.Context, req *task.CreateRequest) (*task.Task, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createTaskFunc = fn
}

func createMockAsyncClient() *task.Client {
	config := &sdk.Config{
		BaseURL:    "http://localhost:8080",
		APIKey:     "test-api-key",
		BusinessID: 1,
		Timeout:    30 * time.Second,
	}

	sdkClient, _ := sdk.NewClient(config)
	return task.NewClient(sdkClient)
}

func TestAsyncClient_Basic(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)

	if asyncClient.IsStarted() {
		t.Error("Expected client to not be started initially")
	}

	asyncClient.Start()
	if !asyncClient.IsStarted() {
		t.Error("Expected client to be started after Start()")
	}

	asyncClient.Stop()
	if asyncClient.IsStarted() {
		t.Error("Expected client to be stopped after Stop()")
	}
}

func TestAsyncClient_CreateTaskAsync(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	request := task.NewCreateRequest().
		WithName("async-test").
		WithType("test")

	var resultReceived bool
	var receivedResult *TaskResult
	var wg sync.WaitGroup
	wg.Add(1)

	callback := func(result *TaskResult) {
		resultReceived = true
		receivedResult = result
		wg.Done()
	}

	taskID, err := asyncClient.CreateTaskAsync(request, callback)
	if err != nil {
		t.Fatalf("Failed to create async task: %v", err)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}

	// 等待回调执行
	wg.Wait()

	if !resultReceived {
		t.Error("Expected callback to be called")
	}

	if receivedResult == nil {
		t.Fatal("Expected result to be received")
	}

	if receivedResult.Error != nil {
		t.Errorf("Expected no error, got: %v", receivedResult.Error)
	}

	if receivedResult.Task == nil {
		t.Error("Expected task to be created")
	}

	if receivedResult.Task.Name != "async-test" {
		t.Errorf("Expected task name 'async-test', got '%s'", receivedResult.Task.Name)
	}
}

func TestAsyncClient_CreateTaskFuture(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	request := task.NewCreateRequest().
		WithName("future-test").
		WithType("test")

	future := asyncClient.CreateTaskFuture(request)
	if future == nil {
		t.Fatal("Expected future to be created")
	}

	if future.IsDone() {
		t.Error("Expected future to not be done initially")
	}

	result, err := future.GetWithTimeout(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to get future result: %v", err)
	}

	if !future.IsDone() {
		t.Error("Expected future to be done after getting result")
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	if result.Task == nil {
		t.Error("Expected task to be created")
	}

	if result.Task.Name != "future-test" {
		t.Errorf("Expected task name 'future-test', got '%s'", result.Task.Name)
	}
}

func TestAsyncClient_Future_Timeout(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	// 设置模拟客户端延迟
	mockAsyncClient := &MockAsyncTaskClient{delay: 2 * time.Second}
	asyncClient.client = mockAsyncClient

	request := task.NewCreateRequest().
		WithName("timeout-test").
		WithType("test")

	future := asyncClient.CreateTaskFuture(request)

	// 使用较短的超时时间
	_, err := future.GetWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !IsTimeout(err) {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestTaskGroup_Basic(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	group := NewTaskGroup(asyncClient)

	// 添加多个任务
	for i := 0; i < 5; i++ {
		request := task.NewCreateRequest().
			WithName("group-test").
			WithType("test")
		group.Add(request)
	}

	if group.Size() != 5 {
		t.Errorf("Expected group size 5, got %d", group.Size())
	}

	// 等待所有任务完成
	results := group.Wait()

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	for i, result := range results {
		if result.Error != nil {
			t.Errorf("Task %d failed: %v", i, result.Error)
		}
		if result.Task == nil {
			t.Errorf("Task %d is nil", i)
		}
	}
}

func TestTaskGroup_Timeout(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	// 设置模拟客户端延迟
	mockAsyncClient := &MockAsyncTaskClient{delay: 2 * time.Second}
	asyncClient.client = mockAsyncClient

	group := NewTaskGroup(asyncClient)

	request := task.NewCreateRequest().
		WithName("timeout-group-test").
		WithType("test")
	group.Add(request)

	// 使用较短的超时时间
	_, err := group.WaitWithTimeout(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !IsTimeout(err) {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestPipeline_Basic(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	pipeline := NewPipeline(asyncClient)

	// 添加处理阶段
	pipeline.AddStage(func(req *task.CreateRequest) *task.CreateRequest {
		req.WithTag("pipeline")
		return req
	})

	pipeline.AddStage(func(req *task.CreateRequest) *task.CreateRequest {
		req.WithPriority(task.PriorityHigh)
		return req
	})

	request := task.NewCreateRequest().
		WithName("pipeline-test").
		WithType("test")

	future := pipeline.Process(request)
	result, err := future.GetWithTimeout(5 * time.Second)

	if err != nil {
		t.Fatalf("Pipeline processing failed: %v", err)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	if result.Task == nil {
		t.Fatal("Expected task to be created")
	}

	// 验证管道处理结果
	if len(result.Task.Tags) == 0 || result.Task.Tags[0] != "pipeline" {
		t.Error("Expected 'pipeline' tag to be added")
	}

	if result.Task.Priority != task.PriorityHigh {
		t.Errorf("Expected high priority, got %v", result.Task.Priority)
	}
}

func TestPipeline_NilStage(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	pipeline := NewPipeline(asyncClient)

	// 添加一个返回nil的阶段
	pipeline.AddStage(func(req *task.CreateRequest) *task.CreateRequest {
		return nil
	})

	request := task.NewCreateRequest().
		WithName("nil-stage-test").
		WithType("test")

	future := pipeline.Process(request)
	result, err := future.GetWithTimeout(5 * time.Second)

	if err != nil {
		t.Fatalf("Failed to get future result: %v", err)
	}

	if result.Error == nil {
		t.Error("Expected pipeline stage error")
	}

	if result.Error != ErrPipelineStageError {
		t.Errorf("Expected pipeline stage error, got: %v", result.Error)
	}
}

func TestWorkerPool_Basic(t *testing.T) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	pool := NewWorkerPool(asyncClient, 3)

	var results []bool
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 提交5个任务到容量为3的工作池
	for i := 0; i < 5; i++ {
		request := task.NewCreateRequest().
			WithName("pool-test").
			WithType("test")

		wg.Add(1)
		err := pool.Submit(request, func(result *TaskResult) {
			mu.Lock()
			results = append(results, result.Error == nil)
			mu.Unlock()
			wg.Done()
		})

		if i < 3 {
			// 前3个任务应该成功提交
			if err != nil {
				t.Errorf("Task %d should be submitted successfully: %v", i, err)
			}
		} else {
			// 后面的任务应该因为工作池满而失败
			if err == nil {
				t.Errorf("Task %d should fail due to full worker pool", i)
			}
			if !IsQueueFull(err) {
				t.Errorf("Expected queue full error, got: %v", err)
			}
			wg.Done() // 手动调用Done，因为回调不会被执行
		}
	}

	wg.Wait()
	pool.Close()

	// 应该有3个成功的结果
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for i, success := range results {
		if !success {
			t.Errorf("Task %d failed", i)
		}
	}
}

func TestAsyncClient_QueueFull(t *testing.T) {
	mockClient := createMockAsyncClient()
	config := &AsyncClientConfig{
		Workers:    1,
		BufferSize: 2, // 小缓冲区
		Timeout:    30 * time.Second,
	}
	asyncClient := NewAsyncClient(mockClient, config)

	// 不启动客户端，这样任务会堆积在队列中
	request := task.NewCreateRequest().
		WithName("queue-full-test").
		WithType("test")

	// 填满队列
	for i := 0; i < 2; i++ {
		_, err := asyncClient.CreateTaskAsync(request, nil)
		if err != nil {
			t.Errorf("Task %d should be queued successfully: %v", i, err)
		}
	}

	// 下一个任务应该失败
	_, err := asyncClient.CreateTaskAsync(request, nil)
	if err == nil {
		t.Error("Expected queue full error")
	}

	if !IsQueueFull(err) {
		t.Errorf("Expected queue full error, got: %v", err)
	}
}

func TestGenerateTaskID(t *testing.T) {
	id1 := generateTaskID()
	id2 := generateTaskID()

	if id1 == "" {
		t.Error("Expected non-empty task ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty task ID")
	}

	if id1 == id2 {
		t.Error("Expected different task IDs")
	}

	// 检查ID格式
	if len(id1) < 10 {
		t.Error("Expected task ID to be at least 10 characters long")
	}

	if id1[:6] != "async_" {
		t.Error("Expected task ID to start with 'async_'")
	}
}

func BenchmarkAsyncClient_CreateTaskAsync(b *testing.B) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	request := task.NewCreateRequest().
		WithName("benchmark-test").
		WithType("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asyncClient.CreateTaskAsync(request, nil)
	}
}

func BenchmarkAsyncClient_CreateTaskFuture(b *testing.B) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	asyncClient.Start()
	defer asyncClient.Stop()

	request := task.NewCreateRequest().
		WithName("benchmark-future-test").
		WithType("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asyncClient.CreateTaskFuture(request)
	}
}

func BenchmarkTaskGroup_Add(b *testing.B) {
	mockClient := createMockAsyncClient()
	asyncClient := NewAsyncClient(mockClient, nil)
	group := NewTaskGroup(asyncClient)

	request := task.NewCreateRequest().
		WithName("benchmark-group-test").
		WithType("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		group.Add(request)
	}
}