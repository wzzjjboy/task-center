package fallback

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestSimpleFallback(t *testing.T) {
	fallbackFunc := func(ctx context.Context, err error) (interface{}, error) {
		return "fallback result", nil
	}

	strategy := NewSimpleFallback("test", fallbackFunc)

	// 测试成功情况
	primary := func() (interface{}, error) {
		return "primary result", nil
	}

	result, err := strategy.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "primary result" {
		t.Errorf("Expected primary result, got %v", result)
	}

	// 测试失败情况
	primary = func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err = strategy.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "fallback result" {
		t.Errorf("Expected fallback result, got %v", result)
	}

	// 测试属性
	if strategy.Name() != "test" {
		t.Errorf("Expected name 'test', got %s", strategy.Name())
	}

	if !strategy.IsAvailable() {
		t.Error("Strategy should be available")
	}
}

func TestSimpleFallbackNilFunction(t *testing.T) {
	strategy := NewSimpleFallback("test", nil)

	if strategy.IsAvailable() {
		t.Error("Strategy with nil fallback should not be available")
	}

	primary := func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err := strategy.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error when fallback is nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestCacheFallback(t *testing.T) {
	fallbackFunc := func(ctx context.Context, err error) (interface{}, error) {
		return "final fallback", nil
	}

	strategy := NewCacheFallback("cache_test", 1*time.Minute, fallbackFunc)

	// 第一次成功调用，应该缓存结果
	primary := func() (interface{}, error) {
		return "cached result", nil
	}

	result, err := strategy.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "cached result" {
		t.Errorf("Expected cached result, got %v", result)
	}

	// 第二次失败调用，应该返回缓存结果
	primary = func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err = strategy.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "cached result" {
		t.Errorf("Expected cached result, got %v", result)
	}

	// 测试属性
	if strategy.Name() != "cache_test" {
		t.Errorf("Expected name 'cache_test', got %s", strategy.Name())
	}

	if !strategy.IsAvailable() {
		t.Error("Cache strategy should always be available")
	}
}

func TestCacheFallbackTTL(t *testing.T) {
	strategy := NewCacheFallback("ttl_test", 50*time.Millisecond, nil)

	// 缓存一个结果
	primary := func() (interface{}, error) {
		return "cached result", nil
	}
	strategy.Execute(context.Background(), primary)

	// 等待TTL过期
	time.Sleep(100 * time.Millisecond)

	// 现在失败的调用不应该返回过期的缓存
	primary = func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err := strategy.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error when cache is expired and no fallback")
	}
	if result != nil {
		t.Errorf("Expected nil result for expired cache, got %v", result)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test_cb", 2, 100*time.Millisecond)

	// 初始状态应该是关闭的
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.GetState())
	}

	if !cb.IsAvailable() {
		t.Error("Circuit breaker should be available initially")
	}

	// 第一次失败
	primary := func() (interface{}, error) {
		return nil, errors.New("failure 1")
	}

	result, err := cb.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error from primary function")
	}
	if cb.GetState() != StateClosed {
		t.Error("Circuit should still be closed after first failure")
	}

	// 第二次失败，应该触发熔断
	result, err = cb.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error from primary function")
	}
	if cb.GetState() != StateOpen {
		t.Error("Circuit should be open after second failure")
	}

	// 现在调用应该直接返回熔断错误
	result, err = cb.Execute(context.Background(), primary)
	if err != ErrCircuitOpen {
		t.Errorf("Expected circuit open error, got %v", err)
	}

	// 等待重置超时
	time.Sleep(150 * time.Millisecond)

	// 现在应该可以尝试执行（半开状态）
	successPrimary := func() (interface{}, error) {
		return "success", nil
	}

	result, err = cb.Execute(context.Background(), successPrimary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected success result, got %v", result)
	}
	if cb.GetState() != StateClosed {
		t.Error("Circuit should be closed after successful execution")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("half_open_test", 1, 50*time.Millisecond)

	// 触发熔断
	primary := func() (interface{}, error) {
		return nil, errors.New("failure")
	}

	cb.Execute(context.Background(), primary)
	if cb.GetState() != StateOpen {
		t.Error("Circuit should be open after failure")
	}

	// 等待重置超时
	time.Sleep(100 * time.Millisecond)

	// 半开状态下的失败应该重新打开熔断器
	_, err := cb.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error from primary function")
	}
	if cb.GetState() != StateOpen {
		t.Error("Circuit should be open again after failure in half-open state")
	}
}

func TestChainFallback(t *testing.T) {
	// 创建多个降级策略
	strategy1 := NewSimpleFallback("fallback1", func(ctx context.Context, err error) (interface{}, error) {
		return nil, errors.New("fallback1 failed")
	})

	strategy2 := NewSimpleFallback("fallback2", func(ctx context.Context, err error) (interface{}, error) {
		return "fallback2 success", nil
	})

	strategy3 := NewSimpleFallback("fallback3", func(ctx context.Context, err error) (interface{}, error) {
		return "fallback3 success", nil
	})

	chain := NewChainFallback("chain_test", strategy1, strategy2, strategy3)

	// 测试主要功能失败，应该执行第二个降级策略
	primary := func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err := chain.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "fallback2 success" {
		t.Errorf("Expected fallback2 result, got %v", result)
	}

	// 测试主要功能成功
	primary = func() (interface{}, error) {
		return "primary success", nil
	}

	result, err = chain.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "primary success" {
		t.Errorf("Expected primary result, got %v", result)
	}

	// 测试属性
	if chain.Name() != "chain_test" {
		t.Errorf("Expected name 'chain_test', got %s", chain.Name())
	}

	if !chain.IsAvailable() {
		t.Error("Chain should be available when it has available strategies")
	}
}

