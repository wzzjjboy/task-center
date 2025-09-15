package callback

import (
	"encoding/json"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMiddlewareChain(t *testing.T) {
	middleware1 := &testMiddleware{}
	middleware2 := &testMiddleware{}

	chain := NewMiddlewareChain(middleware1, middleware2)

	if len(chain.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(chain.middlewares))
	}

	// 测试添加中间件
	middleware3 := &testMiddleware{}
	chain.Add(middleware3)

	if len(chain.middlewares) != 3 {
		t.Errorf("Expected 3 middlewares after Add, got %d", len(chain.middlewares))
	}
}

func TestMiddlewareChain_ExecuteBefore(t *testing.T) {
	middleware1 := &testMiddleware{}
	middleware2 := &testMiddleware{}

	chain := NewMiddlewareChain(middleware1, middleware2)

	req := httptest.NewRequest("POST", "/webhook", nil)
	recorder := httptest.NewRecorder()

	err := chain.ExecuteBefore(recorder, req)

	if err != nil {
		t.Errorf("ExecuteBefore failed: %v", err)
	}

	if !middleware1.beforeCalled {
		t.Error("Middleware1 Before not called")
	}

	if !middleware2.beforeCalled {
		t.Error("Middleware2 Before not called")
	}
}

func TestMiddlewareChain_ExecuteBefore_Error(t *testing.T) {
	middleware1 := &testMiddleware{}
	middleware2 := &testMiddleware{
		beforeError: NewValidationError("middleware2 error"),
	}
	middleware3 := &testMiddleware{}

	chain := NewMiddlewareChain(middleware1, middleware2, middleware3)

	req := httptest.NewRequest("POST", "/webhook", nil)
	recorder := httptest.NewRecorder()

	err := chain.ExecuteBefore(recorder, req)

	if err == nil {
		t.Error("Expected error from ExecuteBefore")
	}

	if !middleware1.beforeCalled {
		t.Error("Middleware1 Before not called")
	}

	if !middleware2.beforeCalled {
		t.Error("Middleware2 Before not called")
	}

	// 第三个中间件不应该被调用，因为第二个出错了
	if middleware3.beforeCalled {
		t.Error("Middleware3 Before should not be called after error")
	}
}

