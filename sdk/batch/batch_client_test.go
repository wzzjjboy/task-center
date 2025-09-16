package batch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"task-center/sdk"
	"task-center/sdk/task"
)

// MockBatchTaskClient 用于测试的模拟任务客户端
type MockBatchTaskClient struct {
	createTaskFunc func(ctx context.Context, req *task.CreateRequest) (*task.Task, error)
	updateTaskFunc func(ctx context.Context, taskID int64, req *task.UpdateRequest) (*task.Task, error)
	deleteTaskFunc func(ctx context.Context, taskID int64) error
	getTaskFunc    func(ctx context.Context, taskID int64) (*task.Task, error)
	cancelTaskFunc func(ctx context.Context, taskID int64) (*task.Task, error)
	retryTaskFunc  func(ctx context.Context, taskID int64) (*task.Task, error)
	delay          time.Duration
	failureRate    float64 // 失败率，0.0-1.0
	callCount      int
	mu             sync.RWMutex
}

func (m *MockBatchTaskClient) CreateTask(ctx context.Context, req *task.CreateRequest) (*task.Task, error) {
	m.mu.Lock()
	m.callCount++
	callCount := m.callCount
	m.mu.Unlock()

	// 模拟延迟
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// 模拟失败
	if m.failureRate > 0 && float64(callCount%100)/100.0 < m.failureRate {
		return nil, errors.New("simulated failure")
	}

	if m.createTaskFunc != nil {
		return m.createTaskFunc(ctx, req)
	}

	return &task.Task{
		ID:        int64(callCount),
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

func (m *MockBatchTaskClient) UpdateTask(ctx context.Context, taskID int64, req *task.UpdateRequest) (*task.Task, error) {
	m.mu.Lock()
	m.callCount++
	callCount := m.callCount
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.failureRate > 0 && float64(callCount%100)/100.0 < m.failureRate {
		return nil, errors.New("simulated failure")
	}

	if m.updateTaskFunc != nil {
		return m.updateTaskFunc(ctx, taskID, req)
	}

	return &task.Task{
		ID:        taskID,
		Name:      "updated-task",
		Status:    task.StatusPending,
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockBatchTaskClient) DeleteTask(ctx context.Context, taskID int64) error {
	m.mu.Lock()
	m.callCount++
	callCount := m.callCount
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.failureRate > 0 && float64(callCount%100)/100.0 < m.failureRate {
		return errors.New("simulated failure")
	}

	if m.deleteTaskFunc != nil {
		return m.deleteTaskFunc(ctx, taskID)
	}

	return nil
}

func (m *MockBatchTaskClient) GetTask(ctx context.Context, taskID int64) (*task.Task, error) {
	m.mu.Lock()
	m.callCount++
	callCount := m.callCount
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.failureRate > 0 && float64(callCount%100)/100.0 < m.failureRate {
		return nil, errors.New("simulated failure")
	}

	if m.getTaskFunc != nil {
		return m.getTaskFunc(ctx, taskID)
	}

	return &task.Task{
		ID:        taskID,
		Name:      "test-task",
		Status:    task.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockBatchTaskClient) CancelTask(ctx context.Context, taskID int64) (*task.Task, error) {
	m.mu.Lock()
	m.callCount++
	callCount := m.callCount
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.failureRate > 0 && float64(callCount%100)/100.0 < m.failureRate {
		return nil, errors.New("simulated failure")
	}

	if m.cancelTaskFunc != nil {
		return m.cancelTaskFunc(ctx, taskID)
	}

	return &task.Task{
		ID:        taskID,
		Name:      "cancelled-task",
		Status:    task.StatusCancelled,
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockBatchTaskClient) RetryTask(ctx context.Context, taskID int64) (*task.Task, error) {
	m.mu.Lock()
	m.callCount++
	callCount := m.callCount
	m.mu.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.failureRate > 0 && float64(callCount%100)/100.0 < m.failureRate {
		return nil, errors.New("simulated failure")
	}

	if m.retryTaskFunc != nil {
		return m.retryTaskFunc(ctx, taskID)
	}

	return &task.Task{
		ID:        taskID,
		Name:      "retried-task",
		Status:    task.StatusPending,
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockBatchTaskClient) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

func (m *MockBatchTaskClient) SetFailureRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureRate = rate
}

func (m *MockBatchTaskClient) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

func (m *MockBatchTaskClient) ResetCallCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
}

func createMockBatchClient() *task.Client {
	config := &sdk.Config{
		BaseURL:    "http://localhost:8080",
		APIKey:     "test-api-key",
		BusinessID: 1,
		Timeout:    30 * time.Second,
	}

	sdkClient, _ := sdk.NewClient(config)
	return task.NewClient(sdkClient)
}

func TestBatchClient_CreateTasks(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	// 创建测试请求
	requests := []*task.CreateRequest{
		task.NewCreateRequest().WithName("task1").WithType("test"),
		task.NewCreateRequest().WithName("task2").WithType("test"),
		task.NewCreateRequest().WithName("task3").WithType("test"),
	}

	ctx := context.Background()
	result, err := batchClient.CreateTasks(ctx, requests)

	if err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}

	if len(result.Success) != 3 {
		t.Errorf("Expected 3 successful tasks, got %d", len(result.Success))
	}

	if len(result.Failed) != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", len(result.Failed))
	}

	// 验证任务名称
	for i, createdTask := range result.Success {
		expectedName := requests[i].Name
		if createdTask.Name != expectedName {
			t.Errorf("Task %d: expected name '%s', got '%s'", i, expectedName, createdTask.Name)
		}
	}
}

func TestBatchClient_CreateTasks_WithFailures(t *testing.T) {
	mockClient := &MockBatchTaskClient{failureRate: 0.5} // 50%失败率
	taskClient := createMockBatchClient()
	taskClient = mockClient // 替换底层客户端

	batchClient := NewBatchClient(taskClient, nil)

	// 创建测试请求
	requests := make([]*task.CreateRequest, 10)
	for i := 0; i < 10; i++ {
		requests[i] = task.NewCreateRequest().WithName("task").WithType("test")
	}

	ctx := context.Background()
	result, err := batchClient.CreateTasks(ctx, requests)

	if err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}

	if result.Total != 10 {
		t.Errorf("Expected total 10, got %d", result.Total)
	}

	if len(result.Success)+len(result.Failed) != 10 {
		t.Errorf("Expected success + failed = 10, got %d + %d = %d",
			len(result.Success), len(result.Failed), len(result.Success)+len(result.Failed))
	}

	if len(result.Failed) == 0 {
		t.Error("Expected some failures with 50% failure rate")
	}

	// 验证失败项包含正确的索引和错误信息
	for _, failure := range result.Failed {
		if failure.Index < 0 || failure.Index >= 10 {
			t.Errorf("Invalid failure index: %d", failure.Index)
		}
		if failure.Error == nil {
			t.Error("Expected error in failure item")
		}
		if failure.Request == nil {
			t.Error("Expected request in failure item")
		}
	}
}

func TestBatchClient_UpdateTasks(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	// 创建测试更新请求
	updates := []BatchUpdateItem{
		{TaskID: 1, Request: task.NewUpdateRequest()},
		{TaskID: 2, Request: task.NewUpdateRequest()},
		{TaskID: 3, Request: task.NewUpdateRequest()},
	}

	ctx := context.Background()
	result, err := batchClient.UpdateTasks(ctx, updates)

	if err != nil {
		t.Fatalf("UpdateTasks failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}

	if len(result.Success) != 3 {
		t.Errorf("Expected 3 successful updates, got %d", len(result.Success))
	}

	if len(result.Failed) != 0 {
		t.Errorf("Expected 0 failed updates, got %d", len(result.Failed))
	}
}

func TestBatchClient_DeleteTasks(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	taskIDs := []int64{1, 2, 3, 4, 5}

	ctx := context.Background()
	result, err := batchClient.DeleteTasks(ctx, taskIDs)

	if err != nil {
		t.Fatalf("DeleteTasks failed: %v", err)
	}

	if result.Total != 5 {
		t.Errorf("Expected total 5, got %d", result.Total)
	}

	if len(result.Success) != 5 {
		t.Errorf("Expected 5 successful deletions, got %d", len(result.Success))
	}

	if len(result.Failed) != 0 {
		t.Errorf("Expected 0 failed deletions, got %d", len(result.Failed))
	}

	// 验证成功删除的任务ID
	for i, deletedID := range result.Success {
		expectedID := taskIDs[i]
		if deletedID != expectedID {
			t.Errorf("Expected deleted ID %d, got %d", expectedID, deletedID)
		}
	}
}

func TestBatchClient_QueryTasks(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	taskIDs := []int64{1, 2, 3}

	ctx := context.Background()
	result, err := batchClient.QueryTasks(ctx, taskIDs)

	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}

	if len(result.Success) != 3 {
		t.Errorf("Expected 3 successful queries, got %d", len(result.Success))
	}

	if len(result.Failed) != 0 {
		t.Errorf("Expected 0 failed queries, got %d", len(result.Failed))
	}

	// 验证查询到的任务ID
	for i, queriedTask := range result.Success {
		expectedID := taskIDs[i]
		if queriedTask.ID != expectedID {
			t.Errorf("Expected task ID %d, got %d", expectedID, queriedTask.ID)
		}
	}
}

func TestBatchClient_CancelTasks(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	taskIDs := []int64{1, 2, 3}

	ctx := context.Background()
	result, err := batchClient.CancelTasks(ctx, taskIDs)

	if err != nil {
		t.Fatalf("CancelTasks failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}

	if len(result.Success) != 3 {
		t.Errorf("Expected 3 successful cancellations, got %d", len(result.Success))
	}

	// 验证取消的任务状态
	for _, cancelledTask := range result.Success {
		if cancelledTask.Status != task.StatusCancelled {
			t.Errorf("Expected cancelled status, got %v", cancelledTask.Status)
		}
	}
}

func TestBatchClient_RetryTasks(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	taskIDs := []int64{1, 2, 3}

	ctx := context.Background()
	result, err := batchClient.RetryTasks(ctx, taskIDs)

	if err != nil {
		t.Fatalf("RetryTasks failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}

	if len(result.Success) != 3 {
		t.Errorf("Expected 3 successful retries, got %d", len(result.Success))
	}

	// 验证重试的任务状态
	for _, retriedTask := range result.Success {
		if retriedTask.Status != task.StatusPending {
			t.Errorf("Expected pending status after retry, got %v", retriedTask.Status)
		}
	}
}

func TestBatchClient_EmptyRequests(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	ctx := context.Background()

	// 测试空的创建请求
	createResult, err := batchClient.CreateTasks(ctx, []*task.CreateRequest{})
	if err != nil {
		t.Errorf("Expected no error for empty create requests, got: %v", err)
	}
	if createResult.Total != 0 {
		t.Errorf("Expected total 0 for empty requests, got %d", createResult.Total)
	}

	// 测试空的删除请求
	deleteResult, err := batchClient.DeleteTasks(ctx, []int64{})
	if err != nil {
		t.Errorf("Expected no error for empty delete requests, got: %v", err)
	}
	if deleteResult.Total != 0 {
		t.Errorf("Expected total 0 for empty requests, got %d", deleteResult.Total)
	}

	// 测试空的查询请求
	queryResult, err := batchClient.QueryTasks(ctx, []int64{})
	if err != nil {
		t.Errorf("Expected no error for empty query requests, got: %v", err)
	}
	if queryResult.Total != 0 {
		t.Errorf("Expected total 0 for empty requests, got %d", queryResult.Total)
	}
}

func TestBatchProcessor_ProcessCreateRequests(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, &BatchConfig{
		Concurrency: 5,
		Timeout:     30 * time.Second,
		BatchSize:   3,
	})

	processor := NewBatchProcessor(batchClient, 3)

	// 创建10个请求，应该分成4批处理（3+3+3+1）
	requests := make([]*task.CreateRequest, 10)
	for i := 0; i < 10; i++ {
		requests[i] = task.NewCreateRequest().WithName("task").WithType("test")
	}

	var processedBatches int
	ctx := context.Background()

	err := processor.ProcessCreateRequests(ctx, requests, func(result *BatchCreateResult) error {
		processedBatches++
		if len(result.Success) == 0 {
			return errors.New("no successful tasks in batch")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ProcessCreateRequests failed: %v", err)
	}

	expectedBatches := 4 // (10 + 3 - 1) / 3 = 4
	if processedBatches != expectedBatches {
		t.Errorf("Expected %d batches, got %d", expectedBatches, processedBatches)
	}
}

func TestStreamProcessor_Basic(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	streamProcessor := NewStreamProcessor(batchClient, 3, 100*time.Millisecond)
	defer streamProcessor.Close()

	// 添加任务
	for i := 0; i < 5; i++ {
		request := task.NewCreateRequest().WithName("stream-task").WithType("test")
		err := streamProcessor.Add(request)
		if err != nil {
			t.Fatalf("Failed to add task %d: %v", i, err)
		}
	}

	// 等待处理结果
	var receivedResults int
	timeout := time.After(2 * time.Second)

	for receivedResults < 2 { // 期望收到2个批次的结果（3+2）
		select {
		case result := <-streamProcessor.Results():
			receivedResults++
			if len(result.Success) == 0 {
				t.Error("Expected successful tasks in result")
			}
			t.Logf("Received batch result: %d successful, %d failed", len(result.Success), len(result.Failed))

		case err := <-streamProcessor.Errors():
			t.Fatalf("Received error: %v", err)

		case <-timeout:
			t.Fatalf("Timeout waiting for results, received %d batches", receivedResults)
		}
	}
}

func TestStreamProcessor_PeriodicFlush(t *testing.T) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	// 使用很短的超时时间来触发定期刷新
	streamProcessor := NewStreamProcessor(batchClient, 10, 50*time.Millisecond)
	defer streamProcessor.Close()

	// 添加少量任务（不会触发批次大小的刷新）
	for i := 0; i < 2; i++ {
		request := task.NewCreateRequest().WithName("periodic-task").WithType("test")
		err := streamProcessor.Add(request)
		if err != nil {
			t.Fatalf("Failed to add task %d: %v", i, err)
		}
	}

	// 等待定期刷新触发
	timeout := time.After(200 * time.Millisecond)

	select {
	case result := <-streamProcessor.Results():
		if len(result.Success) != 2 {
			t.Errorf("Expected 2 successful tasks, got %d", len(result.Success))
		}
		t.Log("Periodic flush worked correctly")

	case err := <-streamProcessor.Errors():
		t.Fatalf("Received error: %v", err)

	case <-timeout:
		t.Fatal("Timeout waiting for periodic flush")
	}
}

func TestCalculateStats(t *testing.T) {
	// 测试创建结果统计
	createResult := &BatchCreateResult{
		Success: make([]*task.Task, 7),
		Failed:  make([]BatchError, 3),
		Total:   10,
	}

	duration := 2 * time.Second
	stats := CalculateStats(createResult, duration)

	if stats.TotalRequested != 10 {
		t.Errorf("Expected total 10, got %d", stats.TotalRequested)
	}
	if stats.Successful != 7 {
		t.Errorf("Expected successful 7, got %d", stats.Successful)
	}
	if stats.Failed != 3 {
		t.Errorf("Expected failed 3, got %d", stats.Failed)
	}
	if stats.ErrorRate != 0.3 {
		t.Errorf("Expected error rate 0.3, got %f", stats.ErrorRate)
	}
	if stats.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, stats.Duration)
	}

	// 测试统计信息字符串输出
	statsStr := stats.String()
	if statsStr == "" {
		t.Error("Expected non-empty stats string")
	}
	t.Logf("Stats string: %s", statsStr)
}

func TestBatchClient_Concurrency(t *testing.T) {
	mockClient := &MockBatchTaskClient{delay: 100 * time.Millisecond}
	taskClient := createMockBatchClient()
	taskClient = mockClient

	// 设置低并发数来测试并发控制
	config := &BatchConfig{
		Concurrency: 2,
		Timeout:     5 * time.Second,
	}
	batchClient := NewBatchClient(taskClient, config)

	// 创建5个请求
	requests := make([]*task.CreateRequest, 5)
	for i := 0; i < 5; i++ {
		requests[i] = task.NewCreateRequest().WithName("concurrent-task").WithType("test")
	}

	start := time.Now()
	ctx := context.Background()
	result, err := batchClient.CreateTasks(ctx, requests)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CreateTasks failed: %v", err)
	}

	if len(result.Success) != 5 {
		t.Errorf("Expected 5 successful tasks, got %d", len(result.Success))
	}

	// 由于并发数为2，处理5个任务应该需要至少3个时间段（2+2+1）
	// 每个任务延迟100ms，所以至少需要300ms
	expectedMinDuration := 250 * time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected duration >= %v (due to concurrency limit), got %v", expectedMinDuration, duration)
	}

	// 但不应该超过串行执行的时间（500ms）
	expectedMaxDuration := 450 * time.Millisecond
	if duration > expectedMaxDuration {
		t.Errorf("Expected duration <= %v (due to concurrency), got %v", expectedMaxDuration, duration)
	}

	t.Logf("Concurrent execution took %v", duration)
}

func BenchmarkBatchClient_CreateTasks(b *testing.B) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)

	requests := make([]*task.CreateRequest, 10)
	for i := 0; i < 10; i++ {
		requests[i] = task.NewCreateRequest().WithName("benchmark-task").WithType("test")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchClient.CreateTasks(ctx, requests)
	}
}

func BenchmarkStreamProcessor_Add(b *testing.B) {
	mockClient := createMockBatchClient()
	batchClient := NewBatchClient(mockClient, nil)
	streamProcessor := NewStreamProcessor(batchClient, 100, 1*time.Second)
	defer streamProcessor.Close()

	request := task.NewCreateRequest().WithName("benchmark-stream-task").WithType("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		streamProcessor.Add(request)
	}
}