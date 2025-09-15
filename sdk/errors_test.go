package sdk

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestBaseError(t *testing.T) {
	err := &BaseError{
		message:    "test error",
		code:       "TEST_ERROR",
		statusCode: 400,
		details:    map[string]string{"field": "value"},
	}

	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got %s", err.Error())
	}

	if err.Code() != "TEST_ERROR" {
		t.Errorf("Expected error code 'TEST_ERROR', got %s", err.Code())
	}

	if err.StatusCode() != 400 {
		t.Errorf("Expected status code 400, got %d", err.StatusCode())
	}

	details, ok := err.Details().(map[string]string)
	if !ok {
		t.Error("Expected details to be map[string]string")
	} else if details["field"] != "value" {
		t.Errorf("Expected details field=value, got %v", details)
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name       string
		constructor func(string) Error
		message     string
		expectedCode string
		expectedStatus int
	}{
		{"validation error", NewValidationError, "validation failed", CodeValidationError, http.StatusBadRequest},
		{"authentication error", NewAuthenticationError, "auth failed", CodeAuthenticationError, http.StatusUnauthorized},
		{"authorization error", NewAuthorizationError, "access denied", CodeAuthorizationError, http.StatusForbidden},
		{"conflict error", NewConflictError, "conflict", CodeConflictError, http.StatusConflict},
		{"rate limit error", NewRateLimitError, "rate limited", CodeRateLimitError, http.StatusTooManyRequests},
		{"server error", NewServerError, "server error", CodeServerError, http.StatusInternalServerError},
		{"network error", NewNetworkError, "network error", CodeNetworkError, 0},
		{"timeout error", NewTimeoutError, "timeout", CodeTimeoutError, http.StatusRequestTimeout},
		{"unknown error", NewUnknownError, "unknown", CodeUnknownError, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor(tt.message)

			if err.Error() != tt.message {
				t.Errorf("Expected message %s, got %s", tt.message, err.Error())
			}

			if err.Code() != tt.expectedCode {
				t.Errorf("Expected code %s, got %s", tt.expectedCode, err.Code())
			}

			if err.StatusCode() != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, err.StatusCode())
			}
		})
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("task")

	if err.Error() != "task not found" {
		t.Errorf("Expected message 'task not found', got %s", err.Error())
	}

	if err.Code() != CodeNotFoundError {
		t.Errorf("Expected code %s, got %s", CodeNotFoundError, err.Code())
	}

	if err.StatusCode() != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, err.StatusCode())
	}
}

func TestNewValidationErrorWithDetails(t *testing.T) {
	details := map[string]string{"field": "required"}
	err := NewValidationErrorWithDetails("validation failed", details)

	if err.Error() != "validation failed" {
		t.Errorf("Expected message 'validation failed', got %s", err.Error())
	}

	if err.Code() != CodeValidationError {
		t.Errorf("Expected code %s, got %s", CodeValidationError, err.Code())
	}

	errDetails, ok := err.Details().(map[string]string)
	if !ok {
		t.Error("Expected details to be map[string]string")
	} else if errDetails["field"] != "required" {
		t.Errorf("Expected details field=required, got %v", errDetails)
	}
}

func TestParseHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantCode   string
		wantMessage string
	}{
		{
			name:       "valid error response",
			statusCode: 400,
			body: []byte(`{
				"success": false,
				"message": "validation failed",
				"code": "VALIDATION_ERROR",
				"details": {"field": "required"}
			}`),
			wantCode:    "VALIDATION_ERROR",
			wantMessage: "validation failed",
		},
		{
			name:       "invalid JSON",
			statusCode: 400,
			body:       []byte(`invalid json`),
			wantCode:   CodeValidationError,
			wantMessage: "bad request",
		},
		{
			name:       "empty body",
			statusCode: 500,
			body:       []byte{},
			wantCode:   CodeServerError,
			wantMessage: "internal server error",
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       nil,
			wantCode:   CodeAuthenticationError,
			wantMessage: "authentication failed",
		},
		{
			name:       "forbidden",
			statusCode: 403,
			body:       nil,
			wantCode:   CodeAuthorizationError,
			wantMessage: "access denied",
		},
		{
			name:       "not found",
			statusCode: 404,
			body:       nil,
			wantCode:   CodeNotFoundError,
			wantMessage: "resource not found",
		},
		{
			name:       "conflict",
			statusCode: 409,
			body:       nil,
			wantCode:   CodeConflictError,
			wantMessage: "resource conflict",
		},
		{
			name:       "rate limit",
			statusCode: 429,
			body:       nil,
			wantCode:   CodeRateLimitError,
			wantMessage: "rate limit exceeded",
		},
		{
			name:       "bad gateway",
			statusCode: 502,
			body:       nil,
			wantCode:   CodeServerError,
			wantMessage: "bad gateway",
		},
		{
			name:       "service unavailable",
			statusCode: 503,
			body:       nil,
			wantCode:   CodeServerError,
			wantMessage: "service unavailable",
		},
		{
			name:       "gateway timeout",
			statusCode: 504,
			body:       nil,
			wantCode:   CodeTimeoutError,
			wantMessage: "gateway timeout",
		},
		{
			name:       "unknown status",
			statusCode: 418,
			body:       nil,
			wantCode:   CodeUnknownError,
			wantMessage: "HTTP 418: request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseHTTPError(tt.statusCode, tt.body)

			if err.Code() != tt.wantCode {
				t.Errorf("Expected code %s, got %s", tt.wantCode, err.Code())
			}

			if err.Error() != tt.wantMessage {
				t.Errorf("Expected message %s, got %s", tt.wantMessage, err.Error())
			}

			if err.StatusCode() != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, err.StatusCode())
			}
		})
	}
}

