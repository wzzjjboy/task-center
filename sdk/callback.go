package sdk

import (
	"log"
	"net/http"
	"time"

	"task-center/sdk/callback"
)

// 重新导出callback包中的关键类型和接口，提供向后兼容性

// CallbackHandler 回调处理器接口（向后兼容）
type CallbackHandler interface {
	HandleTaskCreated(event *CallbackEvent) error
	HandleTaskStarted(event *CallbackEvent) error
	HandleTaskCompleted(event *CallbackEvent) error
	HandleTaskFailed(event *CallbackEvent) error
}

// CallbackMiddleware 回调中间件接口（向后兼容）
type CallbackMiddleware interface {
	Before(w http.ResponseWriter, r *http.Request) error
	After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error
}

// 类型别名，提供向后兼容性
type (
	// Server 回调服务器
	CallbackServer = callback.Server

	// Middleware 中间件接口
	Middleware = callback.Middleware

	// Handler 处理器接口
	Handler = callback.Handler

	// Validator 验证器
	Validator = callback.Validator

	// 中间件实现
	LoggingMiddleware   = callback.LoggingMiddleware
	MetricsMiddleware   = callback.MetricsMiddleware
	SecurityMiddleware  = callback.SecurityMiddleware
	DefaultHandler      = callback.DefaultHandler
)

// 便捷的构造函数和选项

// NewCallbackServer 创建回调服务器（向后兼容）
func NewCallbackServer(apiSecret string, handler CallbackHandler, opts ...CallbackServerOption) *CallbackServer {
	// 创建适配器将旧接口转换为新接口
	adapterHandler := &callbackHandlerAdapter{handler: handler}

	// 转换选项
	var serverOpts []callback.ServerOption
	for _, opt := range opts {
		if middlewareOpt, ok := opt.(*middlewareOption); ok {
			// 转换中间件选项
			serverOpts = append(serverOpts, callback.WithMiddleware(&middlewareAdapter{middleware: middlewareOpt.middleware}))
		}
	}

	return callback.NewServer(apiSecret, adapterHandler, serverOpts...)
}

// CallbackServerOption 回调服务器配置选项（向后兼容）
type CallbackServerOption interface {
	apply()
}

// middlewareOption 中间件选项
type middlewareOption struct {
	middleware CallbackMiddleware
}

func (m *middlewareOption) apply() {}

// WithCallbackMiddleware 添加回调中间件（向后兼容）
func WithCallbackMiddleware(middleware CallbackMiddleware) CallbackServerOption {
	return &middlewareOption{middleware: middleware}
}

// callbackHandlerAdapter 处理器适配器，将旧接口适配到新接口
type callbackHandlerAdapter struct {
	handler CallbackHandler
}

func (a *callbackHandlerAdapter) HandleTaskCreated(event *CallbackEvent) error {
	return a.handler.HandleTaskCreated(event)
}

func (a *callbackHandlerAdapter) HandleTaskStarted(event *CallbackEvent) error {
	return a.handler.HandleTaskStarted(event)
}

func (a *callbackHandlerAdapter) HandleTaskCompleted(event *CallbackEvent) error {
	return a.handler.HandleTaskCompleted(event)
}

func (a *callbackHandlerAdapter) HandleTaskFailed(event *CallbackEvent) error {
	return a.handler.HandleTaskFailed(event)
}

// middlewareAdapter 中间件适配器，将旧接口适配到新接口
type middlewareAdapter struct {
	middleware CallbackMiddleware
}

func (a *middlewareAdapter) Before(w http.ResponseWriter, r *http.Request) error {
	return a.middleware.Before(w, r)
}

func (a *middlewareAdapter) After(w http.ResponseWriter, r *http.Request, event *CallbackEvent) error {
	return a.middleware.After(w, r, event)
}

// 新的便捷构造函数

// NewCallbackServerV2 创建新版本的回调服务器
func NewCallbackServerV2(apiSecret string, handler callback.Handler, opts ...callback.ServerOption) *callback.Server {
	return callback.NewServer(apiSecret, handler, opts...)
}

// NewCallbackValidator 创建回调验证器
func NewCallbackValidator(apiSecret string, opts ...callback.ValidatorOption) *callback.Validator {
	return callback.NewValidator(apiSecret, opts...)
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(opts ...callback.LoggingMiddlewareOption) *callback.LoggingMiddleware {
	return callback.NewLoggingMiddleware(opts...)
}

// NewMetricsMiddleware 创建指标中间件
func NewMetricsMiddleware() *callback.MetricsMiddleware {
	return callback.NewMetricsMiddleware()
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware() *callback.SecurityMiddleware {
	return callback.NewSecurityMiddleware()
}

// NewDefaultHandler 创建默认处理器
func NewDefaultHandler() *callback.DefaultHandler {
	return &callback.DefaultHandler{}
}

// 便捷的构造函数（兼容原有代码）

// SimpleCallbackHandler 简单回调处理器
func SimpleCallbackHandler(
	onCreated func(task *Task),
	onStarted func(task *Task),
	onCompleted func(task *Task),
	onFailed func(task *Task, errorMsg string),
) CallbackHandler {
	return &DefaultCallbackHandler{
		OnTaskCreated: func(event *CallbackEvent) error {
			if onCreated != nil {
				onCreated(&event.Task)
			}
			return nil
		},
		OnTaskStarted: func(event *CallbackEvent) error {
			if onStarted != nil {
				onStarted(&event.Task)
			}
			return nil
		},
		OnTaskCompleted: func(event *CallbackEvent) error {
			if onCompleted != nil {
				onCompleted(&event.Task)
			}
			return nil
		},
		OnTaskFailed: func(event *CallbackEvent) error {
			if onFailed != nil {
				onFailed(&event.Task, event.Task.ErrorMessage)
			}
			return nil
		},
	}
}

// DefaultCallbackHandler 默认回调处理器实现（向后兼容）
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

// 重新导出中间件实现（向后兼容）

// LoggingMiddleware 日志中间件（向后兼容）
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

// MetricsMiddleware 指标中间件（向后兼容）
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

// 工具函数

// CreateTestServer 创建测试用的回调服务器
func CreateTestServer(apiSecret string, handler CallbackHandler) *CallbackServer {
	return NewCallbackServer(apiSecret, handler)
}

// CreateProductionServer 创建生产环境的回调服务器
func CreateProductionServer(apiSecret string, handler CallbackHandler, logger *log.Logger) *CallbackServer {
	loggingMiddleware := &LoggingMiddleware{
		Logger: func(level, message string, fields map[string]interface{}) {
			if logger != nil {
				logger.Printf("[%s] %s: %+v", level, message, fields)
			}
		},
	}

	metricsMiddleware := &MetricsMiddleware{
		IncCounter: func(name string, labels map[string]string) {
			// 这里可以集成实际的指标系统
		},
	}

	return NewCallbackServer(apiSecret, handler,
		WithCallbackMiddleware(loggingMiddleware),
		WithCallbackMiddleware(metricsMiddleware))
}

// ValidateCallbackSignature 验证回调签名（独立函数）
func ValidateCallbackSignature(apiSecret string, r *http.Request, body []byte) error {
	validator := callback.NewValidator(apiSecret)
	return validator.ValidateSignature(r, body)
}

// ParseCallbackEvent 解析回调事件（独立函数）
func ParseCallbackEvent(body []byte) (*CallbackEvent, error) {
	validator := callback.NewValidator("") // 只用于解析，不需要密钥
	return validator.ParseEvent(body)
}