func TestChainFallbackAllFail(t *testing.T) {
	// 创建都会失败的降级策略
	strategy1 := NewSimpleFallback("fail1", func(ctx context.Context, err error) (interface{}, error) {
		return nil, errors.New("fallback1 failed")
	})

	strategy2 := NewSimpleFallback("fail2", func(ctx context.Context, err error) (interface{}, error) {
		return nil, errors.New("fallback2 failed")
	})

	chain := NewChainFallback("chain_fail", strategy1, strategy2)

	primary := func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err := chain.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error when all fallbacks fail")
	}
	if !errors.Is(err, ErrFallbackFailed) {
		t.Errorf("Expected ErrFallbackFailed, got %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()

	// 注册策略
	strategy1 := NewSimpleFallback("strategy1", func(ctx context.Context, err error) (interface{}, error) {
		return "strategy1 result", nil
	})

	strategy2 := NewSimpleFallback("strategy2", func(ctx context.Context, err error) (interface{}, error) {
		return "strategy2 result", nil
	})

	manager.Register(strategy1)
	manager.Register(strategy2)

	// 测试执行指定策略
	primary := func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err := manager.Execute(context.Background(), "strategy1", primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "strategy1 result" {
		t.Errorf("Expected strategy1 result, got %v", result)
	}

	// 测试不存在的策略
	result, err = manager.Execute(context.Background(), "nonexistent", primary)
	if err == nil {
		t.Error("Expected error for nonexistent strategy, but primary should be executed")
	}

	// 测试获取策略
	retrieved, exists := manager.GetStrategy("strategy1")
	if !exists {
		t.Error("Expected strategy1 to exist")
	}
	if retrieved.Name() != "strategy1" {
		t.Errorf("Expected strategy1, got %s", retrieved.Name())
	}

	_, exists = manager.GetStrategy("nonexistent")
	if exists {
		t.Error("Expected nonexistent strategy to not exist")
	}

	// 测试列出策略
	strategies := manager.ListStrategies()
	if len(strategies) != 2 {
		t.Errorf("Expected 2 strategies, got %d", len(strategies))
	}

	expectedNames := map[string]bool{"strategy1": true, "strategy2": true}
	for _, name := range strategies {
		if !expectedNames[name] {
			t.Errorf("Unexpected strategy name: %s", name)
		}
	}
}

func TestDefaultFallbacks(t *testing.T) {
	defaults := &DefaultFallbacks{}

	// 测试空响应降级
	emptyFallback := defaults.EmptyResponse()
	result, err := emptyFallback(context.Background(), errors.New("test error"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if resultMap, ok := result.(map[string]interface{}); ok {
		if data, exists := resultMap["data"]; !exists {
			t.Error("Expected data field in result")
		} else if dataSlice, ok := data.([]interface{}); !ok || len(dataSlice) != 0 {
			t.Errorf("Expected empty slice, got %v", data)
		}

		if message, exists := resultMap["message"]; !exists {
			t.Error("Expected message field in result")
		} else if !contains(message.(string), "temporarily unavailable") {
			t.Errorf("Expected appropriate message, got %v", message)
		}
	} else {
		t.Errorf("Expected map result, got %T", result)
	}

	// 测试缓存响应降级
	cache := map[string]interface{}{"cached": "data"}
	cachedFallback := defaults.CachedResponse(cache)
	result, err = cachedFallback(context.Background(), errors.New("test error"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if resultMap, ok := result.(map[string]interface{}); ok {
		if data, exists := resultMap["data"]; !exists {
			t.Error("Expected data field in result")
		} else if dataMap, ok := data.(map[string]interface{}); !ok {
			t.Errorf("Expected cached data, got %v", data)
		} else if dataMap["cached"] != "data" {
			t.Errorf("Expected cached data, got %v", dataMap)
		}
	} else {
		t.Errorf("Expected map result, got %T", result)
	}

	// 测试错误响应降级
	errorFallback := defaults.ErrorResponse("Custom error")
	result, err = errorFallback(context.Background(), errors.New("original error"))
	if err == nil {
		t.Error("Expected error from error fallback")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if !contains(err.Error(), "Custom error") {
		t.Errorf("Expected custom error message, got %v", err)
	}
}

func TestHTTPFallback(t *testing.T) {
	fallbackFunc := func(ctx context.Context, err error, statusCode int) (*http.Response, error) {
		// 创建一个模拟的HTTP响应
		resp := &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}
		resp.Header.Set("X-Fallback", "true")
		return resp, nil
	}

	strategy := NewHTTPFallback("http_test", fallbackFunc)

	// 测试成功情况
	primary := func() (interface{}, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	result, err := strategy.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp, ok := result.(*http.Response); !ok {
		t.Errorf("Expected HTTP response, got %T", result)
	} else if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 测试失败情况
	primary = func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err = strategy.Execute(context.Background(), primary)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp, ok := result.(*http.Response); !ok {
		t.Errorf("Expected HTTP response, got %T", result)
	} else if resp.Header.Get("X-Fallback") != "true" {
		t.Error("Expected fallback response")
	}

	// 测试属性
	if strategy.Name() != "http_test" {
		t.Errorf("Expected name 'http_test', got %s", strategy.Name())
	}

	if !strategy.IsAvailable() {
		t.Error("HTTP strategy should be available")
	}
}

func TestHTTPFallbackNilFunction(t *testing.T) {
	strategy := NewHTTPFallback("test", nil)

	if strategy.IsAvailable() {
		t.Error("Strategy with nil fallback should not be available")
	}

	primary := func() (interface{}, error) {
		return nil, errors.New("primary failed")
	}

	result, err := strategy.Execute(context.Background(), primary)
	if err == nil {
		t.Error("Expected error when fallback is nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// 辅助函数
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}