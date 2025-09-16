package builder

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"task-center/sdk"
	"task-center/sdk/task"
)

// MockTaskClient 用于测试的模拟任务客户端
type MockTaskClient struct {
	createTaskFunc func(ctx context.Context, req *task.CreateRequest) (*task.Task, error)
}

func (m *MockTaskClient) CreateTask(ctx context.Context, req *task.CreateRequest) (*task.Task, error) {
	if m.createTaskFunc != nil {
		return m.createTaskFunc(ctx, req)
	}
	// 返回默认的模拟任务
	return &task.Task{
		ID:          1,
		Name:        req.Name,
		Type:        req.Type,
		Status:      task.StatusPending,
		Priority:    req.Priority,
		Payload:     req.Payload,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func createMockClient() *task.Client {
	// 创建一个简单的模拟客户端
	config := &sdk.Config{
		BaseURL:    "http://localhost:8080",
		APIKey:     "test-api-key",
		BusinessID: 1,
		Timeout:    30 * time.Second,
	}

	sdkClient, _ := sdk.NewClient(config)
	return task.NewClient(sdkClient)
}

func TestTaskBuilder_BasicChaining(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	// 测试链式调用
	request := builder.
		WithName("test-task").
		WithType("test-type").
		WithPriority(task.PriorityHigh).
		WithTag("test").
		WithDescription("Test task description").
		GetRequest()

	// 验证设置
	if request.Name != "test-task" {
		t.Errorf("Expected name 'test-task', got '%s'", request.Name)
	}
	if request.Type != "test-type" {
		t.Errorf("Expected type 'test-type', got '%s'", request.Type)
	}
	if request.Priority != task.PriorityHigh {
		t.Errorf("Expected priority %v, got %v", task.PriorityHigh, request.Priority)
	}
	if len(request.Tags) != 1 || request.Tags[0] != "test" {
		t.Errorf("Expected tags ['test'], got %v", request.Tags)
	}
	if request.Description == nil || *request.Description != "Test task description" {
		t.Errorf("Expected description 'Test task description', got %v", request.Description)
	}
}

func TestTaskBuilder_WithPayload(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	payload := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	request := builder.
		WithName("payload-test").
		WithType("test").
		WithPayload(payload).
		GetRequest()

	// 验证payload序列化
	var deserializedPayload map[string]interface{}
	err := json.Unmarshal(request.Payload, &deserializedPayload)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if deserializedPayload["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got %v", deserializedPayload["key1"])
	}
	if deserializedPayload["key2"].(float64) != 123 {
		t.Errorf("Expected key2=123, got %v", deserializedPayload["key2"])
	}
	if deserializedPayload["key3"] != true {
		t.Errorf("Expected key3=true, got %v", deserializedPayload["key3"])
	}
}

func TestTaskBuilder_WithDelayAndScheduling(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	delay := 5 * time.Minute
	now := time.Now()

	request := builder.
		WithName("delayed-task").
		WithDelay(delay).
		GetRequest()

	if request.ScheduledAt == nil {
		t.Fatal("Expected ScheduledAt to be set")
	}

	// 验证调度时间（允许1秒误差）
	expectedTime := now.Add(delay)
	diff := request.ScheduledAt.Sub(expectedTime)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Expected scheduled time around %v, got %v", expectedTime, *request.ScheduledAt)
	}
}

func TestTaskBuilder_WithRetryPolicy(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	maxRetries := 5
	retryInterval := 10 * time.Second

	request := builder.
		WithName("retry-task").
		WithRetryPolicy(maxRetries, retryInterval).
		GetRequest()

	if request.MaxRetries == nil || *request.MaxRetries != maxRetries {
		t.Errorf("Expected MaxRetries=%d, got %v", maxRetries, request.MaxRetries)
	}
	if request.RetryInterval == nil || *request.RetryInterval != int(retryInterval.Seconds()) {
		t.Errorf("Expected RetryInterval=%d, got %v", int(retryInterval.Seconds()), request.RetryInterval)
	}
}