func TestMiddlewareChain_ExecuteAfter(t *testing.T) {
	middleware1 := &testMiddleware{}
	middleware2 := &testMiddleware{}

	chain := NewMiddlewareChain(middleware1, middleware2)

	req := httptest.NewRequest("POST", "/webhook", nil)
	recorder := httptest.NewRecorder()

	event := &CallbackEvent{
		EventType:  "task.created",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
	}

	err := chain.ExecuteAfter(recorder, req, event)

	if err != nil {
		t.Errorf("ExecuteAfter failed: %v", err)
	}

	if !middleware1.afterCalled {
		t.Error("Middleware1 After not called")
	}

	if !middleware2.afterCalled {
		t.Error("Middleware2 After not called")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	logger := log.New(os.Stdout, "", 0)

	middleware := NewLoggingMiddleware(
		WithLogger(logger),
		WithLogHeaders(true),
		WithLogBody(true),
		WithLogResponseTime(true),
	)

	if middleware.Logger == nil {
		t.Error("Logger not set")
	}

	if !middleware.LogHeaders {
		t.Error("LogHeaders not enabled")
	}

	if !middleware.LogBody {
		t.Error("LogBody not enabled")
	}

	if !middleware.LogResponseTime {
		t.Error("LogResponseTime not enabled")
	}
}

func TestLoggingMiddleware_Before(t *testing.T) {
	var loggedData map[string]interface{}

	// 创建自定义日志记录器
	customLogger := log.New(&testLogWriter{
		writeFunc: func(data []byte) (int, error) {
			// 解析日志数据
			line := string(data)
			if strings.Contains(line, "Webhook request received") {
				// 提取JSON部分
				start := strings.Index(line, "{")
				if start != -1 {
					json.Unmarshal([]byte(line[start:]), &loggedData)
				}
			}
			return len(data), nil
		},
	}, "", 0)

	middleware := NewLoggingMiddleware(
		WithLogger(customLogger),
		WithLogHeaders(true),
		WithLogResponseTime(true),
	)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Authorization", "Bearer secret")
	req.RemoteAddr = "192.168.1.1:12345"

	recorder := httptest.NewRecorder()

	err := middleware.Before(recorder, req)

	if err != nil {
		t.Errorf("Before failed: %v", err)
	}

	if loggedData == nil {
		t.Fatal("No log data captured")
	}

	if loggedData["method"] != "POST" {
		t.Errorf("Expected method 'POST', got %v", loggedData["method"])
	}

	if loggedData["user_agent"] != "TestAgent/1.0" {
		t.Errorf("Expected user_agent 'TestAgent/1.0', got %v", loggedData["user_agent"])
	}

	// Authorization 头应该被排除
	if headers, ok := loggedData["headers"].(map[string]interface{}); ok {
		if _, exists := headers["Authorization"]; exists {
			t.Error("Authorization header should be excluded from logs")
		}
	}
}

func TestLoggingMiddleware_After(t *testing.T) {
	var loggedData map[string]interface{}

	customLogger := log.New(&testLogWriter{
		writeFunc: func(data []byte) (int, error) {
			line := string(data)
			if strings.Contains(line, "Webhook event processed") {
				start := strings.Index(line, "{")
				if start != -1 {
					json.Unmarshal([]byte(line[start:]), &loggedData)
				}
			}
			return len(data), nil
		},
	}, "", 0)

	middleware := NewLoggingMiddleware(
		WithLogger(customLogger),
		WithLogResponseTime(true),
	)

	req := httptest.NewRequest("POST", "/webhook", nil)
	recorder := httptest.NewRecorder()

	event := &CallbackEvent{
		EventType:  "task.completed",
		EventTime:  time.Now(),
		TaskID:     789,
		BusinessID: 123,
	}

	err := middleware.After(recorder, req, event)

	if err != nil {
		t.Errorf("After failed: %v", err)
	}

	if loggedData == nil {
		t.Fatal("No log data captured")
	}

	if loggedData["event_type"] != "task.completed" {
		t.Errorf("Expected event_type 'task.completed', got %v", loggedData["event_type"])
	}

	if loggedData["task_id"] != float64(789) { // JSON 数字默认解析为 float64
		t.Errorf("Expected task_id 789, got %v", loggedData["task_id"])
	}
}

func TestMetricsMiddleware(t *testing.T) {
	middleware := NewMetricsMiddleware()

	if middleware.requestCount == nil {
		t.Error("requestCount not initialized")
	}

	if middleware.responseTime == nil {
		t.Error("responseTime not initialized")
	}

	if middleware.errorCount == nil {
		t.Error("errorCount not initialized")
	}
}

func TestMetricsMiddleware_Before_After(t *testing.T) {
	var lastEventType string
	var lastDuration time.Duration

	middleware := NewMetricsMiddleware()
	middleware.OnRequestStart = func(eventType string) {
		lastEventType = eventType
	}
	middleware.OnRequestComplete = func(eventType string, duration time.Duration) {
		lastEventType = eventType
		lastDuration = duration
	}

	req := httptest.NewRequest("POST", "/webhook", nil)
	recorder := httptest.NewRecorder()

	// Before
	err := middleware.Before(recorder, req)
	if err != nil {
		t.Errorf("Before failed: %v", err)
	}

	if lastEventType != "webhook" {
		t.Errorf("Expected event type 'webhook', got %s", lastEventType)
	}

	// 模拟一些处理时间
	time.Sleep(10 * time.Millisecond)

	// After
	event := &CallbackEvent{
		EventType:  "task.started",
		EventTime:  time.Now(),
		TaskID:     456,
		BusinessID: 789,
	}

	err = middleware.After(recorder, req, event)
	if err != nil {
		t.Errorf("After failed: %v", err)
	}

	if lastEventType != "task.started" {
		t.Errorf("Expected event type 'task.started', got %s", lastEventType)
	}

	if lastDuration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", lastDuration)
	}

	// 检查指标
	metrics := middleware.GetMetrics()

	requestCount := metrics["request_count"].(map[string]int64)
	if requestCount["task.started"] != 1 {
		t.Errorf("Expected request count 1 for task.started, got %d", requestCount["task.started"])
	}

	if requestCount["total"] != 1 {
		t.Errorf("Expected total request count 1, got %d", requestCount["total"])
	}

	avgResponseTime := metrics["avg_response_time_ms"].(map[string]float64)
	if avgResponseTime["task.started"] < 10 {
		t.Errorf("Expected avg response time >= 10ms, got %v", avgResponseTime["task.started"])
	}
}

func TestMetricsMiddleware_RecordError(t *testing.T) {
	var lastEventType, lastErrorType string

	middleware := NewMetricsMiddleware()
	middleware.OnRequestError = func(eventType, errorType string) {
		lastEventType = eventType
		lastErrorType = errorType
	}

	middleware.RecordError("task.failed", "validation_error")

	if lastEventType != "task.failed" {
		t.Errorf("Expected event type 'task.failed', got %s", lastEventType)
	}

	if lastErrorType != "validation_error" {
		t.Errorf("Expected error type 'validation_error', got %s", lastErrorType)
	}

	metrics := middleware.GetMetrics()
	errorCount := metrics["error_count"].(map[string]int64)

	if errorCount["task.failed:validation_error"] != 1 {
		t.Errorf("Expected error count 1, got %d", errorCount["task.failed:validation_error"])
	}

	if errorCount["total_errors"] != 1 {
		t.Errorf("Expected total error count 1, got %d", errorCount["total_errors"])
	}
}

func TestSecurityMiddleware(t *testing.T) {
	middleware := NewSecurityMiddleware()

	if middleware.requestCounts == nil {
		t.Error("requestCounts not initialized")
	}
}

