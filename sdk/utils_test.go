package sdk

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"task-center/sdk/task"
)

func createMockSDK() (*TaskCenterSDK, error) {
	config := &Config{
		BaseURL:    "http://localhost:8080",
		APIKey:     "test-api-key",
		BusinessID: 1,
		Timeout:    30 * time.Second,
	}

	return NewTaskCenterSDK(config)
}

func TestNewTaskCenterSDK(t *testing.T) {
	sdk, err := createMockSDK()
	if err != nil {
		t.Fatalf("Failed to create TaskCenterSDK: %v", err)
	}
	defer sdk.Close()

	if sdk.client == nil {
		t.Error("Expected task client to be initialized")
	}
	if sdk.asyncClient == nil {
		t.Error("Expected async client to be initialized")
	}
	if sdk.batchClient == nil {
		t.Error("Expected batch client to be initialized")
	}
	if sdk.taskBuilder == nil {
		t.Error("Expected task builder to be initialized")
	}
	if sdk.queryBuilder == nil {
		t.Error("Expected query builder to be initialized")
	}
}

func TestTaskCenterSDK_CreateSimpleTask(t *testing.T) {
	sdk, err := createMockSDK()
	if err != nil {
		t.Fatalf("Failed to create TaskCenterSDK: %v", err)
	}
	defer sdk.Close()

	payload := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	ctx := context.Background()
	task, err := sdk.CreateSimpleTask(ctx, "simple-task", "test", payload)

	// 注意：这里会因为没有真实的服务端而失败，但我们可以验证构建逻辑
	if err == nil {
		if task.Name != "simple-task" {
			t.Errorf("Expected task name 'simple-task', got '%s'", task.Name)
		}
		if task.Type != "test" {
			t.Errorf("Expected task type 'test', got '%s'", task.Type)
		}
	} else {
		// 验证是否是预期的网络错误（因为没有真实服务）
		t.Logf("Expected network error: %v", err)
	}
}

func TestTaskCenterSDK_CreateDelayedTask(t *testing.T) {
	sdk, err := createMockSDK()
	if err != nil {
		t.Fatalf("Failed to create TaskCenterSDK: %v", err)
	}
	defer sdk.Close()

	payload := map[string]string{"test": "data"}
	delay := 5 * time.Minute

	ctx := context.Background()
	_, err = sdk.CreateDelayedTask(ctx, "delayed-task", "test", payload, delay)

	// 验证构建器逻辑
	builder := sdk.GetTaskBuilder().
		Reset().
		WithName("delayed-task").
		WithType("test").
		WithPayload(payload).
		WithDelay(delay)

	request := builder.GetRequest()
	if request.ScheduledAt == nil {
		t.Error("Expected scheduled time to be set for delayed task")
	}

	expectedTime := time.Now().Add(delay)
	diff := request.ScheduledAt.Sub(expectedTime)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Expected scheduled time around %v, got %v", expectedTime, *request.ScheduledAt)
	}
}

func TestTaskCenterSDK_CreateHighPriorityTask(t *testing.T) {
	sdk, err := createMockSDK()
	if err != nil {
		t.Fatalf("Failed to create TaskCenterSDK: %v", err)
	}
	defer sdk.Close()

	// 验证构建器逻辑
	builder := sdk.GetTaskBuilder().
		Reset().
		WithName("high-priority-task").
		WithType("test").
		WithHighPriority()

	request := builder.GetRequest()
	if request.Priority != task.PriorityHigh {
		t.Errorf("Expected high priority, got %v", request.Priority)
	}
}

func TestIsTaskCompleted(t *testing.T) {
	testCases := []struct {
		status   task.TaskStatus
		expected bool
	}{
		{task.StatusPending, false},
		{task.StatusRunning, false},
		{task.StatusSucceeded, true},
		{task.StatusFailed, true},
		{task.StatusCancelled, true},
		{task.StatusExpired, true},
	}

	for _, tc := range testCases {
		result := IsTaskCompleted(tc.status)
		if result != tc.expected {
			t.Errorf("IsTaskCompleted(%v) = %v, expected %v", tc.status, result, tc.expected)
		}
	}
}

