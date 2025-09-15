package callback

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

)

// Validator 回调验证器
type Validator struct {
	apiSecret         string
	timestampTolerance int64
	requiredHeaders   []string
	customValidators  []CustomValidator
}

// CustomValidator 自定义验证器接口
type CustomValidator interface {
	Validate(r *http.Request, body []byte) error
}

// ValidatorOption 验证器配置选项
type ValidatorOption func(*Validator)

// WithTimestampTolerance 设置时间戳容差（秒）
func WithValidatorTimestampTolerance(seconds int64) ValidatorOption {
	return func(v *Validator) {
		v.timestampTolerance = seconds
	}
}

// WithRequiredHeaders 设置必需的请求头
func WithRequiredHeaders(headers ...string) ValidatorOption {
	return func(v *Validator) {
		v.requiredHeaders = headers
	}
}

// WithCustomValidator 添加自定义验证器
func WithCustomValidator(validator CustomValidator) ValidatorOption {
	return func(v *Validator) {
		v.customValidators = append(v.customValidators, validator)
	}
}

// NewValidator 创建新的验证器
func NewValidator(apiSecret string, opts ...ValidatorOption) *Validator {
	validator := &Validator{
		apiSecret:         apiSecret,
		timestampTolerance: 300, // 默认5分钟容差
		requiredHeaders:   []string{"X-TaskCenter-Signature", "X-TaskCenter-Timestamp"},
	}

	for _, opt := range opts {
		opt(validator)
	}

	return validator
}

// ValidateSignature 验证请求签名
func (v *Validator) ValidateSignature(r *http.Request, body []byte) error {
	// 检查必需的请求头
	for _, header := range v.requiredHeaders {
		if r.Header.Get(header) == "" {
			return NewValidationError(fmt.Sprintf("Missing required header: %s", header))
		}
	}

	// 获取签名头
	signature := r.Header.Get("X-TaskCenter-Signature")
	if signature == "" {
		return NewValidationError("Missing signature header")
	}

	// 获取时间戳头
	timestampStr := r.Header.Get("X-TaskCenter-Timestamp")
	if timestampStr == "" {
		return NewValidationError("Missing timestamp header")
	}

	// 验证时间戳格式
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return NewValidationError("Invalid timestamp format")
	}

	// 验证时间戳是否在允许范围内（防止重放攻击）
	if err := v.validateTimestamp(timestamp); err != nil {
		return err
	}

	// 计算期望的签名
	expectedSignature := v.calculateSignature(timestampStr, body)

	// 比较签名（使用常量时间比较防止时序攻击）
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return NewAuthenticationError("Invalid signature")
	}

	// 执行自定义验证器
	for _, customValidator := range v.customValidators {
		if err := customValidator.Validate(r, body); err != nil {
			return err
		}
	}

	return nil
}

// ParseEvent 解析回调事件
func (v *Validator) ParseEvent(body []byte) (*CallbackEvent, error) {
	if len(body) == 0 {
		return nil, NewValidationError("Empty request body")
	}

	var event CallbackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, NewValidationError("Invalid JSON format: " + err.Error())
	}

	// 验证事件字段
	if err := v.validateEvent(&event); err != nil {
		return nil, err
	}

	return &event, nil
}

// validateTimestamp 验证时间戳
func (v *Validator) validateTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	diff := now - timestamp

	// 使用绝对值检查时间差
	if diff < 0 {
		diff = -diff
	}

	if diff > v.timestampTolerance {
		return NewValidationError(fmt.Sprintf(
			"Timestamp too old or too far in the future. Difference: %d seconds, tolerance: %d seconds",
			diff, v.timestampTolerance))
	}

	return nil
}

// calculateSignature 计算签名
func (v *Validator) calculateSignature(timestamp string, body []byte) string {
	// 构建签名字符串：timestamp + "." + body
	signatureString := timestamp + "." + string(body)

	// 使用HMAC-SHA256计算签名
	h := hmac.New(sha256.New, []byte(v.apiSecret))
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))

	return "sha256=" + signature
}

// validateEvent 验证事件数据
func (v *Validator) validateEvent(event *CallbackEvent) error {
	// 验证事件类型
	if event.EventType == "" {
		return NewValidationError("Missing event_type field")
	}

	validEventTypes := []string{
		"task.created",
		"task.started",
		"task.completed",
		"task.failed",
	}

	if !validatorContains(validEventTypes, event.EventType) {
		return NewValidationError(fmt.Sprintf(
			"Invalid event_type: %s. Valid types: %s",
			event.EventType, strings.Join(validEventTypes, ", ")))
	}

	// 验证任务ID
	if event.TaskID <= 0 {
		return NewValidationError("Invalid task_id: must be positive integer")
	}

	// 验证业务ID
	if event.BusinessID <= 0 {
		return NewValidationError("Invalid business_id: must be positive integer")
	}

	// 验证事件时间
	if event.EventTime.IsZero() {
		return NewValidationError("Missing or invalid event_time field")
	}

	// 验证任务数据
	if err := v.validateTask(&event.Task); err != nil {
		return err
	}

	return nil
}

// validateTask 验证任务数据
func (v *Validator) validateTask(task *Task) error {
	// 验证任务ID与事件中的ID一致
	// 这里可以根据实际需求添加更多验证

	// 验证业务唯一ID
	if task.BusinessUniqueID == "" {
		return NewValidationError("Task missing business_unique_id")
	}

	// 验证回调URL
	if task.CallbackURL == "" {
		return NewValidationError("Task missing callback_url")
	}

	// 验证任务状态
	validStatuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusSucceeded,
		TaskStatusFailed,
		TaskStatusCancelled,
		TaskStatusExpired,
	}

	validStatus := false
	for _, status := range validStatuses {
		if task.Status == status {
			validStatus = true
			break
		}
	}

	if !validStatus {
		return NewValidationError(fmt.Sprintf("Invalid task status: %d", int(task.Status)))
	}

	return nil
}

