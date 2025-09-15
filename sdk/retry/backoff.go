package retry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy 退避策略接口
type BackoffStrategy interface {
	Calculate(attempt int, config BackoffConfig) time.Duration
}

// BackoffConfig 退避配置
type BackoffConfig struct {
	BaseDelay  time.Duration // 基础延迟
	MaxDelay   time.Duration // 最大延迟
	Multiplier float64       // 倍数
	Jitter     bool          // 是否添加抖动
}

// ExponentialBackoff 指数退避策略
type ExponentialBackoff struct{}

// Calculate 计算指数退避延迟
func (e *ExponentialBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// 计算指数退避: baseDelay * multiplier^(attempt-1)
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt-1))

	// 限制最大延迟
	if config.MaxDelay > 0 && time.Duration(delay) > config.MaxDelay {
		delay = float64(config.MaxDelay)
	}

	duration := time.Duration(delay)

	// 添加抖动
	if config.Jitter {
		duration = addJitter(duration)
	}

	return duration
}

// LinearBackoff 线性退避策略
type LinearBackoff struct{}

// Calculate 计算线性退避延迟
func (l *LinearBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// 计算线性退避: baseDelay * attempt
	delay := time.Duration(float64(config.BaseDelay) * float64(attempt))

	// 限制最大延迟
	if config.MaxDelay > 0 && delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// 添加抖动
	if config.Jitter {
		delay = addJitter(delay)
	}

	return delay
}

// FixedBackoff 固定退避策略
type FixedBackoff struct{}

// Calculate 计算固定退避延迟
func (f *FixedBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := config.BaseDelay

	// 添加抖动
	if config.Jitter {
		delay = addJitter(delay)
	}

	return delay
}

// DecorrelatedJitterBackoff 去相关抖动退避策略
// 这是AWS推荐的退避策略，避免"惊群效应"
type DecorrelatedJitterBackoff struct {
	lastDelay time.Duration
}

// Calculate 计算去相关抖动退避延迟
func (d *DecorrelatedJitterBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}

	if attempt == 1 {
		d.lastDelay = config.BaseDelay
		return d.lastDelay
	}

	// sleep = random(base, lastDelay * 3)
	maxNext := time.Duration(float64(d.lastDelay) * 3.0)
	if config.MaxDelay > 0 && maxNext > config.MaxDelay {
		maxNext = config.MaxDelay
	}

	// 确保最小延迟
	minDelay := config.BaseDelay
	if maxNext < minDelay {
		maxNext = minDelay
	}

	// 生成随机延迟
	delay := time.Duration(rand.Int63n(int64(maxNext-minDelay))) + minDelay
	d.lastDelay = delay

	return delay
}

// EqualJitterBackoff 等抖动退避策略
type EqualJitterBackoff struct{}

// Calculate 计算等抖动退避延迟
func (e *EqualJitterBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// 计算基础延迟
	baseDelay := float64(config.BaseDelay)
	if attempt > 1 {
		baseDelay = baseDelay * math.Pow(config.Multiplier, float64(attempt-1))
	}

	// 限制最大延迟
	if config.MaxDelay > 0 && time.Duration(baseDelay) > config.MaxDelay {
		baseDelay = float64(config.MaxDelay)
	}

	// 等抖动: delay = baseDelay/2 + random(0, baseDelay/2)
	halfDelay := baseDelay / 2
	jitter := rand.Float64() * halfDelay
	delay := time.Duration(halfDelay + jitter)

	return delay
}

// FullJitterBackoff 全抖动退避策略
type FullJitterBackoff struct{}

// Calculate 计算全抖动退避延迟
func (f *FullJitterBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// 计算基础延迟
	baseDelay := float64(config.BaseDelay)
	if attempt > 1 {
		baseDelay = baseDelay * math.Pow(config.Multiplier, float64(attempt-1))
	}

	// 限制最大延迟
	if config.MaxDelay > 0 && time.Duration(baseDelay) > config.MaxDelay {
		baseDelay = float64(config.MaxDelay)
	}

	// 全抖动: delay = random(0, baseDelay)
	delay := time.Duration(rand.Float64() * baseDelay)

	return delay
}

