package sdk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 测试用的回调处理器
type testCallbackHandler struct {
	createdCalled   bool
	startedCalled   bool
	completedCalled bool
	failedCalled    bool
	lastEvent       *CallbackEvent
}

func (h *testCallbackHandler) HandleTaskCreated(event *CallbackEvent) error {
	h.createdCalled = true
	h.lastEvent = event
	return nil
}

func (h *testCallbackHandler) HandleTaskStarted(event *CallbackEvent) error {
	h.startedCalled = true
	h.lastEvent = event
	return nil
}

func (h *testCallbackHandler) HandleTaskCompleted(event *CallbackEvent) error {
	h.completedCalled = true
	h.lastEvent = event
	return nil
}

func (h *testCallbackHandler) HandleTaskFailed(event *CallbackEvent) error {
	h.failedCalled = true
	h.lastEvent = event
	return nil
}

// 测试用的中间件
type testMiddleware struct {
	beforeCalled bool
	afterCalled  bool
	beforeError  error
	afterError   error
}

func (m *testMiddleware) Before(w http.ResponseWriter, r *http.Request) error {
	m.beforeCalled = true
	return m.beforeError
}

func (m *testMiddleware) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	m.afterCalled = true
	return m.afterError
}

func TestNewCallbackServer(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	if server == nil {
		t.Fatal("NewCallbackServer returned nil")
	}

	if server.apiSecret != "test-secret" {
		t.Errorf("Expected apiSecret 'test-secret', got %s", server.apiSecret)
	}

	if server.handler != handler {
		t.Error("Handler not set correctly")
	}
}

func TestCallbackServer_WithMiddleware(t *testing.T) {
	handler := &testCallbackHandler{}
	middleware := &testMiddleware{}

	server := NewCallbackServer("test-secret", handler,
		WithCallbackMiddleware(middleware))

	if len(server.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(server.middlewares))
	}

	if server.middlewares[0] != middleware {
		t.Error("Middleware not set correctly")
	}
}

func TestCallbackServer_HealthCheck(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if _, ok := response["time"].(string); !ok {
		t.Error("Expected time field in health response")
	}
}

func TestCallbackServer_MethodNotAllowed(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	req := httptest.NewRequest("GET", "/webhook", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCallbackServer_ValidWebhook(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	// 创建测试事件
	event := &CallbackEvent{
		EventType:  "task.completed",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task",
			Status:           TaskStatusSucceeded,
		},
	}

	// 序列化事件
	eventData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// 创建签名
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, eventData)

	// 创建请求
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !handler.completedCalled {
		t.Error("HandleTaskCompleted was not called")
	}

	if handler.lastEvent.TaskID != 123 {
		t.Errorf("Expected TaskID 123, got %d", handler.lastEvent.TaskID)
	}
}

func TestCallbackServer_InvalidSignature(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	event := &CallbackEvent{
		EventType: "task.completed",
		TaskID:    123,
	}

	eventData, _ := json.Marshal(event)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskCenter-Signature", "invalid-signature")
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	if handler.completedCalled {
		t.Error("HandleTaskCompleted should not have been called")
	}
}

func TestCallbackServer_MissingHeaders(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	event := &CallbackEvent{EventType: "task.completed"}
	eventData, _ := json.Marshal(event)

	tests := []struct {
		name        string
		setupHeaders func(*http.Request)
	}{
		{
			name: "missing signature header",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-TaskCenter-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
			},
		},
		{
			name: "missing timestamp header",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-TaskCenter-Signature", "some-signature")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
			req.Header.Set("Content-Type", "application/json")
			tt.setupHeaders(req)

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", w.Code)
			}
		})
	}
}

func TestCallbackServer_ExpiredTimestamp(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	event := &CallbackEvent{EventType: "task.completed"}
	eventData, _ := json.Marshal(event)

	// 使用过期的时间戳（超过5分钟）
	expiredTimestamp := strconv.FormatInt(time.Now().Unix()-400, 10)
	signature := calculateTestSignature("test-secret", expiredTimestamp, eventData)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", expiredTimestamp)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCallbackServer_InvalidJSON(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	invalidJSON := []byte(`{"invalid": json}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, invalidJSON)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCallbackServer_UnknownEventType(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	event := &CallbackEvent{
		EventType: "task.unknown",
		TaskID:    123,
	}

	eventData, _ := json.Marshal(event)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, eventData)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCallbackServer_EventTypes(t *testing.T) {
	handler := &testCallbackHandler{}
	server := NewCallbackServer("test-secret", handler)

	tests := []struct {
		eventType     string
		checkCalled   func() bool
		resetHandler  func()
	}{
		{
			eventType:   "task.created",
			checkCalled: func() bool { return handler.createdCalled },
			resetHandler: func() { handler.createdCalled = false },
		},
		{
			eventType:   "task.started",
			checkCalled: func() bool { return handler.startedCalled },
			resetHandler: func() { handler.startedCalled = false },
		},
		{
			eventType:   "task.completed",
			checkCalled: func() bool { return handler.completedCalled },
			resetHandler: func() { handler.completedCalled = false },
		},
		{
			eventType:   "task.failed",
			checkCalled: func() bool { return handler.failedCalled },
			resetHandler: func() { handler.failedCalled = false },
		},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			tt.resetHandler()

			event := &CallbackEvent{
				EventType: tt.eventType,
				TaskID:    123,
			}

			eventData, _ := json.Marshal(event)
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			signature := calculateTestSignature("test-secret", timestamp, eventData)

			req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-TaskCenter-Signature", signature)
			req.Header.Set("X-TaskCenter-Timestamp", timestamp)

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if !tt.checkCalled() {
				t.Errorf("Handler for %s was not called", tt.eventType)
			}
		})
	}
}