// SetTimestampTolerance 设置时间戳容差
func (v *Validator) SetTimestampTolerance(seconds int64) {
	v.timestampTolerance = seconds
}

// GetTimestampTolerance 获取时间戳容差
func (v *Validator) GetTimestampTolerance() int64 {
	return v.timestampTolerance
}

// AddCustomValidator 添加自定义验证器
func (v *Validator) AddCustomValidator(validator CustomValidator) {
	v.customValidators = append(v.customValidators, validator)
}

// GenerateSignature 生成签名（用于测试和客户端）
func (v *Validator) GenerateSignature(timestamp string, body []byte) string {
	return v.calculateSignature(timestamp, body)
}

// GenerateSignatureWithTimestamp 生成带时间戳的签名
func (v *Validator) GenerateSignatureWithTimestamp(body []byte) (signature, timestamp string) {
	timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	signature = v.calculateSignature(timestamp, body)
	return signature, timestamp
}

// 预定义的自定义验证器

// IPWhitelistValidator IP白名单验证器
type IPWhitelistValidator struct {
	AllowedIPs []string
}

// Validate 验证IP地址
func (v *IPWhitelistValidator) Validate(r *http.Request, body []byte) error {
	if len(v.AllowedIPs) == 0 {
		return nil // 没有限制
	}

	clientIP := getRealIP(r)
	for _, allowedIP := range v.AllowedIPs {
		if clientIP == allowedIP {
			return nil
		}
	}

	return NewAuthorizationError(fmt.Sprintf("IP address %s is not allowed", clientIP))
}

// UserAgentValidator User-Agent验证器
type UserAgentValidator struct {
	RequiredUserAgent string
	AllowedUserAgents []string
}

// Validate 验证User-Agent
func (v *UserAgentValidator) Validate(r *http.Request, body []byte) error {
	userAgent := r.Header.Get("User-Agent")

	if v.RequiredUserAgent != "" {
		if !strings.Contains(userAgent, v.RequiredUserAgent) {
			return NewValidationError(fmt.Sprintf("User-Agent must contain: %s", v.RequiredUserAgent))
		}
	}

	if len(v.AllowedUserAgents) > 0 {
		allowed := false
		for _, allowedUA := range v.AllowedUserAgents {
			if strings.Contains(userAgent, allowedUA) {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewAuthorizationError("User-Agent not allowed")
		}
	}

	return nil
}

// ContentTypeValidator Content-Type验证器
type ContentTypeValidator struct {
	RequiredContentType string
}

// Validate 验证Content-Type
func (v *ContentTypeValidator) Validate(r *http.Request, body []byte) error {
	contentType := r.Header.Get("Content-Type")

	if v.RequiredContentType != "" {
		if !strings.Contains(contentType, v.RequiredContentType) {
			return NewValidationError(fmt.Sprintf("Content-Type must be: %s", v.RequiredContentType))
		}
	}

	return nil
}

// EventTypeValidator 事件类型验证器
type EventTypeValidator struct {
	AllowedEventTypes []string
}

// Validate 验证事件类型
func (v *EventTypeValidator) Validate(r *http.Request, body []byte) error {
	if len(v.AllowedEventTypes) == 0 {
		return nil // 没有限制
	}

	// 预解析事件以获取事件类型
	var event CallbackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return NewValidationError("Cannot parse event for type validation")
	}

	for _, allowedType := range v.AllowedEventTypes {
		if event.EventType == allowedType {
			return nil
		}
	}

	return NewValidationError(fmt.Sprintf(
		"Event type %s is not allowed. Allowed types: %s",
		event.EventType, strings.Join(v.AllowedEventTypes, ", ")))
}

// BusinessValidator 业务逻辑验证器
type BusinessValidator struct {
	ValidateBusinessID func(businessID int64) error
	ValidateTaskID     func(taskID int64) error
	ValidateMetadata   func(metadata map[string]interface{}) error
}

// Validate 业务逻辑验证
func (v *BusinessValidator) Validate(r *http.Request, body []byte) error {
	// 预解析事件
	var event CallbackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return NewValidationError("Cannot parse event for business validation")
	}

	// 验证业务ID
	if v.ValidateBusinessID != nil {
		if err := v.ValidateBusinessID(event.BusinessID); err != nil {
			return err
		}
	}

	// 验证任务ID
	if v.ValidateTaskID != nil {
		if err := v.ValidateTaskID(event.TaskID); err != nil {
			return err
		}
	}

	// 验证元数据
	if v.ValidateMetadata != nil && event.Task.Metadata != nil {
		if err := v.ValidateMetadata(event.Task.Metadata); err != nil {
			return err
		}
	}

	return nil
}

// 工具函数和常量

// 预定义的事件类型常量
const (
	EventTypeTaskCreated   = "task.created"
	EventTypeTaskStarted   = "task.started"
	EventTypeTaskCompleted = "task.completed"
	EventTypeTaskFailed    = "task.failed"
)

// AllEventTypes 所有支持的事件类型
var AllEventTypes = []string{
	EventTypeTaskCreated,
	EventTypeTaskStarted,
	EventTypeTaskCompleted,
	EventTypeTaskFailed,
}

// IsValidEventType 检查是否为有效的事件类型
func IsValidEventType(eventType string) bool {
	return validatorContains(AllEventTypes, eventType)
}

// contains 检查切片是否包含指定元素
func validatorContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}