package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Error 表示SDK错误的接口
type Error interface {
	error
	Code() string
	StatusCode() int
	Details() interface{}
}

// BaseError SDK基础错误类型
type BaseError struct {
	message    string
	code       string
	statusCode int
	details    interface{}
}

// Error 实现error接口
func (e *BaseError) Error() string {
	return e.message
}

// Code 返回错误代码
func (e *BaseError) Code() string {
	return e.code
}

// StatusCode 返回HTTP状态码
func (e *BaseError) StatusCode() int {
	return e.statusCode
}

// Details 返回错误详情
func (e *BaseError) Details() interface{} {
	return e.details
}

// 预定义错误代码
const (
	CodeValidationError     = "VALIDATION_ERROR"
	CodeAuthenticationError = "AUTHENTICATION_ERROR"
	CodeAuthorizationError  = "AUTHORIZATION_ERROR"
	CodeNotFoundError       = "NOT_FOUND_ERROR"
	CodeConflictError       = "CONFLICT_ERROR"
	CodeRateLimitError      = "RATE_LIMIT_ERROR"
	CodeServerError         = "SERVER_ERROR"
	CodeNetworkError        = "NETWORK_ERROR"
	CodeTimeoutError        = "TIMEOUT_ERROR"
	CodeUnknownError        = "UNKNOWN_ERROR"
)

// NewValidationError 创建验证错误
func NewValidationError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeValidationError,
		statusCode: http.StatusBadRequest,
	}
}

// NewValidationErrorWithDetails 创建带详情的验证错误
func NewValidationErrorWithDetails(message string, details interface{}) Error {
	return &BaseError{
		message:    message,
		code:       CodeValidationError,
		statusCode: http.StatusBadRequest,
		details:    details,
	}
}

// NewAuthenticationError 创建认证错误
func NewAuthenticationError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeAuthenticationError,
		statusCode: http.StatusUnauthorized,
	}
}

// NewAuthorizationError 创建授权错误
func NewAuthorizationError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeAuthorizationError,
		statusCode: http.StatusForbidden,
	}
}

// NewNotFoundError 创建资源未找到错误
func NewNotFoundError(resource string) Error {
	return &BaseError{
		message:    fmt.Sprintf("%s not found", resource),
		code:       CodeNotFoundError,
		statusCode: http.StatusNotFound,
	}
}

// NewConflictError 创建冲突错误
func NewConflictError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeConflictError,
		statusCode: http.StatusConflict,
	}
}

// NewRateLimitError 创建速率限制错误
func NewRateLimitError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeRateLimitError,
		statusCode: http.StatusTooManyRequests,
	}
}

// NewServerError 创建服务器错误
func NewServerError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeServerError,
		statusCode: http.StatusInternalServerError,
	}
}

// NewNetworkError 创建网络错误
func NewNetworkError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeNetworkError,
		statusCode: 0, // 网络错误没有HTTP状态码
	}
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeTimeoutError,
		statusCode: http.StatusRequestTimeout,
	}
}

// NewUnknownError 创建未知错误
func NewUnknownError(message string) Error {
	return &BaseError{
		message:    message,
		code:       CodeUnknownError,
		statusCode: 0,
	}
}

// ParseHTTPError 从HTTP响应解析错误
func ParseHTTPError(statusCode int, body []byte) Error {
	var errorResp ErrorResponse

	// 尝试解析错误响应
	if len(body) > 0 {
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return &BaseError{
				message:    errorResp.Message,
				code:       errorResp.Code,
				statusCode: statusCode,
				details:    errorResp.Details,
			}
		}
	}

	// 根据HTTP状态码创建默认错误
	switch statusCode {
	case http.StatusBadRequest:
		return NewValidationError("bad request")
	case http.StatusUnauthorized:
		return NewAuthenticationError("authentication failed")
	case http.StatusForbidden:
		return NewAuthorizationError("access denied")
	case http.StatusNotFound:
		return NewNotFoundError("resource")
	case http.StatusConflict:
		return NewConflictError("resource conflict")
	case http.StatusTooManyRequests:
		return NewRateLimitError("rate limit exceeded")
	case http.StatusInternalServerError:
		return NewServerError("internal server error")
	case http.StatusBadGateway:
		return NewServerError("bad gateway")
	case http.StatusServiceUnavailable:
		return NewServerError("service unavailable")
	case http.StatusGatewayTimeout:
		return NewTimeoutError("gateway timeout")
	default:
		return &BaseError{
			message:    fmt.Sprintf("HTTP %d: request failed", statusCode),
			code:       CodeUnknownError,
			statusCode: statusCode,
		}
	}
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeValidationError
	}
	return false
}

// IsAuthenticationError 检查是否为认证错误
func IsAuthenticationError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeAuthenticationError
	}
	return false
}

// IsAuthorizationError 检查是否为授权错误
func IsAuthorizationError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeAuthorizationError
	}
	return false
}

// IsNotFoundError 检查是否为资源未找到错误
func IsNotFoundError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeNotFoundError
	}
	return false
}

// IsConflictError 检查是否为冲突错误
func IsConflictError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeConflictError
	}
	return false
}

// IsRateLimitError 检查是否为速率限制错误
func IsRateLimitError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeRateLimitError
	}
	return false
}

// IsServerError 检查是否为服务器错误
func IsServerError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeServerError
	}
	return false
}

// IsNetworkError 检查是否为网络错误
func IsNetworkError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeNetworkError
	}
	return false
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		return sdkErr.Code() == CodeTimeoutError
	}
	return false
}

// IsRetryableError 检查是否为可重试的错误
func IsRetryableError(err error) bool {
	if sdkErr, ok := err.(Error); ok {
		switch sdkErr.Code() {
		case CodeRateLimitError, CodeServerError, CodeNetworkError, CodeTimeoutError:
			return true
		}
	}
	return false
}