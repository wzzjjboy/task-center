package sdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// CallbackHandler 回调处理器接口
type CallbackHandler interface {
	HandleTaskCreated(event *CallbackEvent) error
	HandleTaskStarted(event *CallbackEvent) error
	HandleTaskCompleted(event *CallbackEvent) error
	HandleTaskFailed(event *CallbackEvent) error
}

// CallbackMiddleware 回调中间件接口
type CallbackMiddleware interface {
	Before(w http.ResponseWriter, r *http.Request) error
	After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error
}

// CallbackServer 回调服务器
type CallbackServer struct {
	apiSecret   string
	handler     CallbackHandler
	middlewares []CallbackMiddleware
	mux         *http.ServeMux
}

// CallbackServerOption 回调服务器配置选项
type CallbackServerOption func(*CallbackServer)

// WithCallbackMiddleware 添加回调中间件
func WithCallbackMiddleware(middleware CallbackMiddleware) CallbackServerOption {
	return func(s *CallbackServer) {
		s.middlewares = append(s.middlewares, middleware)
	}
}

// NewCallbackServer 创建回调服务器
func NewCallbackServer(apiSecret string, handler CallbackHandler, opts ...CallbackServerOption) *CallbackServer {
	server := &CallbackServer{
		apiSecret: apiSecret,
		handler:   handler,
		mux:       http.NewServeMux(),
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(server)
	}

	// 注册回调路由
	server.mux.HandleFunc("/webhook", server.handleWebhook)
	server.mux.HandleFunc("/health", server.handleHealth)

	return server
}

// ServeHTTP 实现 http.Handler 接口
func (s *CallbackServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleWebhook 处理webhook回调
func (s *CallbackServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// 只接受POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 执行前置中间件
	for _, middleware := range s.middlewares {
		if err := middleware.Before(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 验证签名
	if !s.verifySignature(r, body) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// 解析回调事件
	var event CallbackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 根据事件类型调用对应的处理方法
	var handlerErr error
	switch event.EventType {
	case "task.created":
		handlerErr = s.handler.HandleTaskCreated(&event)
	case "task.started":
		handlerErr = s.handler.HandleTaskStarted(&event)
	case "task.completed":
		handlerErr = s.handler.HandleTaskCompleted(&event)
	case "task.failed":
		handlerErr = s.handler.HandleTaskFailed(&event)
	default:
		http.Error(w, fmt.Sprintf("Unknown event type: %s", event.EventType), http.StatusBadRequest)
		return
	}

	// 处理错误
	if handlerErr != nil {
		http.Error(w, handlerErr.Error(), http.StatusInternalServerError)
		return
	}

	// 执行后置中间件
	for _, middleware := range s.middlewares {
		if err := middleware.After(w, r, &event); err != nil {
			// 后置中间件错误只记录，不影响响应
			// 这里可以添加日志记录
		}
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Event processed successfully",
	})
}

// handleHealth 健康检查处理
func (s *CallbackServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// verifySignature 验证请求签名
func (s *CallbackServer) verifySignature(r *http.Request, body []byte) bool {
	// 获取签名头
	signature := r.Header.Get("X-TaskCenter-Signature")
	if signature == "" {
		return false
	}

	// 获取时间戳头
	timestampStr := r.Header.Get("X-TaskCenter-Timestamp")
	if timestampStr == "" {
		return false
	}

	// 验证时间戳（防止重放攻击）
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	// 检查时间戳是否在允许范围内（5分钟）
	now := time.Now().Unix()
	if abs(now-timestamp) > 300 {
		return false
	}

	// 计算期望的签名
	expectedSignature := s.calculateSignature(timestampStr, body)

	// 比较签名
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// calculateSignature 计算签名
func (s *CallbackServer) calculateSignature(timestamp string, body []byte) string {
	// 构建签名字符串：timestamp + "." + body
	signatureString := timestamp + "." + string(body)

	// 使用HMAC-SHA256计算签名
	h := hmac.New(sha256.New, []byte(s.apiSecret))
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))

	return "sha256=" + signature
}

// abs 绝对值函数
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// 预定义的中间件实现

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	Logger func(level string, message string, fields map[string]interface{})
}

// Before 前置处理
func (m *LoggingMiddleware) Before(w http.ResponseWriter, r *http.Request) error {
	if m.Logger != nil {
		m.Logger("info", "Webhook request received", map[string]interface{}{
			"method":     r.Method,
			"url":        r.URL.String(),
			"user_agent": r.Header.Get("User-Agent"),
			"remote_ip":  r.RemoteAddr,
		})
	}
	return nil
}

// After 后置处理
func (m *LoggingMiddleware) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	if m.Logger != nil {
		m.Logger("info", "Webhook event processed", map[string]interface{}{
			"event_type": event.EventType,
			"task_id":    event.TaskID,
			"event_time": event.EventTime,
		})
	}
	return nil
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct {
	IncCounter func(name string, labels map[string]string)
	RecordTime func(name string, duration time.Duration, labels map[string]string)
}

// Before 前置处理
func (m *MetricsMiddleware) Before(w http.ResponseWriter, r *http.Request) error {
	// 在请求上下文中记录开始时间
	r = r.WithContext(r.Context())
	return nil
}

// After 后置处理
func (m *MetricsMiddleware) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	if m.IncCounter != nil {
		m.IncCounter("webhook_events_total", map[string]string{
			"event_type": event.EventType,
		})
	}
	return nil
}

// DefaultCallbackHandler 默认回调处理器实现
type DefaultCallbackHandler struct {
	OnTaskCreated   func(*CallbackEvent) error
	OnTaskStarted   func(*CallbackEvent) error
	OnTaskCompleted func(*CallbackEvent) error
	OnTaskFailed    func(*CallbackEvent) error
}

// HandleTaskCreated 处理任务创建事件
func (h *DefaultCallbackHandler) HandleTaskCreated(event *CallbackEvent) error {
	if h.OnTaskCreated != nil {
		return h.OnTaskCreated(event)
	}
	return nil
}

// HandleTaskStarted 处理任务开始事件
func (h *DefaultCallbackHandler) HandleTaskStarted(event *CallbackEvent) error {
	if h.OnTaskStarted != nil {
		return h.OnTaskStarted(event)
	}
	return nil
}

// HandleTaskCompleted 处理任务完成事件
func (h *DefaultCallbackHandler) HandleTaskCompleted(event *CallbackEvent) error {
	if h.OnTaskCompleted != nil {
		return h.OnTaskCompleted(event)
	}
	return nil
}

// HandleTaskFailed 处理任务失败事件
func (h *DefaultCallbackHandler) HandleTaskFailed(event *CallbackEvent) error {
	if h.OnTaskFailed != nil {
		return h.OnTaskFailed(event)
	}
	return nil
}