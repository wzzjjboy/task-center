package callback

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

)

func TestNewValidator(t *testing.T) {
	validator := NewValidator("test-secret")

	if validator.apiSecret != "test-secret" {
		t.Errorf("Expected apiSecret 'test-secret', got %s", validator.apiSecret)
	}

	if validator.timestampTolerance != 300 {
		t.Errorf("Expected timestamp tolerance 300, got %d", validator.timestampTolerance)
	}

	expectedHeaders := []string{"X-TaskCenter-Signature", "X-TaskCenter-Timestamp"}
	if len(validator.requiredHeaders) != len(expectedHeaders) {
		t.Errorf("Expected %d required headers, got %d", len(expectedHeaders), len(validator.requiredHeaders))
	}
}

func TestNewValidator_WithOptions(t *testing.T) {
	customValidator := &testCustomValidator{}

	validator := NewValidator("test-secret",
		WithValidatorTimestampTolerance(600),
		WithRequiredHeaders("X-Custom-Header"),
		WithCustomValidator(customValidator),
	)

	if validator.timestampTolerance != 600 {
		t.Errorf("Expected timestamp tolerance 600, got %d", validator.timestampTolerance)
	}

	if len(validator.requiredHeaders) != 1 || validator.requiredHeaders[0] != "X-Custom-Header" {
		t.Errorf("Required headers not set correctly: %v", validator.requiredHeaders)
	}

	if len(validator.customValidators) != 1 {
		t.Errorf("Expected 1 custom validator, got %d", len(validator.customValidators))
	}
}

func TestValidator_ValidateSignature_Success(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)
	req.Header.Set("X-TaskCenter-Signature", validator.GenerateSignature(timestamp, body))

	err := validator.ValidateSignature(req, body)
	if err != nil {
		t.Errorf("ValidateSignature failed: %v", err)
	}
}

func TestValidator_ValidateSignature_MissingHeaders(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))

	// 缺少签名头
	err := validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail when signature header is missing")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for missing signature")
	}

	// 添加签名但缺少时间戳
	req.Header.Set("X-TaskCenter-Signature", "test-signature")
	err = validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail when timestamp header is missing")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for missing timestamp")
	}
}

func TestValidator_ValidateSignature_InvalidTimestamp(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))

	// 无效时间戳格式
	req.Header.Set("X-TaskCenter-Signature", "test-signature")
	req.Header.Set("X-TaskCenter-Timestamp", "invalid-timestamp")

	err := validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail with invalid timestamp format")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for invalid timestamp")
	}
}

func TestValidator_ValidateSignature_ExpiredTimestamp(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)

	// 使用过期的时间戳（10分钟前）
	expiredTimestamp := strconv.FormatInt(time.Now().Unix()-600, 10)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", expiredTimestamp)
	req.Header.Set("X-TaskCenter-Signature", validator.GenerateSignature(expiredTimestamp, body))

	err := validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail with expired timestamp")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for expired timestamp")
	}
}

func TestValidator_ValidateSignature_FutureTimestamp(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)

	// 使用未来的时间戳（10分钟后）
	futureTimestamp := strconv.FormatInt(time.Now().Unix()+600, 10)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", futureTimestamp)
	req.Header.Set("X-TaskCenter-Signature", validator.GenerateSignature(futureTimestamp, body))

	err := validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail with future timestamp")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for future timestamp")
	}
}

func TestValidator_ValidateSignature_InvalidSignature(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)
	req.Header.Set("X-TaskCenter-Signature", "sha256=invalid-signature")

	err := validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail with invalid signature")
	}
	if !IsAuthenticationError(err) {
		t.Error("Should return authentication error for invalid signature")
	}
}

func TestValidator_ValidateSignature_CustomValidator(t *testing.T) {
	customValidator := &testCustomValidator{
		shouldFail: true,
		errorToReturn: NewValidationError("custom validation failed"),
	}

	validator := NewValidator("test-secret", WithCustomValidator(customValidator))

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)
	req.Header.Set("X-TaskCenter-Signature", validator.GenerateSignature(timestamp, body))

	err := validator.ValidateSignature(req, body)
	if err == nil {
		t.Error("Should fail when custom validator fails")
	}

	if !customValidator.called {
		t.Error("Custom validator should be called")
	}

	if err.Error() != "custom validation failed" {
		t.Errorf("Expected custom validation error, got: %v", err)
	}
}