func TestSecurityMiddleware_IPWhitelist(t *testing.T) {
	middleware := NewSecurityMiddleware()
	middleware.AllowedIPs = []string{"192.168.1.1", "10.0.0.1"}

	// 测试允许的IP
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	err := middleware.Before(httptest.NewRecorder(), req)
	if err != nil {
		t.Errorf("Should allow whitelisted IP: %v", err)
	}

	// 测试不允许的IP
	req.RemoteAddr = "192.168.1.2:12345"

	err = middleware.Before(httptest.NewRecorder(), req)
	if err == nil {
		t.Error("Should reject non-whitelisted IP")
	}

	if !IsAuthorizationError(err) {
		t.Error("Should return authorization error for non-whitelisted IP")
	}
}

func TestSecurityMiddleware_UserAgent(t *testing.T) {
	middleware := NewSecurityMiddleware()
	middleware.AllowedUserAgents = []string{"TaskCenter", "WebhookClient"}

	// 测试允许的User-Agent
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.Header.Set("User-Agent", "TaskCenter/1.0")

	err := middleware.Before(httptest.NewRecorder(), req)
	if err != nil {
		t.Errorf("Should allow whitelisted User-Agent: %v", err)
	}

	// 测试不允许的User-Agent
	req.Header.Set("User-Agent", "BadClient/1.0")

	err = middleware.Before(httptest.NewRecorder(), req)
	if err == nil {
		t.Error("Should reject non-whitelisted User-Agent")
	}

	if !IsAuthorizationError(err) {
		t.Error("Should return authorization error for non-whitelisted User-Agent")
	}
}

func TestSecurityMiddleware_RequiredHeaders(t *testing.T) {
	middleware := NewSecurityMiddleware()
	middleware.RequiredHeaders = map[string]string{
		"X-API-Key":     "secret-key",
		"X-Client-Type": "webhook",
	}

	// 测试有效的请求头
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.Header.Set("X-API-Key", "secret-key")
	req.Header.Set("X-Client-Type", "webhook")

	err := middleware.Before(httptest.NewRecorder(), req)
	if err != nil {
		t.Errorf("Should allow request with required headers: %v", err)
	}

	// 测试缺少请求头
	req.Header.Del("X-API-Key")

	err = middleware.Before(httptest.NewRecorder(), req)
	if err == nil {
		t.Error("Should reject request with missing required header")
	}

	if !IsValidationError(err) {
		t.Error("Should return validation error for missing required header")
	}

	// 测试错误的请求头值
	req.Header.Set("X-API-Key", "wrong-key")

	err = middleware.Before(httptest.NewRecorder(), req)
	if err == nil {
		t.Error("Should reject request with wrong header value")
	}
}

func TestSecurityMiddleware_RateLimit(t *testing.T) {
	middleware := NewSecurityMiddleware()
	middleware.RateLimitPerMinute = 2

	req := httptest.NewRequest("POST", "/webhook", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	recorder := httptest.NewRecorder()

	// 前两个请求应该成功
	for i := 0; i < 2; i++ {
		err := middleware.Before(recorder, req)
		if err != nil {
			t.Errorf("Request %d should succeed: %v", i+1, err)
		}
	}

	// 第三个请求应该被限制
	err := middleware.Before(recorder, req)
	if err == nil {
		t.Error("Third request should be rate limited")
	}

	if !IsRateLimitError(err) {
		t.Error("Should return rate limit error")
	}
}

func TestDefaultHandler(t *testing.T) {
	var lastEvent *CallbackEvent

	handler := &DefaultHandler{
		OnTaskCreated: func(event *CallbackEvent) error {
			lastEvent = event
			return nil
		},
	}

	event := &CallbackEvent{
		EventType:  "task.created",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
	}

	err := handler.HandleTaskCreated(event)
	if err != nil {
		t.Errorf("HandleTaskCreated failed: %v", err)
	}

	if lastEvent == nil {
		t.Fatal("Event not passed to handler")
	}

	if lastEvent.TaskID != 123 {
		t.Errorf("Expected task ID 123, got %d", lastEvent.TaskID)
	}

	// 测试未设置的处理器
	err = handler.HandleTaskStarted(event)
	if err != nil {
		t.Errorf("HandleTaskStarted should not fail when handler not set: %v", err)
	}
}

// 辅助类型和函数

type testLogWriter struct {
	writeFunc func([]byte) (int, error)
}

func (w *testLogWriter) Write(data []byte) (int, error) {
	if w.writeFunc != nil {
		return w.writeFunc(data)
	}
	return len(data), nil
}

// 基准测试

func BenchmarkLoggingMiddleware_Before(b *testing.B) {
	middleware := NewLoggingMiddleware()

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("User-Agent", "TestAgent/1.0")
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.Before(recorder, req)
	}
}

func BenchmarkMetricsMiddleware_BeforeAfter(b *testing.B) {
	middleware := NewMetricsMiddleware()

	req := httptest.NewRequest("POST", "/webhook", nil)
	recorder := httptest.NewRecorder()

	event := &CallbackEvent{
		EventType:  "task.created",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.Before(recorder, req)
		middleware.After(recorder, req, event)
	}
}

func BenchmarkSecurityMiddleware_Before(b *testing.B) {
	middleware := NewSecurityMiddleware()

	req := httptest.NewRequest("POST", "/webhook", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.Before(recorder, req)
	}
}