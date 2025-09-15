package task

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"task-center/sdk"
)

func TestOperations_BatchCreate(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tasks/batch" {
			t.Errorf("Expected path /api/v1/tasks/batch, got %s", r.URL.Path)
		}

		var req sdk.BatchCreateTasksRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if len(req.Tasks) != 2 {
			t.Errorf("Expected 2 tasks in batch request, got %d", len(req.Tasks))
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.BatchCreateTasksResponse{
				Succeeded: []sdk.Task{
					{
						ID:               1,
						BusinessUniqueID: req.Tasks[0].BusinessUniqueID,
						Status:           sdk.TaskStatusPending,
					},
					{
						ID:               2,
						BusinessUniqueID: req.Tasks[1].BusinessUniqueID,
						Status:           sdk.TaskStatusPending,
					},
				},
				Failed: []sdk.BatchTaskError{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ops := NewOperations(client)

	batchReq := NewBatchCreateRequest()
	batchReq.AddTask(NewCreateRequest("task1", "https://example.com/callback1"))
	batchReq.AddTask(NewCreateRequest("task2", "https://example.com/callback2"))

	ctx := context.Background()
	resp, err := ops.BatchCreate(ctx, batchReq)
	if err != nil {
		t.Fatalf("BatchCreate() error = %v", err)
	}

	if len(resp.Succeeded) != 2 {
		t.Errorf("Expected 2 succeeded tasks, got %d", len(resp.Succeeded))
	}

	if len(resp.Failed) != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", len(resp.Failed))
	}

	if resp.Succeeded[0].ID != 1 {
		t.Errorf("Expected first task ID 1, got %d", resp.Succeeded[0].ID)
	}
}

func TestOperations_BatchUpdate(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tasks/batch" {
			t.Errorf("Expected path /api/v1/tasks/batch, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		updates, ok := reqBody["updates"].([]interface{})
		if !ok {
			t.Error("Expected updates field in request body")
		}

		if len(updates) != 1 {
			t.Errorf("Expected 1 update in batch request, got %d", len(updates))
		}

		response := BatchUpdateResponse{
			Succeeded: []*Task{
				{
					Task: &sdk.Task{
						ID:     123,
						Status: sdk.TaskStatusCancelled,
					},
				},
			},
			Failed: []*BatchTaskError{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ops := NewOperations(client)

	updates := []BatchUpdateItem{
		{
			TaskID:  123,
			Request: NewUpdateRequest().WithStatus(StatusCancelled),
		},
	}

	ctx := context.Background()
	resp, err := ops.BatchUpdate(ctx, updates)
	if err != nil {
		t.Fatalf("BatchUpdate() error = %v", err)
	}

	if len(resp.Succeeded) != 1 {
		t.Errorf("Expected 1 succeeded update, got %d", len(resp.Succeeded))
	}

	if resp.Succeeded[0].Status != StatusCancelled {
		t.Errorf("Expected cancelled status, got %d", resp.Succeeded[0].Status)
	}
}

func TestOperations_BatchCancel(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tasks/batch/cancel" {
			t.Errorf("Expected path /api/v1/tasks/batch/cancel, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		taskIDs, ok := reqBody["task_ids"].([]interface{})
		if !ok {
			t.Error("Expected task_ids field in request body")
		}

		if len(taskIDs) != 2 {
			t.Errorf("Expected 2 task IDs, got %d", len(taskIDs))
		}

		response := BatchCancelResponse{
			Succeeded: []*Task{
				{Task: &sdk.Task{ID: 1, Status: sdk.TaskStatusCancelled}},
				{Task: &sdk.Task{ID: 2, Status: sdk.TaskStatusCancelled}},
			},
			Failed: []*BatchTaskError{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ops := NewOperations(client)

	taskIDs := []int64{1, 2}
	ctx := context.Background()
	resp, err := ops.BatchCancel(ctx, taskIDs)
	if err != nil {
		t.Fatalf("BatchCancel() error = %v", err)
	}

	if len(resp.Succeeded) != 2 {
		t.Errorf("Expected 2 succeeded cancellations, got %d", len(resp.Succeeded))
	}

	for i, task := range resp.Succeeded {
		if task.Status != StatusCancelled {
			t.Errorf("Expected task %d to be cancelled, got status %d", i, task.Status)
		}
	}
}

func TestOperations_BatchDelete(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tasks/batch" {
			t.Errorf("Expected path /api/v1/tasks/batch, got %s", r.URL.Path)
		}

		response := BatchDeleteResponse{
			Succeeded: []int64{1, 2, 3},
			Failed:    []*BatchTaskError{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ops := NewOperations(client)

	taskIDs := []int64{1, 2, 3}
	ctx := context.Background()
	resp, err := ops.BatchDelete(ctx, taskIDs)
	if err != nil {
		t.Fatalf("BatchDelete() error = %v", err)
	}

	if len(resp.Succeeded) != 3 {
		t.Errorf("Expected 3 succeeded deletions, got %d", len(resp.Succeeded))
	}

	for i, taskID := range resp.Succeeded {
		expectedID := int64(i + 1)
		if taskID != expectedID {
			t.Errorf("Expected succeeded task ID %d, got %d", expectedID, taskID)
		}
	}
}

func TestConcurrentOperations_ConcurrentCreate(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		currentCount := requestCount
		mu.Unlock()

		if r.Method != "POST" || r.URL.Path != "/api/v1/tasks" {
			t.Errorf("Invalid request: %s %s", r.Method, r.URL.Path)
		}

		var req sdk.CreateTaskRequest
		json.NewDecoder(r.Body).Decode(&req)

		// 模拟一些处理时间
		time.Sleep(10 * time.Millisecond)

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:               int64(currentCount),
				BusinessUniqueID: req.BusinessUniqueID,
				Status:           sdk.TaskStatusPending,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	cop := NewConcurrentOperations(client, 3, 5*time.Second)

	requests := []*CreateRequest{
		NewCreateRequest("task1", "https://example.com/callback1"),
		NewCreateRequest("task2", "https://example.com/callback2"),
		NewCreateRequest("task3", "https://example.com/callback3"),
		NewCreateRequest("task4", "https://example.com/callback4"),
		NewCreateRequest("task5", "https://example.com/callback5"),
	}

	ctx := context.Background()
	start := time.Now()
	results := cop.ConcurrentCreate(ctx, requests)
	duration := time.Since(start)

	// 验证结果
	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	successCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		}
	}

	if successCount != 5 {
		t.Errorf("Expected 5 successful results, got %d", successCount)
	}

	// 并发执行应该比顺序执行快
	expectedSequentialTime := 5 * 10 * time.Millisecond // 5 tasks * 10ms each
	if duration >= expectedSequentialTime {
		t.Logf("Concurrent execution took %v, expected less than %v", duration, expectedSequentialTime)
		// 注意：这个测试可能在某些环境下不稳定，所以只记录日志而不失败
	}
}

func TestTaskWatcher(t *testing.T) {
	taskID := int64(123)
	callCount := 0

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/api/v1/tasks/123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// 模拟状态变化
		var status sdk.TaskStatus
		switch callCount {
		case 1:
			status = sdk.TaskStatusPending
		case 2:
			status = sdk.TaskStatusRunning
		default:
			status = sdk.TaskStatusSucceeded
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:     taskID,
				Status: status,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	watcher := NewTaskWatcher(client, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动监控
	go watcher.Start(ctx)

	// 监控任务
	taskChan := watcher.WatchTask(ctx, taskID)

	// 收集状态更新
	var statuses []TaskStatus
	timeout := time.NewTimer(200 * time.Millisecond)

	collectLoop:
	for len(statuses) < 3 {
		select {
		case task := <-taskChan:
			statuses = append(statuses, task.Status)
		case <-timeout.C:
			break collectLoop
		}
	}

	// 停止监控
	watcher.StopWatching(taskID)
	watcher.Stop()

	// 验证状态变化
	if len(statuses) < 2 {
		t.Errorf("Expected at least 2 status updates, got %d", len(statuses))
	}

	// 验证状态序列
	expectedStatuses := []TaskStatus{StatusPending, StatusRunning, StatusSucceeded}
	for i, status := range statuses {
		if i < len(expectedStatuses) && status != expectedStatuses[i] {
			t.Errorf("Expected status %d at position %d, got %d", expectedStatuses[i], i, status)
		}
	}
}

func TestTaskScheduler(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/tasks" {
			t.Errorf("Invalid request: %s %s", r.Method, r.URL.Path)
		}

		var req sdk.CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// 验证计划执行时间
		if req.ScheduledAt == nil {
			t.Error("Expected scheduled time to be set")
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:          1,
				ScheduledAt: *req.ScheduledAt,
				Status:      sdk.TaskStatusPending,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	scheduler := NewTaskScheduler(client)

	// 测试定时调度
	scheduledTime := time.Now().Add(time.Hour)
	req := NewCreateRequest("scheduled-task", "https://example.com/callback")

	ctx := context.Background()
	task, err := scheduler.ScheduleTask(ctx, req, scheduledTime)
	if err != nil {
		t.Fatalf("ScheduleTask() error = %v", err)
	}

	if task.ScheduledAt.Unix() != scheduledTime.Unix() {
		t.Errorf("Expected scheduled time %v, got %v", scheduledTime, task.ScheduledAt)
	}

	// 测试延迟调度
	delay := 30 * time.Minute
	req2 := NewCreateRequest("delayed-task", "https://example.com/callback")

	task2, err := scheduler.ScheduleTaskAfter(ctx, req2, delay)
	if err != nil {
		t.Fatalf("ScheduleTaskAfter() error = %v", err)
	}

	expectedTime := time.Now().Add(delay)
	timeDiff := task2.ScheduledAt.Sub(expectedTime).Abs()
	if timeDiff > time.Second {
		t.Errorf("Scheduled time difference too large: %v", timeDiff)
	}
}

func TestTaskScheduler_CronTask(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req sdk.CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// 验证 cron 元数据
		if req.Metadata == nil {
			t.Error("Expected metadata to be set")
		}

		cronExpr, ok := req.Metadata["cron_expression"].(string)
		if !ok || cronExpr != "0 0 * * *" {
			t.Errorf("Expected cron expression '0 0 * * *', got %v", cronExpr)
		}

		taskType, ok := req.Metadata["task_type"].(string)
		if !ok || taskType != "cron" {
			t.Errorf("Expected task type 'cron', got %v", taskType)
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:       1,
				Metadata: req.Metadata,
				Status:   sdk.TaskStatusPending,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	scheduler := NewTaskScheduler(client)

	req := NewCreateRequest("cron-task", "https://example.com/callback")
	ctx := context.Background()

	task, err := scheduler.ScheduleCronTask(ctx, req, "0 0 * * *")
	if err != nil {
		t.Fatalf("ScheduleCronTask() error = %v", err)
	}

	if task.Metadata["cron_expression"] != "0 0 * * *" {
		t.Errorf("Expected cron expression in task metadata")
	}
}

func TestTaskQuery(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if !strings.HasPrefix(r.URL.Path, "/api/v1/tasks") {
			t.Errorf("Expected path to start with /api/v1/tasks, got %s", r.URL.Path)
		}

		// 验证查询参数
		query := r.URL.Query()
		if query.Get("status") != "2" { // StatusSucceeded
			t.Errorf("Expected status query parameter 2, got %s", query.Get("status"))
		}

		if query.Get("tags") != "production" {
			t.Errorf("Expected tags query parameter production, got %s", query.Get("tags"))
		}

		if query.Get("priority") != "3" { // PriorityHigh
			t.Errorf("Expected priority query parameter 3, got %s", query.Get("priority"))
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.ListTasksResponse{
				Tasks: []sdk.Task{
					{
						ID:       1,
						Status:   sdk.TaskStatusSucceeded,
						Priority: sdk.TaskPriorityHigh,
						Tags:     []string{"production"},
					},
					{
						ID:       2,
						Status:   sdk.TaskStatusSucceeded,
						Priority: sdk.TaskPriorityHigh,
						Tags:     []string{"production"},
					},
				},
				Total:      2,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	query := NewTaskQuery(client).
		Status(StatusSucceeded).
		Tags("production").
		Priority(PriorityHigh).
		Pagination(1, 20)

	ctx := context.Background()

	// 测试执行查询
	resp, err := query.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(resp.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(resp.Tasks))
	}

	// 测试获取数量
	count, err := query.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// 测试获取第一个
	first, err := query.First(ctx)
	if err != nil {
		t.Fatalf("First() error = %v", err)
	}

	if first.ID != 1 {
		t.Errorf("Expected first task ID 1, got %d", first.ID)
	}
}

func TestOperations_ValidationErrors(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Server handler should not be called for validation errors")
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ops := NewOperations(client)
	ctx := context.Background()

	// Test nil batch create request
	_, err := ops.BatchCreate(ctx, nil)
	if err == nil {
		t.Error("Expected validation error for nil batch create request")
	}

	// Test empty batch create request
	emptyBatch := NewBatchCreateRequest()
	_, err = ops.BatchCreate(ctx, emptyBatch)
	if err == nil {
		t.Error("Expected validation error for empty batch create request")
	}

	// Test empty task IDs for batch operations
	_, err = ops.BatchCancel(ctx, []int64{})
	if err == nil {
		t.Error("Expected validation error for empty task IDs")
	}

	_, err = ops.BatchRetry(ctx, []int64{})
	if err == nil {
		t.Error("Expected validation error for empty task IDs")
	}

	_, err = ops.BatchDelete(ctx, []int64{})
	if err == nil {
		t.Error("Expected validation error for empty task IDs")
	}

	// Test empty updates for batch update
	_, err = ops.BatchUpdate(ctx, []BatchUpdateItem{})
	if err == nil {
		t.Error("Expected validation error for empty batch update")
	}
}

func TestConcurrentOperations_Timeout(t *testing.T) {
	// 创建一个响应很慢的服务器
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // 比超时时间长

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:     1,
				Status: sdk.TaskStatusPending,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	// 设置很短的超时时间
	cop := NewConcurrentOperations(client, 2, 50*time.Millisecond)

	requests := []*CreateRequest{
		NewCreateRequest("task1", "https://example.com/callback1"),
	}

	ctx := context.Background()
	results := cop.ConcurrentCreate(ctx, requests)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// 应该有超时错误
	if results[0].Error == nil {
		t.Error("Expected timeout error, got nil")
	}
}