func TestValidator_ParseEvent_Success(t *testing.T) {
	validator := NewValidator("test-secret")

	eventData := map[string]interface{}{
		"event_type":  "task.created",
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

	body, _ := json.Marshal(eventData)

	event, err := validator.ParseEvent(body)
	if err != nil {
		t.Errorf("ParseEvent failed: %v", err)
	}

	if event == nil {
		t.Fatal("Parsed event is nil")
	}

	if event.EventType != "task.created" {
		t.Errorf("Expected event type 'task.created', got %s", event.EventType)
	}

	if event.TaskID != 123 {
		t.Errorf("Expected task ID 123, got %d", event.TaskID)
	}

	if event.BusinessID != 456 {
		t.Errorf("Expected business ID 456, got %d", event.BusinessID)
	}
}

func TestValidator_ParseEvent_EmptyBody(t *testing.T) {
	validator := NewValidator("test-secret")

	_, err := validator.ParseEvent([]byte{})
	if err == nil {
		t.Error("Should fail with empty body")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for empty body")
	}
}

func TestValidator_ParseEvent_InvalidJSON(t *testing.T) {
	validator := NewValidator("test-secret")

	_, err := validator.ParseEvent([]byte(`{"invalid json`))
	if err == nil {
		t.Error("Should fail with invalid JSON")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for invalid JSON")
	}
}

func TestValidator_ParseEvent_MissingFields(t *testing.T) {
	validator := NewValidator("test-secret")

	// 缺少event_type
	eventData := map[string]interface{}{
		"task_id":     123,
		"business_id": 456,
	}

	body, _ := json.Marshal(eventData)

	_, err := validator.ParseEvent(body)
	if err == nil {
		t.Error("Should fail with missing event_type")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for missing event_type")
	}
}

func TestValidator_ParseEvent_InvalidEventType(t *testing.T) {
	validator := NewValidator("test-secret")

	eventData := map[string]interface{}{
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

	body, _ := json.Marshal(eventData)

	_, err := validator.ParseEvent(body)
	if err == nil {
		t.Error("Should fail with invalid event type")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for invalid event type")
	}
}

func TestValidator_ParseEvent_InvalidTaskID(t *testing.T) {
	validator := NewValidator("test-secret")

	eventData := map[string]interface{}{
		"event_type":  "task.created",
		"event_time":  time.Now().Format(time.RFC3339),
		"task_id":     0, // 无效的task_id
		"business_id": 456,
		"task": map[string]interface{}{
			"id":                 123,
			"business_unique_id": "test-task-123",
			"callback_url":       "http://example.com/callback",
			"status":             0,
		},
	}

	body, _ := json.Marshal(eventData)

	_, err := validator.ParseEvent(body)
	if err == nil {
		t.Error("Should fail with invalid task_id")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error for invalid task_id")
	}
}

func TestValidator_GenerateSignatureWithTimestamp(t *testing.T) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)

	signature, timestamp := validator.GenerateSignatureWithTimestamp(body)

	if signature == "" {
		t.Error("Generated signature is empty")
	}

	if timestamp == "" {
		t.Error("Generated timestamp is empty")
	}

	if !strings.HasPrefix(signature, "sha256=") {
		t.Error("Signature should have sha256= prefix")
	}

	// 验证生成的签名是否有效
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)
	req.Header.Set("X-TaskCenter-Signature", signature)

	err := validator.ValidateSignature(req, body)
	if err != nil {
		t.Errorf("Generated signature should be valid: %v", err)
	}
}

func TestValidator_SetTimestampTolerance(t *testing.T) {
	validator := NewValidator("test-secret")

	validator.SetTimestampTolerance(1200)

	if validator.GetTimestampTolerance() != 1200 {
		t.Errorf("Expected timestamp tolerance 1200, got %d", validator.GetTimestampTolerance())
	}
}

func TestValidator_AddCustomValidator(t *testing.T) {
	validator := NewValidator("test-secret")

	customValidator := &testCustomValidator{}
	validator.AddCustomValidator(customValidator)

	if len(validator.customValidators) != 1 {
		t.Errorf("Expected 1 custom validator, got %d", len(validator.customValidators))
	}

	if validator.customValidators[0] != customValidator {
		t.Error("Custom validator not added correctly")
	}
}

// 预定义验证器测试

func TestIPWhitelistValidator(t *testing.T) {
	validator := &IPWhitelistValidator{
		AllowedIPs: []string{"192.168.1.1", "10.0.0.1"},
	}

	// 测试允许的IP
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	err := validator.Validate(req, nil)
	if err != nil {
		t.Errorf("Should allow whitelisted IP: %v", err)
	}

	// 测试不允许的IP
	req.RemoteAddr = "192.168.1.2:12345"

	err = validator.Validate(req, nil)
	if err == nil {
		t.Error("Should reject non-whitelisted IP")
	}
	if !IsAuthorizationError(err) {
		t.Error("Should return authorization error")
	}

	// 测试空白名单（应该允许所有IP）
	validator.AllowedIPs = []string{}
	err = validator.Validate(req, nil)
	if err != nil {
		t.Errorf("Should allow all IPs when whitelist is empty: %v", err)
	}
}

