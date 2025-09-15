package callback

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

)

// testHandler 测试用的回调处理器
type testHandler struct {
	createdCalled   bool
	startedCalled   bool
	completedCalled bool
	failedCalled    bool
	lastEvent       *CallbackEvent
	returnError     error
}

func (h *testHandler) HandleTaskCreated(event *CallbackEvent) error {
	h.createdCalled = true
	h.lastEvent = event
	return h.returnError
}

func (h *testHandler) HandleTaskStarted(event *CallbackEvent) error {
	h.startedCalled = true
	h.lastEvent = event
	return h.returnError
}

func (h *testHandler) HandleTaskCompleted(event *CallbackEvent) error {
	h.completedCalled = true
	h.lastEvent = event
	return h.returnError
}

func (h *testHandler) HandleTaskFailed(event *CallbackEvent) error {
	h.failedCalled = true
	h.lastEvent = event
	return h.returnError
}

// testMiddleware 测试用的中间件
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

func TestNewServer(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.apiSecret != "test-secret" {
		t.Errorf("Expected apiSecret 'test-secret', got %s", server.apiSecret)
	}

	if server.handler != handler {
		t.Error("Handler not set correctly")
	}

	// 检查默认选项
	opts := server.GetOptions()
	if opts.WebhookPath != "/webhook" {
		t.Errorf("Expected webhook path '/webhook', got %s", opts.WebhookPath)
	}

	if opts.HealthPath != "/health" {
		t.Errorf("Expected health path '/health', got %s", opts.HealthPath)
	}

	if !opts.EnableSignatureValidation {
		t.Error("Expected signature validation to be enabled by default")
	}
}

func TestServer_WithOptions(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler,
		WithWebhookPath("/custom-webhook"),
		WithHealthPath("/custom-health"),
		WithSignatureValidation(false),
		WithTimestampTolerance(600),
		WithMaxRequestBodySize(2048),
		WithRequestTimeout(60*time.Second),
	)

	opts := server.GetOptions()
	if opts.WebhookPath != "/custom-webhook" {
		t.Errorf("Expected webhook path '/custom-webhook', got %s", opts.WebhookPath)
	}

	if opts.HealthPath != "/custom-health" {
		t.Errorf("Expected health path '/custom-health', got %s", opts.HealthPath)
	}

	if opts.EnableSignatureValidation {
		t.Error("Expected signature validation to be disabled")
	}

	if opts.TimestampToleranceSeconds != 600 {
		t.Errorf("Expected timestamp tolerance 600, got %d", opts.TimestampToleranceSeconds)
	}

	if opts.MaxRequestBodySize != 2048 {
		t.Errorf("Expected max body size 2048, got %d", opts.MaxRequestBodySize)
	}

	if opts.RequestTimeout != 60*time.Second {
		t.Errorf("Expected request timeout 60s, got %v", opts.RequestTimeout)
	}
}

func TestServer_AddMiddleware(t *testing.T) {
	handler := &testHandler{}
	middleware := &testMiddleware{}
	server := NewServer("test-secret", handler)

	server.AddMiddleware(middleware)

	if len(server.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(server.middlewares))
	}

	if server.middlewares[0] != middleware {
		t.Error("Middleware not added correctly")
	}
}

func TestServer_HandleWebhook_Success(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	// 创建测试事件
	event := &CallbackEvent{
		EventType:  "task.created",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task-123",
			CallbackURL:      "http://example.com/callback",
			Status:           TaskStatusPending,
		},
	}

	// 序列化事件
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal("Failed to marshal event:", err)
	}

	// 创建请求
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	// 添加签名头
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, body)
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	recorder := httptest.NewRecorder()

	// 处理请求
	server.handleWebhook(recorder, req)

	// 检查响应
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	// 检查处理器是否被调用
	if !handler.createdCalled {
		t.Error("Handler HandleTaskCreated not called")
	}

	if handler.lastEvent == nil {
		t.Fatal("Handler lastEvent is nil")
	}

	if handler.lastEvent.EventType != "task.created" {
		t.Errorf("Expected event type 'task.created', got %s", handler.lastEvent.EventType)
	}

	if handler.lastEvent.TaskID != 123 {
		t.Errorf("Expected task ID 123, got %d", handler.lastEvent.TaskID)
	}
}

func TestServer_HandleWebhook_InvalidMethod(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	req := httptest.NewRequest("GET", "/webhook", nil)
	recorder := httptest.NewRecorder()

	server.handleWebhook(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", recorder.Code)
	}
}

