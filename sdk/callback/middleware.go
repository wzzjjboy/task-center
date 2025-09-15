package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

)

// Middleware 回调中间件接口
type Middleware interface {
	Before(w http.ResponseWriter, r *http.Request) error
	After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error
}

// Handler 回调处理器接口
type Handler interface {
	HandleTaskCreated(event *CallbackEvent) error
	HandleTaskStarted(event *CallbackEvent) error
	HandleTaskCompleted(event *CallbackEvent) error
	HandleTaskFailed(event *CallbackEvent) error
}

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain 创建中间件链
func NewMiddlewareChain(middlewares ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

// Add 添加中间件
func (c *MiddlewareChain) Add(middleware Middleware) {
	c.middlewares = append(c.middlewares, middleware)
}

// ExecuteBefore 执行前置中间件
func (c *MiddlewareChain) ExecuteBefore(w http.ResponseWriter, r *http.Request) error {
	for _, middleware := range c.middlewares {
		if err := middleware.Before(w, r); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfter 执行后置中间件
func (c *MiddlewareChain) ExecuteAfter(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	// 后置中间件按照相反的顺序执行
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		if err := c.middlewares[i].After(w, r, event); err != nil {
			return err
		}
	}
	return nil
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	Logger           *log.Logger
	LogHeaders       bool
	LogBody          bool
	LogResponseTime  bool
	ExcludeHeaders   []string
	IncludeOnlyPaths []string
}

// LoggingMiddlewareOption 日志中间件配置选项
type LoggingMiddlewareOption func(*LoggingMiddleware)

// WithLogger 设置日志器
func WithLogger(logger *log.Logger) LoggingMiddlewareOption {
	return func(m *LoggingMiddleware) {
		m.Logger = logger
	}
}

// WithLogHeaders 启用请求头日志
func WithLogHeaders(enabled bool) LoggingMiddlewareOption {
	return func(m *LoggingMiddleware) {
		m.LogHeaders = enabled
	}
}

// WithLogBody 启用请求体日志
func WithLogBody(enabled bool) LoggingMiddlewareOption {
	return func(m *LoggingMiddleware) {
		m.LogBody = enabled
	}
}

// WithLogResponseTime 启用响应时间日志
func WithLogResponseTime(enabled bool) LoggingMiddlewareOption {
	return func(m *LoggingMiddleware) {
		m.LogResponseTime = enabled
	}
}

// WithExcludeHeaders 排除指定的请求头
func WithExcludeHeaders(headers ...string) LoggingMiddlewareOption {
	return func(m *LoggingMiddleware) {
		m.ExcludeHeaders = headers
	}
}

// WithIncludeOnlyPaths 只对指定路径启用日志
func WithIncludeOnlyPaths(paths ...string) LoggingMiddlewareOption {
	return func(m *LoggingMiddleware) {
		m.IncludeOnlyPaths = paths
	}
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(opts ...LoggingMiddlewareOption) *LoggingMiddleware {
	m := &LoggingMiddleware{
		Logger:          log.Default(),
		LogHeaders:      false,
		LogBody:         false,
		LogResponseTime: true,
		ExcludeHeaders:  []string{"Authorization", "X-TaskCenter-Signature"},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Before 前置处理
func (m *LoggingMiddleware) Before(w http.ResponseWriter, r *http.Request) error {
	// 检查是否需要记录此路径
	if len(m.IncludeOnlyPaths) > 0 {
		found := false
		for _, path := range m.IncludeOnlyPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// 在请求上下文中记录开始时间
	if m.LogResponseTime {
		ctx := context.WithValue(r.Context(), "start_time", time.Now())
		*r = *r.WithContext(ctx)
	}

	logData := map[string]interface{}{
		"method":     r.Method,
		"url":        r.URL.String(),
		"user_agent": r.Header.Get("User-Agent"),
		"remote_ip":  getRealIP(r),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	// 记录请求头
	if m.LogHeaders {
		headers := make(map[string]string)
		for key, values := range r.Header {
			// 检查是否需要排除此请求头
			excluded := false
			for _, excludeHeader := range m.ExcludeHeaders {
				if strings.EqualFold(key, excludeHeader) {
					excluded = true
					break
				}
			}
			if !excluded {
				headers[key] = strings.Join(values, ", ")
			}
		}
		logData["headers"] = headers
	}

	// 记录请求体（需要小心处理，避免消费原始请求体）
	if m.LogBody && r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body.Close()

		// 恢复请求体，以便后续处理
		r.Body = io.NopCloser(bytes.NewReader(body))

		// 尝试解析为JSON以美化输出
		var jsonObj interface{}
		if json.Unmarshal(body, &jsonObj) == nil {
			logData["body"] = jsonObj
		} else {
			logData["body"] = string(body)
		}
	}

	m.Logger.Printf("Webhook request received: %s", formatLogData(logData))
	return nil
}

// After 后置处理
func (m *LoggingMiddleware) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	logData := map[string]interface{}{
		"event_type": event.EventType,
		"task_id":    event.TaskID,
		"business_id": event.BusinessID,
		"event_time": event.EventTime.UTC().Format(time.RFC3339),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	// 计算响应时间
	if m.LogResponseTime {
		if startTime, ok := r.Context().Value("start_time").(time.Time); ok {
			logData["response_time_ms"] = time.Since(startTime).Milliseconds()
		}
	}

	m.Logger.Printf("Webhook event processed: %s", formatLogData(logData))
	return nil
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct {
	mu               sync.RWMutex
	requestCount     map[string]int64
	responseTime     map[string][]time.Duration
	errorCount       map[string]int64
	lastRequestTime  time.Time
	requestInFlight  int64

	// 回调函数用于外部指标系统集成
	OnRequestStart    func(eventType string)
	OnRequestComplete func(eventType string, duration time.Duration)
	OnRequestError    func(eventType string, errType string)
}

// NewMetricsMiddleware 创建指标中间件
func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		requestCount: make(map[string]int64),
		responseTime: make(map[string][]time.Duration),
		errorCount:   make(map[string]int64),
	}
}

// Before 前置处理
func (m *MetricsMiddleware) Before(w http.ResponseWriter, r *http.Request) error {
	m.mu.Lock()
	m.requestInFlight++
	m.lastRequestTime = time.Now()
	m.mu.Unlock()

	// 在请求上下文中记录开始时间
	ctx := context.WithValue(r.Context(), "metrics_start_time", time.Now())
	*r = *r.WithContext(ctx)

	if m.OnRequestStart != nil {
		// 这里我们还不知道事件类型，使用通用标识
		m.OnRequestStart("webhook")
	}

	return nil
}

// After 后置处理
func (m *MetricsMiddleware) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestInFlight--

	// 增加请求计数
	m.requestCount[event.EventType]++
	m.requestCount["total"]++

	// 记录响应时间
	if startTime, ok := r.Context().Value("metrics_start_time").(time.Time); ok {
		duration := time.Since(startTime)
		if m.responseTime[event.EventType] == nil {
			m.responseTime[event.EventType] = make([]time.Duration, 0, 100)
		}
		m.responseTime[event.EventType] = append(m.responseTime[event.EventType], duration)

		// 保持最近100个记录以计算平均值
		if len(m.responseTime[event.EventType]) > 100 {
			m.responseTime[event.EventType] = m.responseTime[event.EventType][1:]
		}

		if m.OnRequestComplete != nil {
			m.OnRequestComplete(event.EventType, duration)
		}
	}

	return nil
}

// RecordError 记录错误
func (m *MetricsMiddleware) RecordError(eventType, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorCount[eventType+":"+errorType]++
	m.errorCount["total_errors"]++

	if m.OnRequestError != nil {
		m.OnRequestError(eventType, errorType)
	}
}

// GetMetrics 获取指标数据
func (m *MetricsMiddleware) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := map[string]interface{}{
		"request_count":     copyInt64Map(m.requestCount),
		"error_count":       copyInt64Map(m.errorCount),
		"request_in_flight": m.requestInFlight,
		"last_request_time": m.lastRequestTime.UTC().Format(time.RFC3339),
	}

	// 计算平均响应时间
	avgResponseTime := make(map[string]float64)
	for eventType, durations := range m.responseTime {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avgResponseTime[eventType] = float64(total.Milliseconds()) / float64(len(durations))
		}
	}
	metrics["avg_response_time_ms"] = avgResponseTime

	return metrics
}

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	AllowedIPs          []string
	AllowedUserAgents   []string
	RequiredHeaders     map[string]string
	RateLimitPerMinute  int
	EnableCORS          bool
	CORSOrigins         []string

	// 内部状态
	mu              sync.RWMutex
	requestCounts   map[string][]time.Time
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware() *SecurityMiddleware {
	return &SecurityMiddleware{
		requestCounts: make(map[string][]time.Time),
	}
}

// Before 前置处理
func (s *SecurityMiddleware) Before(w http.ResponseWriter, r *http.Request) error {
	// IP白名单检查
	if len(s.AllowedIPs) > 0 {
		clientIP := getRealIP(r)
		allowed := false
		for _, allowedIP := range s.AllowedIPs {
			if clientIP == allowedIP {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewAuthorizationError(fmt.Sprintf("IP %s is not allowed", clientIP))
		}
	}

	// User-Agent检查
	if len(s.AllowedUserAgents) > 0 {
		userAgent := r.Header.Get("User-Agent")
		allowed := false
		for _, allowedUA := range s.AllowedUserAgents {
			if strings.Contains(userAgent, allowedUA) {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewAuthorizationError("User-Agent not allowed")
		}
	}

	// 必需请求头检查
	for header, expectedValue := range s.RequiredHeaders {
		actualValue := r.Header.Get(header)
		if actualValue != expectedValue {
			return NewValidationError(fmt.Sprintf("Required header %s missing or invalid", header))
		}
	}

	// 速率限制检查
	if s.RateLimitPerMinute > 0 {
		clientIP := getRealIP(r)
		if !s.checkRateLimit(clientIP) {
			return NewRateLimitError("Rate limit exceeded")
		}
	}

	// CORS处理
	if s.EnableCORS {
		origin := r.Header.Get("Origin")
		if len(s.CORSOrigins) == 0 || contains(s.CORSOrigins, origin) || contains(s.CORSOrigins, "*") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-TaskCenter-Signature, X-TaskCenter-Timestamp")
		}
	}

	return nil
}

// After 后置处理
func (s *SecurityMiddleware) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	// 这里可以添加安全相关的后置处理逻辑
	return nil
}

// checkRateLimit 检查速率限制
func (s *SecurityMiddleware) checkRateLimit(clientIP string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// 清理过期的请求记录
	if times, exists := s.requestCounts[clientIP]; exists {
		validTimes := make([]time.Time, 0, len(times))
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		s.requestCounts[clientIP] = validTimes
	}

	// 检查是否超过限制
	currentCount := len(s.requestCounts[clientIP])
	if currentCount >= s.RateLimitPerMinute {
		return false
	}

	// 记录当前请求
	s.requestCounts[clientIP] = append(s.requestCounts[clientIP], now)
	return true
}

// DefaultHandler 默认回调处理器实现
type DefaultHandler struct {
	OnTaskCreated   func(*CallbackEvent) error
	OnTaskStarted   func(*CallbackEvent) error
	OnTaskCompleted func(*CallbackEvent) error
	OnTaskFailed    func(*CallbackEvent) error
}

// HandleTaskCreated 处理任务创建事件
func (h *DefaultHandler) HandleTaskCreated(event *CallbackEvent) error {
	if h.OnTaskCreated != nil {
		return h.OnTaskCreated(event)
	}
	return nil
}

// HandleTaskStarted 处理任务开始事件
func (h *DefaultHandler) HandleTaskStarted(event *CallbackEvent) error {
	if h.OnTaskStarted != nil {
		return h.OnTaskStarted(event)
	}
	return nil
}

// HandleTaskCompleted 处理任务完成事件
func (h *DefaultHandler) HandleTaskCompleted(event *CallbackEvent) error {
	if h.OnTaskCompleted != nil {
		return h.OnTaskCompleted(event)
	}
	return nil
}

// HandleTaskFailed 处理任务失败事件
func (h *DefaultHandler) HandleTaskFailed(event *CallbackEvent) error {
	if h.OnTaskFailed != nil {
		return h.OnTaskFailed(event)
	}
	return nil
}

// 辅助函数

// getRealIP 获取真实IP地址
func getRealIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// 检查X-Real-IP头
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 返回RemoteAddr，去掉端口号
	remoteAddr := r.RemoteAddr
	if colonIndex := strings.LastIndex(remoteAddr, ":"); colonIndex != -1 {
		return remoteAddr[:colonIndex]
	}
	return remoteAddr
}

// formatLogData 格式化日志数据
func formatLogData(data map[string]interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%+v", data)
	}
	return string(jsonData)
}

// copyInt64Map 复制int64映射
func copyInt64Map(original map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}