func TestUserAgentValidator(t *testing.T) {
	validator := &UserAgentValidator{
		RequiredUserAgent: "TaskCenter",
		AllowedUserAgents: []string{"TaskCenter", "WebhookClient"},
	}

	req := httptest.NewRequest("POST", "/webhook", nil)

	// 测试包含必需User-Agent的情况
	req.Header.Set("User-Agent", "TaskCenter/1.0")

	err := validator.Validate(req, nil)
	if err != nil {
		t.Errorf("Should allow User-Agent containing required string: %v", err)
	}

	// 测试不包含必需User-Agent的情况
	req.Header.Set("User-Agent", "BadClient/1.0")

	err = validator.Validate(req, nil)
	if err == nil {
		t.Error("Should reject User-Agent not containing required string")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error")
	}

	// 测试允许列表中的User-Agent
	validator.RequiredUserAgent = ""
	req.Header.Set("User-Agent", "WebhookClient/2.0")

	err = validator.Validate(req, nil)
	if err != nil {
		t.Errorf("Should allow User-Agent in allowed list: %v", err)
	}

	// 测试不在允许列表中的User-Agent
	req.Header.Set("User-Agent", "UnknownClient/1.0")

	err = validator.Validate(req, nil)
	if err == nil {
		t.Error("Should reject User-Agent not in allowed list")
	}
	if !IsAuthorizationError(err) {
		t.Error("Should return authorization error")
	}
}

func TestContentTypeValidator(t *testing.T) {
	validator := &ContentTypeValidator{
		RequiredContentType: "application/json",
	}

	req := httptest.NewRequest("POST", "/webhook", nil)

	// 测试正确的Content-Type
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	err := validator.Validate(req, nil)
	if err != nil {
		t.Errorf("Should allow correct Content-Type: %v", err)
	}

	// 测试错误的Content-Type
	req.Header.Set("Content-Type", "text/plain")

	err = validator.Validate(req, nil)
	if err == nil {
		t.Error("Should reject incorrect Content-Type")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error")
	}
}

func TestEventTypeValidator(t *testing.T) {
	validator := &EventTypeValidator{
		AllowedEventTypes: []string{"task.created", "task.completed"},
	}

	// 测试允许的事件类型
	eventData := map[string]interface{}{
		"event_type": "task.created",
		"task_id":    123,
	}
	body, _ := json.Marshal(eventData)

	err := validator.Validate(nil, body)
	if err != nil {
		t.Errorf("Should allow whitelisted event type: %v", err)
	}

	// 测试不允许的事件类型
	eventData["event_type"] = "task.started"
	body, _ = json.Marshal(eventData)

	err = validator.Validate(nil, body)
	if err == nil {
		t.Error("Should reject non-whitelisted event type")
	}
	if !IsValidationError(err) {
		t.Error("Should return validation error")
	}

	// 测试空白名单（应该允许所有事件类型）
	validator.AllowedEventTypes = []string{}
	err = validator.Validate(nil, body)
	if err != nil {
		t.Errorf("Should allow all event types when whitelist is empty: %v", err)
	}
}

func TestBusinessValidator(t *testing.T) {
	validator := &BusinessValidator{
		ValidateBusinessID: func(businessID int64) error {
			if businessID != 456 {
				return NewValidationError("invalid business ID")
			}
			return nil
		},
		ValidateTaskID: func(taskID int64) error {
			if taskID <= 0 {
				return NewValidationError("invalid task ID")
			}
			return nil
		},
	}

	// 测试有效的事件
	eventData := map[string]interface{}{
		"event_type":  "task.created",
		"task_id":     123,
		"business_id": 456,
	}
	body, _ := json.Marshal(eventData)

	err := validator.Validate(nil, body)
	if err != nil {
		t.Errorf("Should allow valid event: %v", err)
	}

	// 测试无效的business_id
	eventData["business_id"] = 999
	body, _ = json.Marshal(eventData)

	err = validator.Validate(nil, body)
	if err == nil {
		t.Error("Should reject invalid business ID")
	}

	// 测试无效的task_id
	eventData["business_id"] = 456
	eventData["task_id"] = 0
	body, _ = json.Marshal(eventData)

	err = validator.Validate(nil, body)
	if err == nil {
		t.Error("Should reject invalid task ID")
	}
}

func TestIsValidEventType(t *testing.T) {
	validTypes := []string{
		"task.created",
		"task.started",
		"task.completed",
		"task.failed",
	}

	for _, eventType := range validTypes {
		if !IsValidEventType(eventType) {
			t.Errorf("Event type %s should be valid", eventType)
		}
	}

	invalidTypes := []string{
		"task.unknown",
		"user.created",
		"invalid",
		"",
	}

	for _, eventType := range invalidTypes {
		if IsValidEventType(eventType) {
			t.Errorf("Event type %s should be invalid", eventType)
		}
	}
}

// 辅助类型

type testCustomValidator struct {
	called        bool
	shouldFail    bool
	errorToReturn error
}

func (v *testCustomValidator) Validate(r *http.Request, body []byte) error {
	v.called = true
	if v.shouldFail {
		return v.errorToReturn
	}
	return nil
}

// 基准测试

func BenchmarkValidator_ValidateSignature(b *testing.B) {
	validator := NewValidator("test-secret")

	body := []byte(`{"event_type":"task.created","task_id":123}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := validator.GenerateSignature(timestamp, body)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-TaskCenter-Timestamp", timestamp)
	req.Header.Set("X-TaskCenter-Signature", signature)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateSignature(req, body)
	}
}

func BenchmarkValidator_ParseEvent(b *testing.B) {
	validator := NewValidator("test-secret")

	eventData := map[string]interface{}{
		"event_type":  "task.created",
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

	body, _ := json.Marshal(eventData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ParseEvent(body)
	}
}