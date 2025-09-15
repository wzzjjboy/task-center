package retry

import (
	"math"
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	strategy := &ExponentialBackoff{}
	config := BackoffConfig{
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},  // 1 * 2^0
		{2, 2 * time.Second},  // 1 * 2^1
		{3, 4 * time.Second},  // 1 * 2^2
		{4, 8 * time.Second},  // 1 * 2^3
		{5, 16 * time.Second}, // 1 * 2^4
		{6, 30 * time.Second}, // 应该被限制在MaxDelay
		{10, 30 * time.Second}, // 应该被限制在MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := strategy.Calculate(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestLinearBackoff(t *testing.T) {
	strategy := &LinearBackoff{}
	config := BackoffConfig{
		BaseDelay:  1 * time.Second,
		MaxDelay:   10 * time.Second,
		Multiplier: 0, // Linear backoff doesn't use multiplier
		Jitter:     false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},  // 1 * 1
		{2, 2 * time.Second},  // 1 * 2
		{3, 3 * time.Second},  // 1 * 3
		{4, 4 * time.Second},  // 1 * 4
		{5, 5 * time.Second},  // 1 * 5
		{15, 10 * time.Second}, // 应该被限制在MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := strategy.Calculate(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestFixedBackoff(t *testing.T) {
	strategy := &FixedBackoff{}
	config := BackoffConfig{
		BaseDelay:  2 * time.Second,
		MaxDelay:   0, // 固定退避不使用MaxDelay
		Multiplier: 0, // 固定退避不使用Multiplier
		Jitter:     false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 2 * time.Second},
		{2, 2 * time.Second},
		{3, 2 * time.Second},
		{10, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := strategy.Calculate(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestDecorrelatedJitterBackoff(t *testing.T) {
	strategy := &DecorrelatedJitterBackoff{}
	config := BackoffConfig{
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 0, // 去相关抖动不使用标准multiplier
		Jitter:     false, // 内置抖动逻辑
	}

	// 第一次尝试应该返回BaseDelay
	result1 := strategy.Calculate(1, config)
	if result1 != config.BaseDelay {
		t.Errorf("First attempt should return BaseDelay, got %v", result1)
	}

	// 后续尝试应该在合理范围内
	for i := 2; i <= 5; i++ {
		result := strategy.Calculate(i, config)
		if result <= 0 {
			t.Errorf("Attempt %d: result should be positive, got %v", i, result)
		}
		if result > config.MaxDelay {
			t.Errorf("Attempt %d: result %v should not exceed MaxDelay %v", i, result, config.MaxDelay)
		}
		t.Logf("Attempt %d: %v", i, result)
	}
}

func TestEqualJitterBackoff(t *testing.T) {
	strategy := &EqualJitterBackoff{}
	config := BackoffConfig{
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     false, // 内置抖动逻辑
	}

	for i := 1; i <= 5; i++ {
		result := strategy.Calculate(i, config)

		// 计算期望的基础延迟
		expectedBase := float64(config.BaseDelay)
		if i > 1 {
			expectedBase = expectedBase * math.Pow(config.Multiplier, float64(i-1))
		}
		if config.MaxDelay > 0 && time.Duration(expectedBase) > config.MaxDelay {
			expectedBase = float64(config.MaxDelay)
		}

		// 等抖动应该在[baseDelay/2, baseDelay]范围内
		minExpected := time.Duration(expectedBase / 2)
		maxExpected := time.Duration(expectedBase)

		if result < minExpected || result > maxExpected {
			t.Errorf("Attempt %d: result %v should be in range [%v, %v]", i, result, minExpected, maxExpected)
		}
		t.Logf("Attempt %d: %v (expected range [%v, %v])", i, result, minExpected, maxExpected)
	}
}

func TestFullJitterBackoff(t *testing.T) {
	strategy := &FullJitterBackoff{}
	config := BackoffConfig{
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     false, // 内置抖动逻辑
	}

	for i := 1; i <= 5; i++ {
		result := strategy.Calculate(i, config)

		// 计算期望的基础延迟
		expectedBase := float64(config.BaseDelay)
		if i > 1 {
			expectedBase = expectedBase * math.Pow(config.Multiplier, float64(i-1))
		}
		if config.MaxDelay > 0 && time.Duration(expectedBase) > config.MaxDelay {
			expectedBase = float64(config.MaxDelay)
		}

		// 全抖动应该在[0, baseDelay]范围内
		maxExpected := time.Duration(expectedBase)

		if result < 0 || result > maxExpected {
			t.Errorf("Attempt %d: result %v should be in range [0, %v]", i, result, maxExpected)
		}
		t.Logf("Attempt %d: %v (expected range [0, %v])", i, result, maxExpected)
	}
}

func TestCustomBackoff(t *testing.T) {
	customFunc := func(attempt int, config BackoffConfig) time.Duration {
		// 自定义算法：attempt^2 * BaseDelay
		return time.Duration(attempt*attempt) * config.BaseDelay
	}

	strategy := &CustomBackoff{Calculator: customFunc}
	config := BackoffConfig{
		BaseDelay: 100 * time.Millisecond,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 100 * time.Millisecond}, // 1^2 * 100ms
		{2, 400 * time.Millisecond}, // 2^2 * 100ms
		{3, 900 * time.Millisecond}, // 3^2 * 100ms
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := strategy.Calculate(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestCustomBackoffNilCalculator(t *testing.T) {
	strategy := &CustomBackoff{Calculator: nil}
	config := BackoffConfig{BaseDelay: 1 * time.Second}

	result := strategy.Calculate(1, config)
	if result != 0 {
		t.Errorf("Expected 0 for nil calculator, got %v", result)
	}
}

func TestBackoffSequence(t *testing.T) {
	sequence := BackoffSequence{
		100 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	config := BackoffConfig{
		MaxDelay: 800 * time.Millisecond,
		Jitter:   false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 500 * time.Millisecond},
		{4, 800 * time.Millisecond}, // 被MaxDelay限制
		{5, 800 * time.Millisecond}, // 超出序列，使用最后一个值，但被MaxDelay限制
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := sequence.Calculate(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestPredefinedSequences(t *testing.T) {
	config := BackoffConfig{Jitter: false}

	// 测试快速序列
	if len(PredefinedSequences.Fast) == 0 {
		t.Error("Fast sequence should not be empty")
	}

	result := PredefinedSequences.Fast.Calculate(1, config)
	expected := 100 * time.Millisecond
	if result != expected {
		t.Errorf("Fast sequence first attempt: expected %v, got %v", expected, result)
	}

	// 测试标准序列
	if len(PredefinedSequences.Standard) == 0 {
		t.Error("Standard sequence should not be empty")
	}

	result = PredefinedSequences.Standard.Calculate(1, config)
	expected = 1 * time.Second
	if result != expected {
		t.Errorf("Standard sequence first attempt: expected %v, got %v", expected, result)
	}

	// 测试保守序列
	if len(PredefinedSequences.Conservative) == 0 {
		t.Error("Conservative sequence should not be empty")
	}

	// 测试网络序列
	if len(PredefinedSequences.Network) == 0 {
		t.Error("Network sequence should not be empty")
	}
}

func TestNewBackoffStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		expected string
	}{
		{"exponential", "*retry.ExponentialBackoff"},
		{"linear", "*retry.LinearBackoff"},
		{"fixed", "*retry.FixedBackoff"},
		{"decorrelated", "*retry.DecorrelatedJitterBackoff"},
		{"equal_jitter", "*retry.EqualJitterBackoff"},
		{"full_jitter", "*retry.FullJitterBackoff"},
		{"unknown", "*retry.ExponentialBackoff"}, // 默认返回指数退避
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			result := NewBackoffStrategy(tt.strategy)
			if result == nil {
				t.Error("Strategy should not be nil")
			}
			// 这里可以添加更具体的类型检查
		})
	}
}

func TestAddJitter(t *testing.T) {
	baseDuration := 1 * time.Second

	// 测试多次以确保抖动的随机性
	for i := 0; i < 10; i++ {
		result := addJitter(baseDuration)

		// 抖动结果应该在基础值的75%到125%之间（±25%）
		minExpected := time.Duration(float64(baseDuration) * 0.75)
		maxExpected := time.Duration(float64(baseDuration) * 1.25)

		if result < minExpected || result > maxExpected {
			t.Errorf("Iteration %d: jittered result %v should be in range [%v, %v]",
				i, result, minExpected, maxExpected)
		}
	}

	// 测试零值
	result := addJitter(0)
	if result != 0 {
		t.Errorf("Jitter of zero duration should return zero, got %v", result)
	}

	// 测试负值
	result = addJitter(-1 * time.Second)
	if result != -1*time.Second {
		t.Errorf("Jitter of negative duration should return original value, got %v", result)
	}
}

func TestAddJitterPercent(t *testing.T) {
	baseDuration := 1 * time.Second

	// 测试10%抖动
	for i := 0; i < 10; i++ {
		result := AddJitterPercent(baseDuration, 0.1)

		// 10%抖动应该在90%到110%之间
		minExpected := time.Duration(float64(baseDuration) * 0.9)
		maxExpected := time.Duration(float64(baseDuration) * 1.1)

		if result < minExpected || result > maxExpected {
			t.Errorf("10%% jitter result %v should be in range [%v, %v]",
				result, minExpected, maxExpected)
		}
	}

	// 测试零抖动
	result := AddJitterPercent(baseDuration, 0)
	if result != baseDuration {
		t.Errorf("0%% jitter should return original duration, got %v", result)
	}

	// 测试超过100%的抖动
	result = AddJitterPercent(baseDuration, 1.5)
	// 应该被限制在100%
	// 这个测试比较复杂，因为随机性，这里只检查不会panic
	if result < 0 {
		t.Error("Jitter result should not be negative")
	}
}

func TestCalculateWithCap(t *testing.T) {
	strategy := &ExponentialBackoff{}
	config := BackoffConfig{
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     false,
	}

	cap := 5 * time.Second

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second}, // 1s < 5s cap
		{2, 2 * time.Second}, // 2s < 5s cap
		{3, 4 * time.Second}, // 4s < 5s cap
		{4, 5 * time.Second}, // 8s 被cap限制为5s
		{5, 5 * time.Second}, // 16s 被cap限制为5s
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := CalculateWithCap(strategy, tt.attempt, config, cap)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		})
	}
}

func TestCalculateTotal(t *testing.T) {
	strategy := &FixedBackoff{}
	config := BackoffConfig{
		BaseDelay: 1 * time.Second,
		Jitter:    false,
	}

	// 固定1秒延迟，5次尝试，总共4次延迟（第一次不延迟）
	total := CalculateTotal(strategy, 5, config)
	expected := 4 * time.Second

	if total != expected {
		t.Errorf("Expected total delay %v, got %v", expected, total)
	}

	// 测试单次尝试
	total = CalculateTotal(strategy, 1, config)
	expected = 0 // 单次尝试没有延迟

	if total != expected {
		t.Errorf("Single attempt should have no delay, got %v", total)
	}
}