// CustomBackoff 自定义退避策略
type CustomBackoff struct {
	Calculator func(attempt int, config BackoffConfig) time.Duration
}

// Calculate 计算自定义退避延迟
func (c *CustomBackoff) Calculate(attempt int, config BackoffConfig) time.Duration {
	if c.Calculator == nil {
		return 0
	}
	return c.Calculator(attempt, config)
}

// addJitter 添加随机抖动 (±25%)
func addJitter(duration time.Duration) time.Duration {
	if duration <= 0 {
		return duration
	}

	// 计算抖动范围 (±25%)
	jitterRange := float64(duration) * 0.25
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange // -jitterRange 到 +jitterRange

	result := time.Duration(float64(duration) + jitter)
	if result < 0 {
		result = duration / 4 // 最小延迟
	}

	return result
}

// AddJitterPercent 添加指定百分比的随机抖动
func AddJitterPercent(duration time.Duration, percent float64) time.Duration {
	if duration <= 0 || percent <= 0 {
		return duration
	}

	if percent > 1.0 {
		percent = 1.0 // 最大100%抖动
	}

	jitterRange := float64(duration) * percent
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange

	result := time.Duration(float64(duration) + jitter)
	if result < 0 {
		result = time.Duration(float64(duration) * (1 - percent))
	}

	return result
}

// BackoffSequence 预定义的退避序列
type BackoffSequence []time.Duration

// Calculate 根据预定义序列计算延迟
func (s BackoffSequence) Calculate(attempt int, config BackoffConfig) time.Duration {
	if attempt <= 0 || len(s) == 0 {
		return 0
	}

	// 如果超出序列长度，使用最后一个值
	index := attempt - 1
	if index >= len(s) {
		index = len(s) - 1
	}

	delay := s[index]

	// 限制最大延迟
	if config.MaxDelay > 0 && delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// 添加抖动
	if config.Jitter {
		delay = addJitter(delay)
	}

	return delay
}

// PredefinedSequences 预定义的常用退避序列
var PredefinedSequences = struct {
	// 快速序列：适用于快速重试场景
	Fast BackoffSequence
	// 标准序列：适用于一般场景
	Standard BackoffSequence
	// 保守序列：适用于重要操作
	Conservative BackoffSequence
	// 网络序列：适用于网络不稳定场景
	Network BackoffSequence
}{
	Fast: BackoffSequence{
		100 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
	},
	Standard: BackoffSequence{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second,
	},
	Conservative: BackoffSequence{
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		20 * time.Second,
		30 * time.Second,
	},
	Network: BackoffSequence{
		500 * time.Millisecond,
		1 * time.Second,
		3 * time.Second,
		7 * time.Second,
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
	},
}

// NewBackoffStrategy 创建退避策略的工厂函数
func NewBackoffStrategy(strategy string) BackoffStrategy {
	switch strategy {
	case "exponential":
		return &ExponentialBackoff{}
	case "linear":
		return &LinearBackoff{}
	case "fixed":
		return &FixedBackoff{}
	case "decorrelated":
		return &DecorrelatedJitterBackoff{}
	case "equal_jitter":
		return &EqualJitterBackoff{}
	case "full_jitter":
		return &FullJitterBackoff{}
	default:
		return &ExponentialBackoff{} // 默认使用指数退避
	}
}

// CalculateWithCap 计算带上限的退避延迟
func CalculateWithCap(strategy BackoffStrategy, attempt int, config BackoffConfig, cap time.Duration) time.Duration {
	delay := strategy.Calculate(attempt, config)
	if cap > 0 && delay > cap {
		delay = cap
	}
	return delay
}

// CalculateTotal 计算总的重试延迟时间
func CalculateTotal(strategy BackoffStrategy, maxAttempts int, config BackoffConfig) time.Duration {
	var total time.Duration
	for i := 1; i < maxAttempts; i++ { // 从第二次尝试开始计算延迟
		total += strategy.Calculate(i, config)
	}
	return total
}