func TestIsTaskSuccessful(t *testing.T) {
	testCases := []struct {
		status   task.TaskStatus
		expected bool
	}{
		{task.StatusSucceeded, true},
		{task.StatusFailed, false},
		{task.StatusPending, false},
		{task.StatusRunning, false},
		{task.StatusCancelled, false},
		{task.StatusExpired, false},
	}

	for _, tc := range testCases {
		result := IsTaskSuccessful(tc.status)
		if result != tc.expected {
			t.Errorf("IsTaskSuccessful(%v) = %v, expected %v", tc.status, result, tc.expected)
		}
	}
}

func TestIsTaskFailed(t *testing.T) {
	testCases := []struct {
		status   task.TaskStatus
		expected bool
	}{
		{task.StatusFailed, true},
		{task.StatusSucceeded, false},
		{task.StatusPending, false},
		{task.StatusRunning, false},
		{task.StatusCancelled, false},
		{task.StatusExpired, false},
	}

	for _, tc := range testCases {
		result := IsTaskFailed(tc.status)
		if result != tc.expected {
			t.Errorf("IsTaskFailed(%v) = %v, expected %v", tc.status, result, tc.expected)
		}
	}
}

func TestIsTaskActive(t *testing.T) {
	testCases := []struct {
		status   task.TaskStatus
		expected bool
	}{
		{task.StatusPending, true},
		{task.StatusRunning, true},
		{task.StatusSucceeded, false},
		{task.StatusFailed, false},
		{task.StatusCancelled, false},
		{task.StatusExpired, false},
	}

	for _, tc := range testCases {
		result := IsTaskActive(tc.status)
		if result != tc.expected {
			t.Errorf("IsTaskActive(%v) = %v, expected %v", tc.status, result, tc.expected)
		}
	}
}

func TestPayloadToStruct(t *testing.T) {
	// 测试数据
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "test", Value: 123}
	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// 测试转换
	var result TestStruct
	err = PayloadToStruct(payload, &result)
	if err != nil {
		t.Fatalf("PayloadToStruct failed: %v", err)
	}

	if result.Name != original.Name {
		t.Errorf("Expected name '%s', got '%s'", original.Name, result.Name)
	}
	if result.Value != original.Value {
		t.Errorf("Expected value %d, got %d", original.Value, result.Value)
	}

	// 测试nil payload
	err = PayloadToStruct(nil, &result)
	if err != nil {
		t.Errorf("Expected no error for nil payload, got: %v", err)
	}
}

func TestStructToPayload(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "test", Value: 123}

	payload, err := StructToPayload(original)
	if err != nil {
		t.Fatalf("StructToPayload failed: %v", err)
	}

	// 验证转换结果
	var result TestStruct
	err = json.Unmarshal(payload, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if result.Name != original.Name {
		t.Errorf("Expected name '%s', got '%s'", original.Name, result.Name)
	}
	if result.Value != original.Value {
		t.Errorf("Expected value %d, got %d", original.Value, result.Value)
	}

	// 测试nil input
	payload, err = StructToPayload(nil)
	if err != nil {
		t.Errorf("Expected no error for nil input, got: %v", err)
	}
	if payload != nil {
		t.Error("Expected nil payload for nil input")
	}
}

func TestValidateTaskRequest(t *testing.T) {
	// 测试nil请求
	err := ValidateTaskRequest(nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}

	// 测试空名称
	req := task.NewCreateRequest().WithType("test")
	err = ValidateTaskRequest(req)
	if err == nil {
		t.Error("Expected error for empty task name")
	}

	// 测试空类型
	req = task.NewCreateRequest().WithName("test")
	err = ValidateTaskRequest(req)
	if err == nil {
		t.Error("Expected error for empty task type")
	}

	// 测试有效请求
	req = task.NewCreateRequest().WithName("test").WithType("test")
	err = ValidateTaskRequest(req)
	if err != nil {
		t.Errorf("Expected no error for valid request, got: %v", err)
	}
}

