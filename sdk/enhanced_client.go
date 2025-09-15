package sdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"task-center/sdk/fallback"
	"task-center/sdk/retry"
)

// EnhancedClient 增强型客户端，支持高级重试和降级功能
type EnhancedClient struct {
	*Client                    // 嵌入基础客户端
	retryPolicy *retry.Policy // 高级重试策略
	fallbackMgr *fallback.Manager // 降级管理器
	circuitBreaker *fallback.CircuitBreaker // 熔断器
}

// EnhancedConfig 增强配置
type EnhancedConfig struct {
	*Config                           // 基础配置
	RetryPolicy    *retry.Policy     // 高级重试策略
	FallbackConfig *FallbackConfig   // 降级配置
	CircuitConfig  *CircuitConfig    // 熔断器配置
}

// FallbackConfig 降级配置
type FallbackConfig struct {
	EnableCache     bool          // 启用缓存降级
	CacheTTL        time.Duration // 缓存TTL
	EnableEmptyFallback bool      // 启用空响应降级
	CustomFallbacks map[string]fallback.FallbackFunc // 自定义降级函数
}

// CircuitConfig 熔断器配置
type CircuitConfig struct {
	MaxFailures  int64         // 最大失败次数
	ResetTimeout time.Duration // 重置超时时间
}

// DefaultEnhancedConfig 返回默认增强配置
func DefaultEnhancedConfig() *EnhancedConfig {
	return &EnhancedConfig{
		Config:      DefaultConfig(),
		RetryPolicy: retry.DefaultPolicy(),
		FallbackConfig: &FallbackConfig{
			EnableCache:         true,
			CacheTTL:           5 * time.Minute,
			EnableEmptyFallback: true,
			CustomFallbacks:     make(map[string]fallback.FallbackFunc),
		},
		CircuitConfig: &CircuitConfig{
			MaxFailures:  5,
			ResetTimeout: 30 * time.Second,
		},
	}
}

// NewEnhancedClient 创建增强型客户端
func NewEnhancedClient(config *EnhancedConfig) (*EnhancedClient, error) {
	if config == nil {
		config = DefaultEnhancedConfig()
	}

	// 创建基础客户端
	baseClient, err := NewClient(config.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base client: %w", err)
	}

	// 创建增强客户端
	enhancedClient := &EnhancedClient{
		Client:      baseClient,
		retryPolicy: config.RetryPolicy,
		fallbackMgr: fallback.NewManager(),
	}

	// 创建熔断器
	if config.CircuitConfig != nil {
		enhancedClient.circuitBreaker = fallback.NewCircuitBreaker(
			"main",
			config.CircuitConfig.MaxFailures,
			config.CircuitConfig.ResetTimeout,
		)
	}

	// 注册降级策略
	if config.FallbackConfig != nil {
		enhancedClient.setupFallbackStrategies(config.FallbackConfig)
	}

	return enhancedClient, nil
}

// setupFallbackStrategies 设置降级策略
func (c *EnhancedClient) setupFallbackStrategies(config *FallbackConfig) {
	defaults := &fallback.DefaultFallbacks{}

	// 缓存降级
	if config.EnableCache {
		cacheStrategy := fallback.NewCacheFallback(
			"cache",
			config.CacheTTL,
			defaults.EmptyResponse(),
		)
		c.fallbackMgr.Register(cacheStrategy)
	}

	// 空响应降级
	if config.EnableEmptyFallback {
		emptyStrategy := fallback.NewSimpleFallback(
			"empty",
			defaults.EmptyResponse(),
		)
		c.fallbackMgr.Register(emptyStrategy)
	}

	// 自定义降级策略
	for name, fallbackFunc := range config.CustomFallbacks {
		customStrategy := fallback.NewSimpleFallback(name, fallbackFunc)
		c.fallbackMgr.Register(customStrategy)
	}

	// 注册熔断器
	if c.circuitBreaker != nil {
		c.fallbackMgr.Register(c.circuitBreaker)
	}
}

// DoRequestWithEnhancedResilience 执行带增强弹性的HTTP请求
func (c *EnhancedClient) DoRequestWithEnhancedResilience(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	return c.DoRequestWithResilienceStrategy(ctx, method, path, body, "cache")
}

