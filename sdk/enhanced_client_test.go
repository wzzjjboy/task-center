package sdk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"task-center/sdk/fallback"
	"task-center/sdk/retry"
)

func TestNewEnhancedClient(t *testing.T) {
	config := DefaultEnhancedConfig()
	config.Config.BaseURL = "http://localhost:8080"
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create enhanced client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.retryPolicy == nil {
		t.Error("Retry policy should not be nil")
	}

	if client.fallbackMgr == nil {
		t.Error("Fallback manager should not be nil")
	}

	if client.circuitBreaker == nil {
		t.Error("Circuit breaker should not be nil")
	}
}

func TestNewEnhancedClientWithNilConfig(t *testing.T) {
	client, err := NewEnhancedClient(nil)
	if err == nil {
		t.Error("Expected error with nil config")
	}
	if client != nil {
		t.Error("Client should be nil on error")
	}
}

func TestDefaultEnhancedConfig(t *testing.T) {
	config := DefaultEnhancedConfig()

	if config == nil {
		t.Fatal("Config should not be nil")
	}

	if config.Config == nil {
		t.Error("Base config should not be nil")
	}

	if config.RetryPolicy == nil {
		t.Error("Retry policy should not be nil")
	}

	if config.FallbackConfig == nil {
		t.Error("Fallback config should not be nil")
	}

	if config.CircuitConfig == nil {
		t.Error("Circuit config should not be nil")
	}

	// 检查默认值
	if !config.FallbackConfig.EnableCache {
		t.Error("Cache should be enabled by default")
	}

	if config.FallbackConfig.CacheTTL != 5*time.Minute {
		t.Errorf("Expected cache TTL 5m, got %v", config.FallbackConfig.CacheTTL)
	}

	if !config.FallbackConfig.EnableEmptyFallback {
		t.Error("Empty fallback should be enabled by default")
	}

	if config.CircuitConfig.MaxFailures != 5 {
		t.Errorf("Expected max failures 5, got %d", config.CircuitConfig.MaxFailures)
	}

	if config.CircuitConfig.ResetTimeout != 30*time.Second {
		t.Errorf("Expected reset timeout 30s, got %v", config.CircuitConfig.ResetTimeout)
	}
}

