package retry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// RetryCondition 重试条件判断函数
type RetryCondition func(attempt int, err error, resp *http.Response) bool

// BeforeRetry 重试前回调函数
type BeforeRetry func(attempt int, err error, backoff time.Duration)

// AfterRetry 重试后回调函数
type AfterRetry func(attempt int, err error, resp *http.Response, elapsed time.Duration)

// Policy 高级重试策略
type Policy struct {
	// 基础配置
	MaxAttempts     int           // 最大尝试次数（包含首次）
	MaxElapsedTime  time.Duration // 最大总重试时间
	BaseDelay       time.Duration // 基础延迟时间
	MaxDelay        time.Duration // 最大延迟时间
	Multiplier      float64       // 延迟倍数
	Jitter          bool          // 是否添加随机抖动

	// 高级配置
	RetryConditions []RetryCondition // 自定义重试条件
	RetryableErrors []error          // 可重试的错误类型
	RetryableCodes  []int            // 可重试的HTTP状态码
	NonRetryableCodes []int          // 明确不可重试的状态码

	// 回调函数
	BeforeRetry BeforeRetry // 重试前回调
	AfterRetry  AfterRetry  // 重试后回调

	// 内部状态
	backoffStrategy BackoffStrategy
}

// DefaultPolicy 返回默认重试策略
func DefaultPolicy() *Policy {
	return &Policy{
		MaxAttempts:    3,
		MaxElapsedTime: 5 * time.Minute,
		BaseDelay:      1 * time.Second,
		MaxDelay:       30 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
		RetryableCodes: []int{
			http.StatusTooManyRequests,     // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout,      // 504
		},
		NonRetryableCodes: []int{
			http.StatusBadRequest,          // 400
			http.StatusUnauthorized,        // 401
			http.StatusForbidden,           // 403
			http.StatusNotFound,            // 404
			http.StatusMethodNotAllowed,    // 405
			http.StatusConflict,           // 409
			http.StatusGone,               // 410
			http.StatusUnprocessableEntity, // 422
		},
		backoffStrategy: &ExponentialBackoff{},
	}
}

// ConservativePolicy 返回保守的重试策略（用于重要操作）
func ConservativePolicy() *Policy {
	policy := DefaultPolicy()
	policy.MaxAttempts = 2
	policy.MaxElapsedTime = 2 * time.Minute
	policy.BaseDelay = 2 * time.Second
	policy.MaxDelay = 10 * time.Second
	policy.Jitter = false
	return policy
}

// AggressivePolicy 返回激进的重试策略（用于非关键操作）
func AggressivePolicy() *Policy {
	policy := DefaultPolicy()
	policy.MaxAttempts = 5
	policy.MaxElapsedTime = 10 * time.Minute
	policy.BaseDelay = 500 * time.Millisecond
	policy.MaxDelay = 60 * time.Second
	policy.Multiplier = 1.5
	return policy
}

// NetworkPolicy 返回针对网络错误优化的重试策略
func NetworkPolicy() *Policy {
	policy := DefaultPolicy()
	policy.MaxAttempts = 4
	policy.BaseDelay = 2 * time.Second
	policy.MaxDelay = 15 * time.Second

	// 添加网络相关的重试条件
	policy.RetryConditions = append(policy.RetryConditions, func(attempt int, err error, resp *http.Response) bool {
		return IsNetworkError(err)
	})

	return policy
}

// WithMaxAttempts 设置最大尝试次数
func (p *Policy) WithMaxAttempts(attempts int) *Policy {
	if attempts < 1 {
		attempts = 1
	}
	p.MaxAttempts = attempts
	return p
}

// WithMaxElapsedTime 设置最大总重试时间
func (p *Policy) WithMaxElapsedTime(duration time.Duration) *Policy {
	p.MaxElapsedTime = duration
	return p
}

// WithDelay 设置延迟配置
func (p *Policy) WithDelay(base, max time.Duration, multiplier float64) *Policy {
	p.BaseDelay = base
	p.MaxDelay = max
	p.Multiplier = multiplier
	return p
}

// WithJitter 设置是否启用抖动
func (p *Policy) WithJitter(enabled bool) *Policy {
	p.Jitter = enabled
	return p
}

// WithRetryableCodes 设置可重试的HTTP状态码
func (p *Policy) WithRetryableCodes(codes ...int) *Policy {
	p.RetryableCodes = codes
	return p
}

// WithNonRetryableCodes 设置不可重试的HTTP状态码
func (p *Policy) WithNonRetryableCodes(codes ...int) *Policy {
	p.NonRetryableCodes = codes
	return p
}

// WithRetryCondition 添加自定义重试条件
func (p *Policy) WithRetryCondition(condition RetryCondition) *Policy {
	p.RetryConditions = append(p.RetryConditions, condition)
	return p
}