// DoRequestWithResilienceStrategy 执行带指定降级策略的HTTP请求
func (c *EnhancedClient) DoRequestWithResilienceStrategy(ctx context.Context, method, path string, body interface{}, fallbackStrategy string) (*http.Response, error) {
	// 创建重试上下文
	retryCtx := retry.NewContext(c.retryPolicy)

	// 主要请求函数
	primaryFunc := func() (interface{}, error) {
		return c.doRequest(ctx, method, path, body)
	}

	// 执行带降级的重试请求
	result, err := c.fallbackMgr.Execute(ctx, fallbackStrategy, func() (interface{}, error) {
		return c.executeWithAdvancedRetry(ctx, retryCtx, primaryFunc)
	})

	if err != nil {
		return nil, err
	}

	if resp, ok := result.(*http.Response); ok {
		return resp, nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// executeWithAdvancedRetry 执行高级重试逻辑
func (c *EnhancedClient) executeWithAdvancedRetry(ctx context.Context, retryCtx *retry.Context, primaryFunc func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var lastResp *http.Response

	for {
		// 执行主要逻辑
		result, err := primaryFunc()

		// 转换响应
		var resp *http.Response
		if result != nil {
			if r, ok := result.(*http.Response); ok {
				resp = r
			}
		}

		// 检查是否需要重试
		if !retryCtx.ShouldRetry(err, resp) {
			retryCtx.Finish(err, resp)
			return result, err
		}

		// 计算退避时间并等待
		backoff := retryCtx.NextAttempt(err, resp)
		if backoff > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				// 继续重试
			}
		}

		lastErr = err
		lastResp = resp
	}
}

// CreateTaskWithResilience 创建任务（带弹性处理）
func (c *EnhancedClient) CreateTaskWithResilience(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	resp, err := c.DoRequestWithEnhancedResilience(ctx, "POST", "/api/v1/tasks", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, ParseHTTPError(resp.StatusCode, nil)
	}

	var task Task
	if err := c.parseResponse(resp, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// GetTaskWithResilience 获取任务（带弹性处理）
func (c *EnhancedClient) GetTaskWithResilience(ctx context.Context, taskID string) (*Task, error) {
	path := fmt.Sprintf("/api/v1/tasks/%s", taskID)
	resp, err := c.DoRequestWithEnhancedResilience(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ParseHTTPError(resp.StatusCode, body)
	}

	var task Task
	if err := c.parseResponse(resp, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// ListTasksWithResilience 列出任务（带弹性处理）
func (c *EnhancedClient) ListTasksWithResilience(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error) {
	resp, err := c.DoRequestWithEnhancedResilience(ctx, "GET", "/api/v1/tasks", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ParseHTTPError(resp.StatusCode, body)
	}

	var listResp ListTasksResponse
	if err := c.parseResponse(resp, &listResp); err != nil {
		return nil, err
	}

	return &listResp, nil
}

// UpdateRetryPolicy 更新重试策略
func (c *EnhancedClient) UpdateRetryPolicy(policy *retry.Policy) {
	c.retryPolicy = policy
}

// GetRetryPolicy 获取当前重试策略
func (c *EnhancedClient) GetRetryPolicy() *retry.Policy {
	return c.retryPolicy
}

// RegisterFallbackStrategy 注册降级策略
func (c *EnhancedClient) RegisterFallbackStrategy(strategy fallback.Strategy) {
	c.fallbackMgr.Register(strategy)
}

// GetCircuitBreakerState 获取熔断器状态
func (c *EnhancedClient) GetCircuitBreakerState() fallback.CircuitState {
	if c.circuitBreaker != nil {
		return c.circuitBreaker.GetState()
	}
	return fallback.StateClosed
}

// ResetCircuitBreaker 重置熔断器
func (c *EnhancedClient) ResetCircuitBreaker() {
	if c.circuitBreaker != nil {
		// 创建新的熔断器实例来重置状态
		// 这里可以根据需要实现更精确的重置逻辑
	}
}

// WithCustomRetryPolicy 使用自定义重试策略
func (c *EnhancedClient) WithCustomRetryPolicy(policy *retry.Policy) *EnhancedClient {
	c.retryPolicy = policy
	return c
}

// WithFallbackStrategy 添加降级策略
func (c *EnhancedClient) WithFallbackStrategy(strategy fallback.Strategy) *EnhancedClient {
	c.fallbackMgr.Register(strategy)
	return c
}

// GetFallbackManager 获取降级管理器
func (c *EnhancedClient) GetFallbackManager() *fallback.Manager {
	return c.fallbackMgr
}

// HealthCheck 健康检查（带弹性处理）
func (c *EnhancedClient) HealthCheck(ctx context.Context) error {
	resp, err := c.DoRequestWithEnhancedResilience(ctx, "GET", "/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetStats 获取客户端统计信息
func (c *EnhancedClient) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 基础统计
	stats["circuit_breaker_state"] = c.GetCircuitBreakerState()
	stats["fallback_strategies"] = c.fallbackMgr.ListStrategies()

	// 重试策略信息
	if c.retryPolicy != nil {
		stats["retry_max_attempts"] = c.retryPolicy.MaxAttempts
		stats["retry_base_delay"] = c.retryPolicy.BaseDelay
		stats["retry_max_delay"] = c.retryPolicy.MaxDelay
	}

	return stats
}