func TestEnhancedClientMethods(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/tasks":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":"123","status":"pending"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"tasks":[],"total":0}`))
			}
		case "/api/v1/tasks/123":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"123","status":"completed"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := DefaultEnhancedConfig()
	config.Config.BaseURL = server.URL
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 测试健康检查
	err = client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// 测试创建任务
	req := &CreateTaskRequest{
		BusinessUniqueID: "test-task-001",
		CallbackURL:      "http://callback.test",
		Metadata:         map[string]interface{}{"test": "data"},
	}

	task, err := client.CreateTaskWithResilience(ctx, req)
	if err != nil {
		t.Errorf("Create task failed: %v", err)
	}
	if task == nil {
		t.Error("Task should not be nil")
	}

	// 测试获取任务
	task, err = client.GetTaskWithResilience(ctx, "123")
	if err != nil {
		t.Errorf("Get task failed: %v", err)
	}
	if task == nil {
		t.Error("Task should not be nil")
	}

	// 测试列出任务
	listReq := &ListTasksRequest{
		Page:     1,
		PageSize: 10,
	}

	listResp, err := client.ListTasksWithResilience(ctx, listReq)
	if err != nil {
		t.Errorf("List tasks failed: %v", err)
	}
	if listResp == nil {
		t.Error("List response should not be nil")
	}
}

func TestEnhancedClientRetryPolicy(t *testing.T) {
	config := DefaultEnhancedConfig()
	config.Config.BaseURL = "http://localhost:8080"
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 测试获取重试策略
	policy := client.GetRetryPolicy()
	if policy == nil {
		t.Error("Retry policy should not be nil")
	}

	// 测试更新重试策略
	newPolicy := retry.AggressivePolicy()
	client.UpdateRetryPolicy(newPolicy)

	updatedPolicy := client.GetRetryPolicy()
	if updatedPolicy != newPolicy {
		t.Error("Retry policy should be updated")
	}

	// 测试链式调用
	customPolicy := retry.DefaultPolicy().WithMaxAttempts(10)
	result := client.WithCustomRetryPolicy(customPolicy)
	if result != client {
		t.Error("WithCustomRetryPolicy should return the same client")
	}

	if client.GetRetryPolicy() != customPolicy {
		t.Error("Custom retry policy should be set")
	}
}

func TestEnhancedClientFallbackStrategies(t *testing.T) {
	config := DefaultEnhancedConfig()
	config.Config.BaseURL = "http://localhost:8080"
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 测试注册自定义降级策略
	customStrategy := fallback.NewSimpleFallback("custom", func(ctx context.Context, err error) (interface{}, error) {
		return "custom fallback", nil
	})

	result := client.WithFallbackStrategy(customStrategy)
	if result != client {
		t.Error("WithFallbackStrategy should return the same client")
	}

	// 验证策略已注册
	manager := client.GetFallbackManager()
	strategy, exists := manager.GetStrategy("custom")
	if !exists {
		t.Error("Custom strategy should be registered")
	}
	if strategy.Name() != "custom" {
		t.Errorf("Expected strategy name 'custom', got %s", strategy.Name())
	}
}

func TestEnhancedClientCircuitBreaker(t *testing.T) {
	config := DefaultEnhancedConfig()
	config.Config.BaseURL = "http://localhost:8080"
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 测试获取熔断器状态
	state := client.GetCircuitBreakerState()
	if state != fallback.StateClosed {
		t.Errorf("Expected initial state Closed, got %v", state)
	}

	// 测试重置熔断器（这里只是确保方法不会panic）
	client.ResetCircuitBreaker()
}

func TestEnhancedClientStats(t *testing.T) {
	config := DefaultEnhancedConfig()
	config.Config.BaseURL = "http://localhost:8080"
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	stats := client.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	// 检查统计信息字段
	if _, exists := stats["circuit_breaker_state"]; !exists {
		t.Error("Stats should include circuit breaker state")
	}

	if _, exists := stats["fallback_strategies"]; !exists {
		t.Error("Stats should include fallback strategies")
	}

	if _, exists := stats["retry_max_attempts"]; !exists {
		t.Error("Stats should include retry max attempts")
	}

	if _, exists := stats["retry_base_delay"]; !exists {
		t.Error("Stats should include retry base delay")
	}

	if _, exists := stats["retry_max_delay"]; !exists {
		t.Error("Stats should include retry max delay")
	}
}

func TestEnhancedClientWithFailingServer(t *testing.T) {
	// 创建会失败的测试服务器
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer server.Close()

	config := DefaultEnhancedConfig()
	config.Config.BaseURL = server.URL
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	// 设置更激进的重试策略以便快速测试
	config.RetryPolicy = retry.DefaultPolicy().
		WithMaxAttempts(5).
		WithDelay(10*time.Millisecond, 100*time.Millisecond, 2.0)

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 测试健康检查重试
	err = client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check should succeed after retries: %v", err)
	}

	// 验证确实进行了重试
	if attempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attempts)
	}
}

func TestEnhancedClientWithResilienceStrategy(t *testing.T) {
	// 创建会失败的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultEnhancedConfig()
	config.Config.BaseURL = server.URL
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 使用空响应降级策略测试
	resp, err := client.DoRequestWithResilienceStrategy(ctx, "GET", "/test", nil, "empty")

	// 由于服务器总是失败，应该触发降级
	// 这个测试的确切行为取决于降级策略的实现
	// 这里我们主要验证方法不会panic
	if err != nil {
		t.Logf("Expected error due to server failure: %v", err)
	}
	if resp != nil {
		t.Logf("Got response: %v", resp)
	}
}

func TestEnhancedClientConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*EnhancedConfig)
		expectError bool
	}{
		{
			name: "Valid config",
			modifyConfig: func(c *EnhancedConfig) {
				c.Config.BaseURL = "http://localhost:8080"
				c.Config.APIKey = "test-key"
				c.Config.BusinessID = 123
			},
			expectError: false,
		},
		{
			name: "Missing BaseURL",
			modifyConfig: func(c *EnhancedConfig) {
				c.Config.BaseURL = ""
				c.Config.APIKey = "test-key"
				c.Config.BusinessID = 123
			},
			expectError: true,
		},
		{
			name: "Missing APIKey",
			modifyConfig: func(c *EnhancedConfig) {
				c.Config.BaseURL = "http://localhost:8080"
				c.Config.APIKey = ""
				c.Config.BusinessID = 123
			},
			expectError: true,
		},
		{
			name: "Invalid BusinessID",
			modifyConfig: func(c *EnhancedConfig) {
				c.Config.BaseURL = "http://localhost:8080"
				c.Config.APIKey = "test-key"
				c.Config.BusinessID = 0
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultEnhancedConfig()
			tt.modifyConfig(config)

			client, err := NewEnhancedClient(config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if client != nil {
					t.Error("Expected nil client on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if client == nil {
					t.Error("Expected non-nil client")
				}
			}
		})
	}
}

func TestEnhancedClientCustomFallbacks(t *testing.T) {
	config := DefaultEnhancedConfig()
	config.Config.BaseURL = "http://localhost:8080"
	config.Config.APIKey = "test-key"
	config.Config.BusinessID = 123

	// 添加自定义降级函数
	config.FallbackConfig.CustomFallbacks = map[string]fallback.FallbackFunc{
		"custom1": func(ctx context.Context, err error) (interface{}, error) {
			return "custom1 result", nil
		},
		"custom2": func(ctx context.Context, err error) (interface{}, error) {
			return "custom2 result", nil
		},
	}

	client, err := NewEnhancedClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 验证自定义降级策略已注册
	manager := client.GetFallbackManager()
	strategies := manager.ListStrategies()

	expectedStrategies := []string{"cache", "empty", "custom1", "custom2", "main"}
	for _, expected := range expectedStrategies {
		found := false
		for _, actual := range strategies {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected strategy %s not found in %v", expected, strategies)
		}
	}
}

