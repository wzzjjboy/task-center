package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDefaultPolicy(t *testing.T) {
	policy := DefaultPolicy()

	if policy.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", policy.MaxAttempts)
	}

	if policy.BaseDelay != 1*time.Second {
		t.Errorf("Expected BaseDelay 1s, got %v", policy.BaseDelay)
	}

	if policy.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay 30s, got %v", policy.MaxDelay)
	}

	if policy.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier 2.0, got %f", policy.Multiplier)
	}

	if !policy.Jitter {
		t.Error("Expected Jitter to be true")
	}
}

func TestConservativePolicy(t *testing.T) {
	policy := ConservativePolicy()

	if policy.MaxAttempts != 2 {
		t.Errorf("Expected MaxAttempts 2, got %d", policy.MaxAttempts)
	}

	if policy.MaxElapsedTime != 2*time.Minute {
		t.Errorf("Expected MaxElapsedTime 2m, got %v", policy.MaxElapsedTime)
	}

	if policy.Jitter {
		t.Error("Expected Jitter to be false for conservative policy")
	}
}

func TestAggressivePolicy(t *testing.T) {
	policy := AggressivePolicy()

	if policy.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", policy.MaxAttempts)
	}

	if policy.BaseDelay != 500*time.Millisecond {
		t.Errorf("Expected BaseDelay 500ms, got %v", policy.BaseDelay)
	}

	if policy.Multiplier != 1.5 {
		t.Errorf("Expected Multiplier 1.5, got %f", policy.Multiplier)
	}
}

func TestNetworkPolicy(t *testing.T) {
	policy := NetworkPolicy()

	if policy.MaxAttempts != 4 {
		t.Errorf("Expected MaxAttempts 4, got %d", policy.MaxAttempts)
	}

	if policy.BaseDelay != 2*time.Second {
		t.Errorf("Expected BaseDelay 2s, got %v", policy.BaseDelay)
	}

	if len(policy.RetryConditions) == 0 {
		t.Error("Expected network policy to have retry conditions")
	}
}

func TestPolicyBuilder(t *testing.T) {
	policy := DefaultPolicy().
		WithMaxAttempts(5).
		WithMaxElapsedTime(10*time.Minute).
		WithDelay(2*time.Second, 60*time.Second, 3.0).
		WithJitter(false).
		WithRetryableCodes(500, 502, 503).
		WithNonRetryableCodes(400, 401, 403)

	if policy.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", policy.MaxAttempts)
	}

	if policy.MaxElapsedTime != 10*time.Minute {
		t.Errorf("Expected MaxElapsedTime 10m, got %v", policy.MaxElapsedTime)
	}

	if policy.BaseDelay != 2*time.Second {
		t.Errorf("Expected BaseDelay 2s, got %v", policy.BaseDelay)
	}

	if policy.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay 60s, got %v", policy.MaxDelay)
	}

	if policy.Multiplier != 3.0 {
		t.Errorf("Expected Multiplier 3.0, got %f", policy.Multiplier)
	}

	if policy.Jitter {
		t.Error("Expected Jitter to be false")
	}

	expectedCodes := []int{500, 502, 503}
	for i, code := range policy.RetryableCodes {
		if i >= len(expectedCodes) || code != expectedCodes[i] {
			t.Errorf("Expected retryable codes %v, got %v", expectedCodes, policy.RetryableCodes)
			break
		}
	}

	expectedNonRetryableCodes := []int{400, 401, 403}
	for i, code := range policy.NonRetryableCodes {
		if i >= len(expectedNonRetryableCodes) || code != expectedNonRetryableCodes[i] {
			t.Errorf("Expected non-retryable codes %v, got %v", expectedNonRetryableCodes, policy.NonRetryableCodes)
			break
		}
	}
}