// WithBeforeRetry 设置重试前回调
func (p *Policy) WithBeforeRetry(callback BeforeRetry) *Policy {
	p.BeforeRetry = callback
	return p
}

// WithAfterRetry 设置重试后回调
func (p *Policy) WithAfterRetry(callback AfterRetry) *Policy {
	p.AfterRetry = callback
	return p
}

// WithBackoffStrategy 设置退避策略
func (p *Policy) WithBackoffStrategy(strategy BackoffStrategy) *Policy {
	p.backoffStrategy = strategy
	return p
}

// ShouldRetry 判断是否应该重试
func (p *Policy) ShouldRetry(attempt int, err error, resp *http.Response, elapsed time.Duration) bool {
	// 检查最大尝试次数
	if attempt >= p.MaxAttempts {
		return false
	}

	// 检查最大总时间
	if p.MaxElapsedTime > 0 && elapsed >= p.MaxElapsedTime {
		return false
	}

	// 如果有响应，检查状态码
	if resp != nil {
		statusCode := resp.StatusCode

		// 检查明确不可重试的状态码
		for _, code := range p.NonRetryableCodes {
			if statusCode == code {
				return false
			}
		}

		// 检查可重试的状态码
		for _, code := range p.RetryableCodes {
			if statusCode == code {
				return true
			}
		}

		// 2xx 状态码不重试
		if statusCode >= 200 && statusCode < 300 {
			return false
		}
	}

	// 检查错误类型
	if err != nil {
		// 检查可重试的错误类型
		for _, retryableErr := range p.RetryableErrors {
			if errors.Is(err, retryableErr) {
				return true
			}
		}

		// 检查默认的可重试错误
		if IsRetryableError(err) {
			return true
		}
	}

	// 执行自定义重试条件
	for _, condition := range p.RetryConditions {
		if condition(attempt, err, resp) {
			return true
		}
	}

	return false
}

// CalculateBackoff 计算退避时间
func (p *Policy) CalculateBackoff(attempt int) time.Duration {
	if p.backoffStrategy == nil {
		p.backoffStrategy = &ExponentialBackoff{}
	}

	config := BackoffConfig{
		BaseDelay:  p.BaseDelay,
		MaxDelay:   p.MaxDelay,
		Multiplier: p.Multiplier,
		Jitter:     p.Jitter,
	}

	return p.backoffStrategy.Calculate(attempt, config)
}

// ExecuteBeforeRetry 执行重试前回调
func (p *Policy) ExecuteBeforeRetry(attempt int, err error, backoff time.Duration) {
	if p.BeforeRetry != nil {
		p.BeforeRetry(attempt, err, backoff)
	}
}

// ExecuteAfterRetry 执行重试后回调
func (p *Policy) ExecuteAfterRetry(attempt int, err error, resp *http.Response, elapsed time.Duration) {
	if p.AfterRetry != nil {
		p.AfterRetry(attempt, err, resp, elapsed)
	}
}

// IsRetryableError 检查是否为可重试的错误
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 检查网络错误
	if IsNetworkError(err) {
		return true
	}

	// 检查超时错误
	if IsTimeoutError(err) {
		return true
	}

	// 检查连接错误
	if IsConnectionError(err) {
		return true
	}

	return false
}

// IsNetworkError 检查是否为网络错误
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return IsNetworkError(urlErr.Err)
	}

	return false
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return IsTimeoutError(urlErr.Err)
	}

	return errors.Is(err, context.DeadlineExceeded)
}

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 检查系统级连接错误
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return IsConnectionError(urlErr.Err)
	}

	return false
}

// Context 重试执行上下文
type Context struct {
	StartTime    time.Time
	Attempt      int
	LastError    error
	LastResponse *http.Response
	Policy       *Policy
}

// NewContext 创建新的重试上下文
func NewContext(policy *Policy) *Context {
	return &Context{
		StartTime: time.Now(),
		Attempt:   0,
		Policy:    policy,
	}
}

// ShouldRetry 检查是否应该继续重试
func (c *Context) ShouldRetry(err error, resp *http.Response) bool {
	elapsed := time.Since(c.StartTime)
	return c.Policy.ShouldRetry(c.Attempt, err, resp, elapsed)
}

// NextAttempt 准备下一次尝试
func (c *Context) NextAttempt(err error, resp *http.Response) time.Duration {
	c.LastError = err
	c.LastResponse = resp
	c.Attempt++

	if c.Attempt > 1 { // 不是第一次尝试才需要退避
		backoff := c.Policy.CalculateBackoff(c.Attempt - 1)
		c.Policy.ExecuteBeforeRetry(c.Attempt-1, err, backoff)
		return backoff
	}

	return 0
}

// Finish 完成重试，执行最终回调
func (c *Context) Finish(err error, resp *http.Response) {
	if c.Attempt > 1 {
		elapsed := time.Since(c.StartTime)
		c.Policy.ExecuteAfterRetry(c.Attempt-1, err, resp, elapsed)
	}
}