func TestTaskBuilder_WithCallback(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	callbackURL := "http://example.com/callback"
	headers := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
	}

	request := builder.
		WithName("callback-task").
		WithCallback(callbackURL).
		WithCallbackHeaders(headers).
		WithCallbackHeader("X-Custom", "value").
		GetRequest()

	if request.CallbackURL == nil || *request.CallbackURL != callbackURL {
		t.Errorf("Expected CallbackURL='%s', got %v", callbackURL, request.CallbackURL)
	}

	if request.CallbackHeaders == nil {
		t.Fatal("Expected CallbackHeaders to be set")
	}

	if request.CallbackHeaders["Authorization"] != "Bearer token" {
		t.Errorf("Expected Authorization header, got %v", request.CallbackHeaders["Authorization"])
	}
	if request.CallbackHeaders["X-Custom"] != "value" {
		t.Errorf("Expected X-Custom header, got %v", request.CallbackHeaders["X-Custom"])
	}
}

func TestTaskBuilder_Clone(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	original := builder.
		WithName("original-task").
		WithType("test").
		WithTag("original").
		WithPriority(task.PriorityHigh)

	// 克隆构建器
	cloned := original.Clone()

	// 修改克隆的构建器
	cloned.WithName("cloned-task").WithTag("cloned")

	originalRequest := original.GetRequest()
	clonedRequest := cloned.GetRequest()

	// 验证原始构建器未被修改
	if originalRequest.Name != "original-task" {
		t.Errorf("Original builder name was modified: %s", originalRequest.Name)
	}
	if len(originalRequest.Tags) != 1 || originalRequest.Tags[0] != "original" {
		t.Errorf("Original builder tags were modified: %v", originalRequest.Tags)
	}

	// 验证克隆的构建器被正确修改
	if clonedRequest.Name != "cloned-task" {
		t.Errorf("Cloned builder name not set: %s", clonedRequest.Name)
	}
	if len(clonedRequest.Tags) != 2 {
		t.Errorf("Expected 2 tags in cloned builder, got %d", len(clonedRequest.Tags))
	}
}

func TestTaskBuilder_Reset(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	// 设置一些值
	builder.
		WithName("test-task").
		WithType("test").
		WithTag("test")

	// 重置构建器
	builder.Reset()

	request := builder.GetRequest()

	// 验证所有设置都被重置
	if request.Name != "" {
		t.Errorf("Expected empty name after reset, got '%s'", request.Name)
	}
	if request.Type != "" {
		t.Errorf("Expected empty type after reset, got '%s'", request.Type)
	}
	if len(request.Tags) != 0 {
		t.Errorf("Expected empty tags after reset, got %v", request.Tags)
	}
}

func TestTemplate_Basic(t *testing.T) {
	template := NewTemplate("email-task", "email").
		SetPriority(task.PriorityHigh).
		SetTimeout(30 * time.Second).
		SetRetryPolicy(3, 5*time.Second).
		SetTags("email", "notification")

	client := createMockClient()
	builder := template.CreateBuilder(client)

	request := builder.GetRequest()

	// 验证模板设置
	if request.Name != "email-task" {
		t.Errorf("Expected name 'email-task', got '%s'", request.Name)
	}
	if request.Type != "email" {
		t.Errorf("Expected type 'email', got '%s'", request.Type)
	}
	if request.Priority != task.PriorityHigh {
		t.Errorf("Expected priority %v, got %v", task.PriorityHigh, request.Priority)
	}
	if request.TimeoutSeconds == nil || *request.TimeoutSeconds != 30 {
		t.Errorf("Expected timeout 30 seconds, got %v", request.TimeoutSeconds)
	}
	if request.MaxRetries == nil || *request.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %v", request.MaxRetries)
	}
	if len(request.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(request.Tags))
	}
}

func TestQuickBuilder_Email(t *testing.T) {
	client := createMockClient()
	quick := NewQuickBuilder(client)

	builder := quick.Email("test@example.com", "Test Subject", "Test Body")
	request := builder.GetRequest()

	// 验证邮件任务设置
	if request.Name != "send_email" {
		t.Errorf("Expected name 'send_email', got '%s'", request.Name)
	}
	if request.Type != "email" {
		t.Errorf("Expected type 'email', got '%s'", request.Type)
	}

	// 验证payload
	var payload map[string]string
	err := json.Unmarshal(request.Payload, &payload)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payload["to"] != "test@example.com" {
		t.Errorf("Expected to='test@example.com', got '%s'", payload["to"])
	}
	if payload["subject"] != "Test Subject" {
		t.Errorf("Expected subject='Test Subject', got '%s'", payload["subject"])
	}
	if payload["body"] != "Test Body" {
		t.Errorf("Expected body='Test Body', got '%s'", payload["body"])
	}
}