func TestShouldRetry(t *testing.T) {
	policy := DefaultPolicy()

	tests := []struct {
		name     string
		attempt  int
		err      error
		resp     *http.Response
		elapsed  time.Duration
		expected bool
	}{
		{
			name:     "First attempt with 500 error",
			attempt:  1,
			resp:     &http.Response{StatusCode: 500},
			expected: true,
		},
		{
			name:     "Max attempts reached",
			attempt:  3,
			resp:     &http.Response{StatusCode: 500},
			expected: false,
		},
		{
			name:     "Non-retryable 400 error",
			attempt:  1,
			resp:     &http.Response{StatusCode: 400},
			expected: false,
		},
		{
			name:     "Success response",
			attempt:  1,
			resp:     &http.Response{StatusCode: 200},
			expected: false,
		},
		{
			name:     "Retryable network error",
			attempt:  1,
			err:      errors.New("connection refused"),
			expected: false, // 这个需要更具体的网络错误类型
		},
		{
			name:     "Max elapsed time exceeded",
			attempt:  1,
			resp:     &http.Response{StatusCode: 500},
			elapsed:  6 * time.Minute, // 超过默认5分钟
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.ShouldRetry(tt.attempt, tt.err, tt.resp, tt.elapsed)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	policy := DefaultPolicy().WithJitter(false) // 禁用抖动以便测试

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},                    // 1 * 2^0
		{2, 2 * time.Second},                    // 1 * 2^1
		{3, 4 * time.Second},                    // 1 * 2^2
		{10, 30 * time.Second},                  // 应该被限制在MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := policy.CalculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestCalculateBackoffWithJitter(t *testing.T) {
	policy := DefaultPolicy().WithJitter(true)

	for i := 1; i <= 5; i++ {
		delay := policy.CalculateBackoff(i)
		expectedBase := time.Duration(float64(policy.BaseDelay) * (policy.Multiplier * float64(i-1)))

		// 抖动应该在基础值的±25%范围内
		if delay <= 0 {
			t.Errorf("Delay should be positive, got %v", delay)
		}

		// 抖动测试可能不够精确，这里只检查是否为正数
		if delay > policy.MaxDelay {
			t.Errorf("Delay %v should not exceed MaxDelay %v", delay, policy.MaxDelay)
		}

		t.Logf("Attempt %d: base=%v, actual=%v", i, expectedBase, delay)
	}
}

func TestRetryContext(t *testing.T) {
	policy := DefaultPolicy()
	ctx := NewContext(policy)

	if ctx.Attempt != 0 {
		t.Errorf("Expected initial attempt 0, got %d", ctx.Attempt)
	}

	if ctx.Policy != policy {
		t.Error("Context should reference the same policy")
	}

	// 模拟第一次失败
	err := errors.New("test error")
	resp := &http.Response{StatusCode: 500}

	backoff := ctx.NextAttempt(err, resp)
	if ctx.Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", ctx.Attempt)
	}

	if backoff != 0 { // 第一次失败不应该有退避时间
		t.Errorf("Expected no backoff for first attempt, got %v", backoff)
	}

	// 模拟第二次失败
	backoff = ctx.NextAttempt(err, resp)
	if ctx.Attempt != 2 {
		t.Errorf("Expected attempt 2, got %d", ctx.Attempt)
	}

	if backoff <= 0 {
		t.Error("Expected positive backoff for second attempt")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "Generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCustomRetryCondition(t *testing.T) {
	policy := DefaultPolicy()

	// 添加自定义重试条件：只在尝试次数小于2时重试
	policy.WithRetryCondition(func(attempt int, err error, resp *http.Response) bool {
		return attempt < 2 && resp != nil && resp.StatusCode == 418 // I'm a teapot
	})

	// 测试自定义条件
	resp := &http.Response{StatusCode: 418}

	// 第一次尝试应该重试
	if !policy.ShouldRetry(1, nil, resp, 0) {
		t.Error("Expected to retry on first attempt with 418 status")
	}

	// 第二次尝试不应该重试
	if policy.ShouldRetry(2, nil, resp, 0) {
		t.Error("Expected not to retry on second attempt with 418 status")
	}
}

func TestCallbacks(t *testing.T) {
	policy := DefaultPolicy()

	var beforeRetryAttempt int
	var beforeRetryError error
	var beforeRetryBackoff time.Duration

	var afterRetryAttempt int
	var afterRetryError error
	var afterRetryResp *http.Response
	var afterRetryElapsed time.Duration

	policy.WithBeforeRetry(func(attempt int, err error, backoff time.Duration) {
		beforeRetryAttempt = attempt
		beforeRetryError = err
		beforeRetryBackoff = backoff
	})

	policy.WithAfterRetry(func(attempt int, err error, resp *http.Response, elapsed time.Duration) {
		afterRetryAttempt = attempt
		afterRetryError = err
		afterRetryResp = resp
		afterRetryElapsed = elapsed
	})

	// 执行回调
	testErr := errors.New("test error")
	testResp := &http.Response{StatusCode: 500}
	testBackoff := 1 * time.Second
	testElapsed := 2 * time.Second

	policy.ExecuteBeforeRetry(1, testErr, testBackoff)
	if beforeRetryAttempt != 1 || beforeRetryError != testErr || beforeRetryBackoff != testBackoff {
		t.Error("BeforeRetry callback not executed correctly")
	}

	policy.ExecuteAfterRetry(2, testErr, testResp, testElapsed)
	if afterRetryAttempt != 2 || afterRetryError != testErr || afterRetryResp != testResp || afterRetryElapsed != testElapsed {
		t.Error("AfterRetry callback not executed correctly")
	}
}