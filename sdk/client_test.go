package sdk

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				BaseURL:    "http://example.com",
				APIKey:     "test-key",
				BusinessID: 123,
				Timeout:    30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: true, // 会失败因为缺少必需字段
		},
		{
			name: "missing BaseURL",
			config: &Config{
				APIKey:     "test-key",
				BusinessID: 123,
			},
			wantErr: true,
		},
		{
			name: "missing APIKey",
			config: &Config{
				BaseURL:    "http://example.com",
				BusinessID: 123,
			},
			wantErr: true,
		},
		{
			name: "invalid BusinessID",
			config: &Config{
				BaseURL: "http://example.com",
				APIKey:  "test-key",
				BusinessID: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid BaseURL",
			config: &Config{
				BaseURL:    "not-a-url",
				APIKey:     "test-key",
				BusinessID: 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestNewClientWithDefaults(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		apiKey     string
		businessID int64
		wantErr    bool
	}{
		{
			name:       "valid parameters",
			baseURL:    "http://example.com",
			apiKey:     "test-key",
			businessID: 123,
			wantErr:    false,
		},
		{
			name:       "empty baseURL",
			baseURL:    "",
			apiKey:     "test-key",
			businessID: 123,
			wantErr:    true,
		},
		{
			name:       "empty apiKey",
			baseURL:    "http://example.com",
			apiKey:     "",
			businessID: 123,
			wantErr:    true,
		},
		{
			name:       "zero businessID",
			baseURL:    "http://example.com",
			apiKey:     "test-key",
			businessID: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientWithDefaults(tt.baseURL, tt.apiKey, tt.businessID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClientWithDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClientWithDefaults() returned nil client without error")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.UserAgent != "TaskCenter-Go-SDK/1.0.0" {
		t.Errorf("Expected user agent 'TaskCenter-Go-SDK/1.0.0', got %s", config.UserAgent)
	}

	if config.RetryPolicy == nil {
		t.Error("Expected non-nil RetryPolicy")
		return
	}

	if config.RetryPolicy.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", config.RetryPolicy.MaxRetries)
	}

	if config.RetryPolicy.InitialInterval != 1*time.Second {
		t.Errorf("Expected initial interval 1s, got %v", config.RetryPolicy.InitialInterval)
	}
}

func TestClient_shouldRetry(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "http://example.com"
	config.APIKey = "test-key"
	config.BusinessID = 123

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"success", 200, false},
		{"bad request", 400, false},
		{"unauthorized", 401, false},
		{"not found", 404, false},
		{"rate limit", 429, true},
		{"server error", 500, true},
		{"bad gateway", 502, true},
		{"service unavailable", 503, true},
		{"gateway timeout", 504, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.shouldRetry(tt.statusCode); got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_calculateRetryInterval(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "http://example.com"
	config.APIKey = "test-key"
	config.BusinessID = 123

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first retry", 1, 1 * time.Second},
		{"second retry", 2, 2 * time.Second},
		{"third retry", 3, 4 * time.Second},
		{"max interval", 10, 30 * time.Second}, // 应该被限制在最大间隔
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.calculateRetryInterval(tt.attempt)
			if got != tt.want {
				t.Errorf("calculateRetryInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_doRequest(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求头
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got %s", auth)
		}

		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
		}

		if userAgent := r.Header.Get("User-Agent"); userAgent != "TaskCenter-Go-SDK/1.0.0" {
			t.Errorf("Expected User-Agent 'TaskCenter-Go-SDK/1.0.0', got %s", userAgent)
		}

		if businessID := r.Header.Get("X-Business-ID"); businessID != "123" {
			t.Errorf("Expected X-Business-ID '123', got %s", businessID)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// 创建客户端
	client, err := NewClientWithDefaults(server.URL, "test-key", 123)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 测试请求
	resp, err := client.doRequest(nil, "GET", "/test", nil)
	if err != nil {
		t.Fatalf("doRequest() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestClient_doRequestWithRetry(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			// 前两次返回服务器错误，触发重试
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// 第三次返回成功
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// 创建客户端，设置较短的重试间隔
	config := DefaultConfig()
	config.BaseURL = server.URL
	config.APIKey = "test-key"
	config.BusinessID = 123
	config.RetryPolicy.InitialInterval = 10 * time.Millisecond
	config.RetryPolicy.MaxInterval = 100 * time.Millisecond

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 测试重试机制
	resp, err := client.doRequest(nil, "GET", "/test", nil)
	if err != nil {
		t.Fatalf("doRequest() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempt != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempt)
	}
}

func TestClient_doRequestMaxRetries(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		// 总是返回服务器错误
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// 创建客户端，设置较短的重试间隔
	config := DefaultConfig()
	config.BaseURL = server.URL
	config.APIKey = "test-key"
	config.BusinessID = 123
	config.RetryPolicy.InitialInterval = 10 * time.Millisecond
	config.RetryPolicy.MaxInterval = 100 * time.Millisecond
	config.RetryPolicy.MaxRetries = 2

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 测试最大重试次数
	_, err = client.doRequest(nil, "GET", "/test", nil)
	if err == nil {
		t.Error("Expected error after max retries, got nil")
	}

	expectedAttempts := config.RetryPolicy.MaxRetries + 1 // 初始尝试 + 重试次数
	if attempt != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempt)
	}
}