func TestQuickBuilder_Payment(t *testing.T) {
	client := createMockClient()
	quick := NewQuickBuilder(client)

	builder := quick.Payment("order-123", 99.99, "USD")
	request := builder.GetRequest()

	// 验证支付任务设置
	if request.Name != "process_payment" {
		t.Errorf("Expected name 'process_payment', got '%s'", request.Name)
	}
	if request.Type != "payment" {
		t.Errorf("Expected type 'payment', got '%s'", request.Type)
	}
	if request.Priority != task.PriorityHigh {
		t.Errorf("Expected priority %v, got %v", task.PriorityHigh, request.Priority)
	}

	// 验证payload
	var payload map[string]interface{}
	err := json.Unmarshal(request.Payload, &payload)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payload["order_id"] != "order-123" {
		t.Errorf("Expected order_id='order-123', got '%v'", payload["order_id"])
	}
	if payload["amount"].(float64) != 99.99 {
		t.Errorf("Expected amount=99.99, got %v", payload["amount"])
	}
	if payload["currency"] != "USD" {
		t.Errorf("Expected currency='USD', got '%v'", payload["currency"])
	}
}

func TestTaskBuilder_PriorityHelpers(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	// 测试高优先级辅助方法
	highRequest := builder.WithHighPriority().GetRequest()
	if highRequest.Priority != task.PriorityHigh {
		t.Errorf("Expected high priority, got %v", highRequest.Priority)
	}

	// 重置并测试普通优先级
	builder.Reset()
	normalRequest := builder.WithNormalPriority().GetRequest()
	if normalRequest.Priority != task.PriorityNormal {
		t.Errorf("Expected normal priority, got %v", normalRequest.Priority)
	}

	// 重置并测试低优先级
	builder.Reset()
	lowRequest := builder.WithLowPriority().GetRequest()
	if lowRequest.Priority != task.PriorityLow {
		t.Errorf("Expected low priority, got %v", lowRequest.Priority)
	}
}

func TestTaskBuilder_BusinessUniqueID(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	businessID := "business-unique-123"
	request := builder.
		WithName("unique-task").
		WithBusinessUniqueID(businessID).
		GetRequest()

	if request.BusinessUniqueID == nil || *request.BusinessUniqueID != businessID {
		t.Errorf("Expected BusinessUniqueID='%s', got %v", businessID, request.BusinessUniqueID)
	}
}

func TestTaskBuilder_WithExpiration(t *testing.T) {
	client := createMockClient()
	builder := NewTaskBuilder(client)

	expiredAt := time.Now().Add(1 * time.Hour)
	request := builder.
		WithName("expiring-task").
		WithExpiration(expiredAt).
		GetRequest()

	if request.ExpiredAt == nil {
		t.Fatal("Expected ExpiredAt to be set")
	}

	// 验证过期时间（允许1秒误差）
	diff := request.ExpiredAt.Sub(expiredAt)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Expected expiration time %v, got %v", expiredAt, *request.ExpiredAt)
	}
}

func BenchmarkTaskBuilder_Chaining(b *testing.B) {
	client := createMockClient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewTaskBuilder(client)
		builder.
			WithName("benchmark-task").
			WithType("benchmark").
			WithPriority(task.PriorityNormal).
			WithTimeout(30*time.Second).
			WithRetryPolicy(3, 5*time.Second).
			WithTag("benchmark").
			WithDescription("Benchmark test task")
	}
}

func BenchmarkTaskBuilder_Clone(b *testing.B) {
	client := createMockClient()
	builder := NewTaskBuilder(client).
		WithName("template-task").
		WithType("template").
		WithPriority(task.PriorityNormal).
		WithTimeout(30*time.Second).
		WithTags("tag1", "tag2", "tag3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.Clone()
	}
}