package fallback

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ErrFallbackFailed 降级失败错误
var ErrFallbackFailed = errors.New("all fallback strategies failed")

// ErrCircuitOpen 熔断器开启错误
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Strategy 降级策略接口
type Strategy interface {
	// Execute 执行策略
	Execute(ctx context.Context, primary func() (interface{}, error)) (interface{}, error)
	// Name 策略名称
	Name() string
	// IsAvailable 检查策略是否可用
	IsAvailable() bool
}

// FallbackFunc 降级函数类型
type FallbackFunc func(ctx context.Context, err error) (interface{}, error)

// SimpleFallback 简单降级策略
type SimpleFallback struct {
	name     string
	fallback FallbackFunc
}

// NewSimpleFallback 创建简单降级策略
func NewSimpleFallback(name string, fallback FallbackFunc) *SimpleFallback {
	return &SimpleFallback{
		name:     name,
		fallback: fallback,
	}
}

// Execute 执行简单降级
func (s *SimpleFallback) Execute(ctx context.Context, primary func() (interface{}, error)) (interface{}, error) {
	result, err := primary()
	if err != nil && s.fallback != nil {
		return s.fallback(ctx, err)
	}
	return result, err
}

// Name 返回策略名称
func (s *SimpleFallback) Name() string {
	return s.name
}

// IsAvailable 检查是否可用
func (s *SimpleFallback) IsAvailable() bool {
	return s.fallback != nil
}

// CacheFallback 缓存降级策略
type CacheFallback struct {
	name     string
	cache    map[string]CacheEntry
	cacheMu  sync.RWMutex
	ttl      time.Duration
	fallback FallbackFunc
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Value     interface{}
	Timestamp time.Time
}

// NewCacheFallback 创建缓存降级策略
func NewCacheFallback(name string, ttl time.Duration, fallback FallbackFunc) *CacheFallback {
	return &CacheFallback{
		name:     name,
		cache:    make(map[string]CacheEntry),
		ttl:      ttl,
		fallback: fallback,
	}
}

// Execute 执行缓存降级
func (c *CacheFallback) Execute(ctx context.Context, primary func() (interface{}, error)) (interface{}, error) {
	// 尝试执行主要逻辑
	result, err := primary()
	if err == nil {
		// 成功时更新缓存
		c.updateCache("last_success", result)
		return result, nil
	}

	// 失败时尝试从缓存获取
	if cached, found := c.getFromCache("last_success"); found {
		return cached, nil
	}

	// 缓存也没有，执行降级函数
	if c.fallback != nil {
		return c.fallback(ctx, err)
	}

	return nil, err
}

// Name 返回策略名称
func (c *CacheFallback) Name() string {
	return c.name
}

// IsAvailable 检查是否可用
func (c *CacheFallback) IsAvailable() bool {
	return true
}

// updateCache 更新缓存
func (c *CacheFallback) updateCache(key string, value interface{}) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache[key] = CacheEntry{
		Value:     value,
		Timestamp: time.Now(),
	}
}

// getFromCache 从缓存获取
func (c *CacheFallback) getFromCache(key string) (interface{}, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// 检查TTL
	if c.ttl > 0 && time.Since(entry.Timestamp) > c.ttl {
		return nil, false
	}

	return entry.Value, true
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name            string
	maxFailures     int64
	resetTimeout    time.Duration
	currentFailures int64
	lastFailTime    time.Time
	state           int32 // 0: Closed, 1: Open, 2: HalfOpen
	mu              sync.RWMutex
}

// CircuitState 熔断器状态
type CircuitState int32

const (
	StateClosed   CircuitState = 0
	StateOpen     CircuitState = 1
	StateHalfOpen CircuitState = 2
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(name string, maxFailures int64, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        int32(StateClosed),
	}
}

// Execute 执行熔断器保护的操作
func (cb *CircuitBreaker) Execute(ctx context.Context, primary func() (interface{}, error)) (interface{}, error) {
	// 检查熔断器状态
	if !cb.canExecute() {
		return nil, ErrCircuitOpen
	}

	// 执行操作
	result, err := primary()

	// 更新熔断器状态
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return result, err
}

// Name 返回熔断器名称
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// IsAvailable 检查熔断器是否可用
func (cb *CircuitBreaker) IsAvailable() bool {
	return cb.canExecute()
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitState {
	return CircuitState(atomic.LoadInt32(&cb.state))
}

// canExecute 检查是否可以执行
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state := CircuitState(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否可以转为半开状态
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			return cb.tryHalfOpen()
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// tryHalfOpen 尝试转为半开状态
func (cb *CircuitBreaker) tryHalfOpen() bool {
	if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
		return true
	}
	return false
}

// recordFailure 记录失败
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	failures := atomic.AddInt64(&cb.currentFailures, 1)
	cb.lastFailTime = time.Now()

	state := CircuitState(atomic.LoadInt32(&cb.state))

	// 如果失败次数超过阈值，开启熔断器
	if failures >= cb.maxFailures && state == StateClosed {
		atomic.StoreInt32(&cb.state, int32(StateOpen))
	} else if state == StateHalfOpen {
		// 半开状态下失败，直接开启
		atomic.StoreInt32(&cb.state, int32(StateOpen))
	}
}

