package async

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// 异步操作相关错误
var (
	ErrTaskQueueFull       = errors.New("task queue is full")
	ErrTimeout             = errors.New("operation timed out")
	ErrWorkerPoolFull      = errors.New("worker pool is full")
	ErrPipelineStageError  = errors.New("pipeline stage returned nil")
	ErrInvalidTaskID       = errors.New("invalid task ID")
	ErrTaskNotFound        = errors.New("task not found")
	ErrClientNotStarted    = errors.New("async client not started")
	ErrClientAlreadyStarted = errors.New("async client already started")
)

// AsyncError 异步操作错误
type AsyncError struct {
	Op      string    // 操作名称
	TaskID  string    // 任务ID
	Err     error     // 原始错误
	Time    time.Time // 错误发生时间
}

// Error 实现error接口
func (e *AsyncError) Error() string {
	if e.TaskID != "" {
		return fmt.Sprintf("async %s (task %s): %v", e.Op, e.TaskID, e.Err)
	}
	return fmt.Sprintf("async %s: %v", e.Op, e.Err)
}

// Unwrap 返回原始错误
func (e *AsyncError) Unwrap() error {
	return e.Err
}

// NewAsyncError 创建异步错误
func NewAsyncError(op, taskID string, err error) *AsyncError {
	return &AsyncError{
		Op:     op,
		TaskID: taskID,
		Err:    err,
		Time:   time.Now(),
	}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("async_%x_%d", bytes, time.Now().UnixNano())
}

// TimeoutError 超时错误
type TimeoutError struct {
	Operation string
	Timeout   time.Duration
}

// Error 实现error接口
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("%s timed out after %v", e.Operation, e.Timeout)
}

// IsTimeout 检查是否为超时错误
func IsTimeout(err error) bool {
	var timeoutErr *TimeoutError
	return errors.As(err, &timeoutErr) || errors.Is(err, ErrTimeout)
}

// QueueFullError 队列满错误
type QueueFullError struct {
	QueueType string
	Capacity  int
}

// Error 实现error接口
func (e *QueueFullError) Error() string {
	return fmt.Sprintf("%s queue is full (capacity: %d)", e.QueueType, e.Capacity)
}

// IsQueueFull 检查是否为队列满错误
func IsQueueFull(err error) bool {
	var queueFullErr *QueueFullError
	return errors.As(err, &queueFullErr) || errors.Is(err, ErrTaskQueueFull) || errors.Is(err, ErrWorkerPoolFull)
}

// ConcurrencyError 并发相关错误
type ConcurrencyError struct {
	Operation   string
	Concurrency int
	Limit       int
}

// Error 实现error接口
func (e *ConcurrencyError) Error() string {
	return fmt.Sprintf("%s: concurrency %d exceeds limit %d", e.Operation, e.Concurrency, e.Limit)
}

// ResultError 结果相关错误
type ResultError struct {
	TaskID    string
	Operation string
	Cause     error
}

// Error 实现error接口
func (e *ResultError) Error() string {
	return fmt.Sprintf("result error for task %s in %s: %v", e.TaskID, e.Operation, e.Cause)
}

// Unwrap 返回原始错误
func (e *ResultError) Unwrap() error {
	return e.Cause
}

// NewResultError 创建结果错误
func NewResultError(taskID, operation string, cause error) *ResultError {
	return &ResultError{
		TaskID:    taskID,
		Operation: operation,
		Cause:     cause,
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error 实现error接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s' (value: %v): %s", e.Field, e.Value, e.Message)
}

// NewValidationError 创建验证错误
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// ErrorCollector 错误收集器
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector 创建错误收集器
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{}
}

// Add 添加错误
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors 检查是否有错误
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors 获取所有错误
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Error 实现error接口，返回合并的错误信息
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return "no errors"
	}
	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}

	var msg = fmt.Sprintf("multiple errors (%d):", len(ec.errors))
	for i, err := range ec.errors {
		msg += fmt.Sprintf("\n  %d: %v", i+1, err)
	}
	return msg
}

// Clear 清空错误
func (ec *ErrorCollector) Clear() {
	ec.errors = ec.errors[:0]
}

// Count 获取错误数量
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// First 获取第一个错误
func (ec *ErrorCollector) First() error {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}

// Last 获取最后一个错误
func (ec *ErrorCollector) Last() error {
	if len(ec.errors) > 0 {
		return ec.errors[len(ec.errors)-1]
	}
	return nil
}

// Retry 重试配置
type RetryConfig struct {
	MaxAttempts int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	RetryIf      func(error) bool
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		RetryIf: func(err error) bool {
			// 默认只重试超时和队列满错误
			return IsTimeout(err) || IsQueueFull(err)
		},
	}
}

// ShouldRetry 检查是否应该重试
func (rc *RetryConfig) ShouldRetry(err error, attempt int) bool {
	if attempt >= rc.MaxAttempts {
		return false
	}
	if rc.RetryIf != nil {
		return rc.RetryIf(err)
	}
	return true
}

// GetDelay 获取重试延迟
func (rc *RetryConfig) GetDelay(attempt int) time.Duration {
	delay := time.Duration(float64(rc.InitialDelay) * (rc.Multiplier * float64(attempt)))
	if delay > rc.MaxDelay {
		delay = rc.MaxDelay
	}
	return delay
}

// Circuit breaker 状态
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// String 返回状态字符串
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerError 熔断器错误
type CircuitBreakerError struct {
	State       CircuitState
	LastError   error
	FailureRate float64
}

// Error 实现error接口
func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker is %s (failure rate: %.2f%%)", e.State, e.FailureRate*100)
}

// IsCircuitOpen 检查是否为熔断器开启错误
func IsCircuitOpen(err error) bool {
	var circuitErr *CircuitBreakerError
	return errors.As(err, &circuitErr) && circuitErr.State == CircuitOpen
}