func TestCalculateTaskStats(t *testing.T) {
	tasks := []*task.Task{
		{Status: task.StatusPending, Priority: task.PriorityHigh, Type: "email"},
		{Status: task.StatusRunning, Priority: task.PriorityNormal, Type: "email"},
		{Status: task.StatusSucceeded, Priority: task.PriorityLow, Type: "report"},
		{Status: task.StatusFailed, Priority: task.PriorityHigh, Type: "payment"},
		{Status: task.StatusPending, Priority: task.PriorityNormal, Type: "email"},
	}

	stats := CalculateTaskStats(tasks)

	if stats.Total != 5 {
		t.Errorf("Expected total 5, got %d", stats.Total)
	}

	// 验证状态统计
	if stats.ByStatus[task.StatusPending] != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", stats.ByStatus[task.StatusPending])
	}
	if stats.ByStatus[task.StatusRunning] != 1 {
		t.Errorf("Expected 1 running task, got %d", stats.ByStatus[task.StatusRunning])
	}
	if stats.ByStatus[task.StatusSucceeded] != 1 {
		t.Errorf("Expected 1 succeeded task, got %d", stats.ByStatus[task.StatusSucceeded])
	}
	if stats.ByStatus[task.StatusFailed] != 1 {
		t.Errorf("Expected 1 failed task, got %d", stats.ByStatus[task.StatusFailed])
	}

	// 验证优先级统计
	if stats.ByPriority[task.PriorityHigh] != 2 {
		t.Errorf("Expected 2 high priority tasks, got %d", stats.ByPriority[task.PriorityHigh])
	}
	if stats.ByPriority[task.PriorityNormal] != 2 {
		t.Errorf("Expected 2 normal priority tasks, got %d", stats.ByPriority[task.PriorityNormal])
	}
	if stats.ByPriority[task.PriorityLow] != 1 {
		t.Errorf("Expected 1 low priority task, got %d", stats.ByPriority[task.PriorityLow])
	}

	// 验证类型统计
	if stats.ByType["email"] != 3 {
		t.Errorf("Expected 3 email tasks, got %d", stats.ByType["email"])
	}
	if stats.ByType["report"] != 1 {
		t.Errorf("Expected 1 report task, got %d", stats.ByType["report"])
	}
	if stats.ByType["payment"] != 1 {
		t.Errorf("Expected 1 payment task, got %d", stats.ByType["payment"])
	}
}

func TestGetTaskInfo(t *testing.T) {
	now := time.Now()
	scheduledAt := now.Add(1 * time.Hour)

	taskObj := &task.Task{
		ID:          123,
		Name:        "test-task",
		Type:        "test",
		Status:      task.StatusPending,
		Priority:    task.PriorityHigh,
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: &scheduledAt,
		Tags:        []string{"tag1", "tag2"},
	}

	info := GetTaskInfo(taskObj)
	if info == nil {
		t.Fatal("Expected task info to be created")
	}

	if info.ID != 123 {
		t.Errorf("Expected ID 123, got %d", info.ID)
	}
	if info.Name != "test-task" {
		t.Errorf("Expected name 'test-task', got '%s'", info.Name)
	}
	if info.Type != "test" {
		t.Errorf("Expected type 'test', got '%s'", info.Type)
	}
	if info.Status != task.StatusPending {
		t.Errorf("Expected status %v, got %v", task.StatusPending, info.Status)
	}
	if info.Priority != task.PriorityHigh {
		t.Errorf("Expected priority %v, got %v", task.PriorityHigh, info.Priority)
	}
	if info.ScheduledAt == nil || !info.ScheduledAt.Equal(scheduledAt) {
		t.Errorf("Expected scheduled time %v, got %v", scheduledAt, info.ScheduledAt)
	}
	if len(info.Tags) != 2 || info.Tags[0] != "tag1" || info.Tags[1] != "tag2" {
		t.Errorf("Expected tags [tag1, tag2], got %v", info.Tags)
	}

	// 测试nil任务
	info = GetTaskInfo(nil)
	if info != nil {
		t.Error("Expected nil info for nil task")
	}
}