func TestCallbackServer_MiddlewareExecution(t *testing.T) {
	handler := &testCallbackHandler{}
	middleware := &testMiddleware{}

	server := NewCallbackServer("test-secret", handler,
		WithCallbackMiddleware(middleware))

	event := &CallbackEvent{
		EventType: "task.completed",
		TaskID:    123,
	}

	eventData, _ := json.Marshal(event)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, eventData)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !middleware.beforeCalled {
		t.Error("Before middleware was not called")
	}

	if !middleware.afterCalled {
		t.Error("After middleware was not called")
	}

	if !handler.completedCalled {
		t.Error("Handler was not called")
	}
}

func TestCallbackServer_MiddlewareBeforeError(t *testing.T) {
	handler := &testCallbackHandler{}
	middleware := &testMiddleware{
		beforeError: fmt.Errorf("before error"),
	}

	server := NewCallbackServer("test-secret", handler,
		WithCallbackMiddleware(middleware))

	event := &CallbackEvent{EventType: "task.completed"}
	eventData, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(eventData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !middleware.beforeCalled {
		t.Error("Before middleware was not called")
	}

	if middleware.afterCalled {
		t.Error("After middleware should not have been called")
	}

	if handler.completedCalled {
		t.Error("Handler should not have been called")
	}
}

func TestDefaultCallbackHandler(t *testing.T) {
	var calledEvents []string

	handler := &DefaultCallbackHandler{
		OnTaskCreated: func(event *CallbackEvent) error {
			calledEvents = append(calledEvents, "created")
			return nil
		},
		OnTaskStarted: func(event *CallbackEvent) error {
			calledEvents = append(calledEvents, "started")
			return nil
		},
		OnTaskCompleted: func(event *CallbackEvent) error {
			calledEvents = append(calledEvents, "completed")
			return nil
		},
		OnTaskFailed: func(event *CallbackEvent) error {
			calledEvents = append(calledEvents, "failed")
			return nil
		},
	}

	event := &CallbackEvent{TaskID: 123}

	// 测试所有事件类型
	handler.HandleTaskCreated(event)
	handler.HandleTaskStarted(event)
	handler.HandleTaskCompleted(event)
	handler.HandleTaskFailed(event)

	expected := []string{"created", "started", "completed", "failed"}
	if len(calledEvents) != len(expected) {
		t.Errorf("Expected %d events, got %d", len(expected), len(calledEvents))
	}

	for i, expected := range expected {
		if i >= len(calledEvents) || calledEvents[i] != expected {
			t.Errorf("Expected event %s at index %d, got %v", expected, i, calledEvents)
		}
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var logMessages []string
	var logLevels []string

	middleware := &LoggingMiddleware{
		Logger: func(level, message string, fields map[string]interface{}) {
			logLevels = append(logLevels, level)
			logMessages = append(logMessages, message)
		},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "127.0.0.1:12345"

	w := httptest.NewRecorder()

	// 测试Before方法
	err := middleware.Before(w, req)
	if err != nil {
		t.Errorf("Before should not return error, got %v", err)
	}

	// 测试After方法
	event := &CallbackEvent{
		EventType: "task.completed",
		TaskID:    123,
		EventTime: time.Now(),
	}

	err = middleware.After(w, req, event)
	if err != nil {
		t.Errorf("After should not return error, got %v", err)
	}

	// 验证日志调用
	if len(logMessages) != 2 {
		t.Errorf("Expected 2 log messages, got %d", len(logMessages))
	}

	if len(logLevels) != 2 {
		t.Errorf("Expected 2 log levels, got %d", len(logLevels))
	}

	for _, level := range logLevels {
		if level != "info" {
			t.Errorf("Expected log level 'info', got %s", level)
		}
	}
}

// 计算测试签名的辅助函数
func calculateTestSignature(secret, timestamp string, body []byte) string {
	signatureString := timestamp + "." + string(body)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))
	return "sha256=" + signature
}

func TestSignatureVerification(t *testing.T) {
	server := &CallbackServer{apiSecret: "test-secret"}

	body := []byte(`{"test": "data"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// 测试有效签名
	validSignature := calculateTestSignature("test-secret", timestamp, body)

	req := httptest.NewRequest("POST", "/webhook", nil)
	req.Header.Set("X-TaskCenter-Signature", validSignature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	if !server.verifySignature(req, body) {
		t.Error("Valid signature should pass verification")
	}

	// 测试无效签名
	req.Header.Set("X-TaskCenter-Signature", "invalid-signature")
	if server.verifySignature(req, body) {
		t.Error("Invalid signature should fail verification")
	}

	// 测试错误的密钥
	wrongSecretSignature := calculateTestSignature("wrong-secret", timestamp, body)
	req.Header.Set("X-TaskCenter-Signature", wrongSecretSignature)
	if server.verifySignature(req, body) {
		t.Error("Signature with wrong secret should fail verification")
	}
}

func TestAbsFunction(t *testing.T) {
	tests := []struct {
		input    int64
		expected int64
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}