func TestServer_HandleWebhook_InvalidSignature(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	// 添加无效签名
	req.Header.Set("X-TaskCenter-Signature", "invalid-signature")
	req.Header.Set("X-TaskCenter-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	recorder := httptest.NewRecorder()
	server.handleWebhook(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	// 确保处理器没有被调用
	if handler.createdCalled {
		t.Error("Handler should not be called with invalid signature")
	}
}

func TestServer_HandleWebhook_WithMiddleware(t *testing.T) {
	handler := &testHandler{}
	middleware := &testMiddleware{}
	server := NewServer("test-secret", handler)
	server.AddMiddleware(middleware)

	// 创建有效的请求
	event := &CallbackEvent{
		EventType:  "task.started",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task-123",
			CallbackURL:      "http://example.com/callback",
			Status:           TaskStatusRunning,
		},
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, body)
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	recorder := httptest.NewRecorder()
	server.handleWebhook(recorder, req)

	// 检查中间件是否被调用
	if !middleware.beforeCalled {
		t.Error("Middleware Before not called")
	}

	if !middleware.afterCalled {
		t.Error("Middleware After not called")
	}

	// 检查处理器是否被调用
	if !handler.startedCalled {
		t.Error("Handler HandleTaskStarted not called")
	}
}

func TestServer_HandleWebhook_MiddlewareError(t *testing.T) {
	handler := &testHandler{}
	middleware := &testMiddleware{
		beforeError: NewValidationError("middleware error"),
	}
	server := NewServer("test-secret", handler)
	server.AddMiddleware(middleware)

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, body)
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	recorder := httptest.NewRecorder()
	server.handleWebhook(recorder, req)

	// 中间件错误应该阻止处理
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}

	// 确保处理器没有被调用
	if handler.createdCalled {
		t.Error("Handler should not be called when middleware fails")
	}
}

func TestServer_HandleHealth(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	server.handleHealth(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal("Failed to parse health response:", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["service"] != "TaskCenter Callback Server" {
		t.Errorf("Expected service name, got %v", response["service"])
	}
}

func TestServer_HandleHealth_InvalidMethod(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	req := httptest.NewRequest("POST", "/health", nil)
	recorder := httptest.NewRecorder()

	server.handleHealth(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", recorder.Code)
	}
}

func TestServer_DisableSignatureValidation(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler, WithSignatureValidation(false))

	// 创建没有签名的请求
	event := &CallbackEvent{
		EventType:  "task.completed",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task-123",
			CallbackURL:      "http://example.com/callback",
			Status:           TaskStatusSucceeded,
		},
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.handleWebhook(recorder, req)

	// 应该成功处理，即使没有签名
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if !handler.completedCalled {
		t.Error("Handler HandleTaskCompleted not called")
	}
}

func TestServer_HandlerError(t *testing.T) {
	handler := &testHandler{
		returnError: NewServerError("handler error"),
	}
	server := NewServer("test-secret", handler)

	event := &CallbackEvent{
		EventType:  "task.failed",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task-123",
			CallbackURL:      "http://example.com/callback",
			Status:           TaskStatusFailed,
		},
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, body)
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	recorder := httptest.NewRecorder()
	server.handleWebhook(recorder, req)

	// 处理器错误应该返回500
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}

	// 确保处理器被调用了
	if !handler.failedCalled {
		t.Error("Handler HandleTaskFailed not called")
	}
}

func TestServer_UnknownEventType(t *testing.T) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	// 创建未知事件类型的事件
	event := map[string]interface{}{
		"event_type":  "task.unknown",
		"event_time":  time.Now().Format(time.RFC3339),
		"task_id":     123,
		"business_id": 456,
		"task": map[string]interface{}{
			"id":                 123,
			"business_unique_id": "test-task-123",
			"callback_url":       "http://example.com/callback",
			"status":             0,
		},
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, body)
	req.Header.Set("X-TaskCenter-Signature", signature)
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)

	recorder := httptest.NewRecorder()
	server.handleWebhook(recorder, req)

	// 未知事件类型应该返回400
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}
}

// 辅助函数

// calculateTestSignature 计算测试签名
func calculateTestSignature(secret, timestamp string, body []byte) string {
	signatureString := timestamp + "." + string(body)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))
	return "sha256=" + signature
}

// 基准测试

func BenchmarkServer_HandleWebhook(b *testing.B) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler)

	event := &CallbackEvent{
		EventType:  "task.created",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task-123",
			CallbackURL:      "http://example.com/callback",
			Status:           TaskStatusPending,
		},
	}

	body, _ := json.Marshal(event)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := calculateTestSignature("test-secret", timestamp, body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		req.Header.Set("X-TaskCenter-Signature", signature)
		req.Header.Set("X-TaskCenter-Timestamp", timestamp)

		recorder := httptest.NewRecorder()
		server.handleWebhook(recorder, req)
	}
}

func BenchmarkServer_HandleWebhook_NoValidation(b *testing.B) {
	handler := &testHandler{}
	server := NewServer("test-secret", handler, WithSignatureValidation(false))

	event := &CallbackEvent{
		EventType:  "task.created",
		EventTime:  time.Now(),
		TaskID:     123,
		BusinessID: 456,
		Task: Task{
			ID:               123,
			BusinessUniqueID: "test-task-123",
			CallbackURL:      "http://example.com/callback",
			Status:           TaskStatusPending,
		},
	}

	body, _ := json.Marshal(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		recorder := httptest.NewRecorder()
		server.handleWebhook(recorder, req)
	}
}