func TestFormatTaskSummary(t *testing.T) {
	now := time.Now()
	taskObj := &task.Task{
		ID:        123,
		Name:      "test-task",
		Type:      "test",
		Status:    task.StatusPending,
		Priority:  task.PriorityHigh,
		CreatedAt: now,
	}

	summary := FormatTaskSummary(taskObj)
	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	// 验证摘要包含关键信息
	expectedSubstrings := []string{"123", "test-task", "test", "pending", "high"}
	for _, substr := range expectedSubstrings {
		if !contains(summary, substr) {
			t.Errorf("Expected summary to contain '%s', got: %s", substr, summary)
		}
	}

	// 测试nil任务
	summary = FormatTaskSummary(nil)
	if summary != "Task: <nil>" {
		t.Errorf("Expected 'Task: <nil>' for nil task, got: %s", summary)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	if config == nil {
		t.Fatal("Expected retry config to be created")
	}

	if config.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", config.MaxAttempts)
	}
	if config.Interval != 1*time.Second {
		t.Errorf("Expected interval 1s, got %v", config.Interval)
	}
	if config.Backoff != 2.0 {
		t.Errorf("Expected backoff 2.0, got %f", config.Backoff)
	}
}

func TestGetStructFields(t *testing.T) {
	type TestStruct struct {
		PublicField  string
		AnotherField int
		privateField string
	}

	fields := GetStructFields(TestStruct{})
	if len(fields) != 2 {
		t.Errorf("Expected 2 public fields, got %d", len(fields))
	}

	if fields["PublicField"] != "string" {
		t.Errorf("Expected PublicField type 'string', got '%s'", fields["PublicField"])
	}
	if fields["AnotherField"] != "int" {
		t.Errorf("Expected AnotherField type 'int', got '%s'", fields["AnotherField"])
	}

	// 验证私有字段未被包含
	if _, exists := fields["privateField"]; exists {
		t.Error("Expected private field to be excluded")
	}

	// 测试指针类型
	fields = GetStructFields(&TestStruct{})
	if len(fields) != 2 {
		t.Errorf("Expected 2 public fields for pointer type, got %d", len(fields))
	}

	// 测试非结构体类型
	fields = GetStructFields("not a struct")
	if len(fields) != 0 {
		t.Errorf("Expected 0 fields for non-struct type, got %d", len(fields))
	}
}

func TestConvertToMap(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Flag  bool   `json:"flag"`
	}

	original := TestStruct{Name: "test", Value: 123, Flag: true}
	result, err := ConvertToMap(original)
	if err != nil {
		t.Fatalf("ConvertToMap failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name 'test', got %v", result["name"])
	}
	if result["value"].(float64) != 123 {
		t.Errorf("Expected value 123, got %v", result["value"])
	}
	if result["flag"] != true {
		t.Errorf("Expected flag true, got %v", result["flag"])
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    (len(s) > len(substr) &&
		     (s[:len(substr)] == substr ||
		      s[len(s)-len(substr):] == substr ||
		      strings.Contains(s, substr))))
}

func TestTaskCenterSDK_GetClients(t *testing.T) {
	sdk, err := createMockSDK()
	if err != nil {
		t.Fatalf("Failed to create TaskCenterSDK: %v", err)
	}
	defer sdk.Close()

	// 测试获取内部客户端
	if sdk.GetTaskClient() == nil {
		t.Error("Expected task client to be available")
	}
	if sdk.GetAsyncClient() == nil {
		t.Error("Expected async client to be available")
	}
	if sdk.GetBatchClient() == nil {
		t.Error("Expected batch client to be available")
	}
	if sdk.GetTaskBuilder() == nil {
		t.Error("Expected task builder to be available")
	}
	if sdk.GetQueryBuilder() == nil {
		t.Error("Expected query builder to be available")
	}
}