// recordSuccess 记录成功
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := CircuitState(atomic.LoadInt32(&cb.state))

	// 重置失败计数
	atomic.StoreInt64(&cb.currentFailures, 0)

	// 如果是半开状态，转为关闭状态
	if state == StateHalfOpen {
		atomic.StoreInt32(&cb.state, int32(StateClosed))
	}
}

// ChainFallback 链式降级策略
type ChainFallback struct {
	name       string
	strategies []Strategy
}

// NewChainFallback 创建链式降级策略
func NewChainFallback(name string, strategies ...Strategy) *ChainFallback {
	return &ChainFallback{
		name:       name,
		strategies: strategies,
	}
}

// Execute 执行链式降级
func (c *ChainFallback) Execute(ctx context.Context, primary func() (interface{}, error)) (interface{}, error) {
	// 先尝试主要逻辑
	result, err := primary()
	if err == nil {
		return result, nil
	}

	// 逐一尝试降级策略
	for _, strategy := range c.strategies {
		if !strategy.IsAvailable() {
			continue
		}

		// 将主要逻辑包装为总是失败的函数，这样降级策略会执行自己的逻辑
		fallbackPrimary := func() (interface{}, error) {
			return nil, err
		}

		result, fallbackErr := strategy.Execute(ctx, fallbackPrimary)
		if fallbackErr == nil {
			return result, nil
		}

		// 记录降级失败，继续下一个策略
		err = fmt.Errorf("fallback strategy %s failed: %w", strategy.Name(), fallbackErr)
	}

	return nil, fmt.Errorf("%w: %v", ErrFallbackFailed, err)
}

// Name 返回策略名称
func (c *ChainFallback) Name() string {
	return c.name
}

// IsAvailable 检查是否有可用的策略
func (c *ChainFallback) IsAvailable() bool {
	for _, strategy := range c.strategies {
		if strategy.IsAvailable() {
			return true
		}
	}
	return false
}

// Manager 降级管理器
type Manager struct {
	strategies map[string]Strategy
	mu         sync.RWMutex
}

// NewManager 创建降级管理器
func NewManager() *Manager {
	return &Manager{
		strategies: make(map[string]Strategy),
	}
}

// Register 注册降级策略
func (m *Manager) Register(strategy Strategy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategies[strategy.Name()] = strategy
}

// Execute 执行指定的降级策略
func (m *Manager) Execute(ctx context.Context, name string, primary func() (interface{}, error)) (interface{}, error) {
	m.mu.RLock()
	strategy, exists := m.strategies[name]
	m.mu.RUnlock()

	if !exists {
		return primary()
	}

	return strategy.Execute(ctx, primary)
}

// GetStrategy 获取指定的降级策略
func (m *Manager) GetStrategy(name string) (Strategy, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	strategy, exists := m.strategies[name]
	return strategy, exists
}

// ListStrategies 列出所有策略
func (m *Manager) ListStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.strategies))
	for name := range m.strategies {
		names = append(names, name)
	}
	return names
}

// DefaultFallbacks 默认降级策略工厂
type DefaultFallbacks struct{}

// EmptyResponse 空响应降级
func (d *DefaultFallbacks) EmptyResponse() FallbackFunc {
	return func(ctx context.Context, err error) (interface{}, error) {
		return map[string]interface{}{
			"data":    []interface{}{},
			"message": "Service temporarily unavailable, returning empty data",
			"error":   err.Error(),
		}, nil
	}
}

// CachedResponse 缓存响应降级
func (d *DefaultFallbacks) CachedResponse(cache map[string]interface{}) FallbackFunc {
	return func(ctx context.Context, err error) (interface{}, error) {
		return map[string]interface{}{
			"data":    cache,
			"message": "Service temporarily unavailable, returning cached data",
			"error":   err.Error(),
		}, nil
	}
}

// ErrorResponse 错误响应降级
func (d *DefaultFallbacks) ErrorResponse(defaultMessage string) FallbackFunc {
	return func(ctx context.Context, err error) (interface{}, error) {
		return nil, fmt.Errorf("%s: %w", defaultMessage, err)
	}
}

// HTTPFallback HTTP特定的降级策略
type HTTPFallback struct {
	name         string
	fallbackFunc func(ctx context.Context, err error, statusCode int) (*http.Response, error)
}

// NewHTTPFallback 创建HTTP降级策略
func NewHTTPFallback(name string, fallbackFunc func(ctx context.Context, err error, statusCode int) (*http.Response, error)) *HTTPFallback {
	return &HTTPFallback{
		name:         name,
		fallbackFunc: fallbackFunc,
	}
}

// Execute 执行HTTP降级
func (h *HTTPFallback) Execute(ctx context.Context, primary func() (interface{}, error)) (interface{}, error) {
	result, err := primary()
	if err != nil && h.fallbackFunc != nil {
		// 尝试从错误中提取状态码
		statusCode := 500 // 默认500
		if httpErr, ok := err.(interface{ StatusCode() int }); ok {
			statusCode = httpErr.StatusCode()
		}
		return h.fallbackFunc(ctx, err, statusCode)
	}
	return result, err
}

// Name 返回策略名称
func (h *HTTPFallback) Name() string {
	return h.name
}

// IsAvailable 检查是否可用
func (h *HTTPFallback) IsAvailable() bool {
	return h.fallbackFunc != nil
}