func TestErrorTypeCheckers(t *testing.T) {
	tests := []struct {
		name     string
		error    Error
		checkers map[string]func(error) bool
		expected map[string]bool
	}{
		{
			name:  "validation error",
			error: NewValidationError("test"),
			checkers: map[string]func(error) bool{
				"IsValidationError":     IsValidationError,
				"IsAuthenticationError": IsAuthenticationError,
				"IsAuthorizationError":  IsAuthorizationError,
				"IsNotFoundError":       IsNotFoundError,
				"IsConflictError":       IsConflictError,
				"IsRateLimitError":      IsRateLimitError,
				"IsServerError":         IsServerError,
				"IsNetworkError":        IsNetworkError,
				"IsTimeoutError":        IsTimeoutError,
				"IsRetryableError":      IsRetryableError,
			},
			expected: map[string]bool{
				"IsValidationError":     true,
				"IsAuthenticationError": false,
				"IsAuthorizationError":  false,
				"IsNotFoundError":       false,
				"IsConflictError":       false,
				"IsRateLimitError":      false,
				"IsServerError":         false,
				"IsNetworkError":        false,
				"IsTimeoutError":        false,
				"IsRetryableError":      false,
			},
		},
		{
			name:  "server error",
			error: NewServerError("test"),
			checkers: map[string]func(error) bool{
				"IsServerError":    IsServerError,
				"IsRetryableError": IsRetryableError,
			},
			expected: map[string]bool{
				"IsServerError":    true,
				"IsRetryableError": true,
			},
		},
		{
			name:  "rate limit error",
			error: NewRateLimitError("test"),
			checkers: map[string]func(error) bool{
				"IsRateLimitError": IsRateLimitError,
				"IsRetryableError": IsRetryableError,
			},
			expected: map[string]bool{
				"IsRateLimitError": true,
				"IsRetryableError": true,
			},
		},
		{
			name:  "network error",
			error: NewNetworkError("test"),
			checkers: map[string]func(error) bool{
				"IsNetworkError":   IsNetworkError,
				"IsRetryableError": IsRetryableError,
			},
			expected: map[string]bool{
				"IsNetworkError":   true,
				"IsRetryableError": true,
			},
		},
		{
			name:  "timeout error",
			error: NewTimeoutError("test"),
			checkers: map[string]func(error) bool{
				"IsTimeoutError":   IsTimeoutError,
				"IsRetryableError": IsRetryableError,
			},
			expected: map[string]bool{
				"IsTimeoutError":   true,
				"IsRetryableError": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for checkerName, checker := range tt.checkers {
				expected, exists := tt.expected[checkerName]
				if !exists {
					continue
				}

				result := checker(tt.error)
				if result != expected {
					t.Errorf("%s(%s) = %v, want %v", checkerName, tt.name, result, expected)
				}
			}
		})
	}
}

func TestErrorTypeCheckersWithNonSDKError(t *testing.T) {
	// 测试非SDK错误
	nonSDKErr := &json.SyntaxError{}

	checkers := []func(error) bool{
		IsValidationError,
		IsAuthenticationError,
		IsAuthorizationError,
		IsNotFoundError,
		IsConflictError,
		IsRateLimitError,
		IsServerError,
		IsNetworkError,
		IsTimeoutError,
		IsRetryableError,
	}

	for _, checker := range checkers {
		if checker(nonSDKErr) {
			t.Errorf("Error checker should return false for non-SDK error")
		}
	}
}