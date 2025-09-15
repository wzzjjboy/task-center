package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Version SDK版本信息
const Version = "1.0.0"

// Server HTTP回调服务器
type Server struct {
	mu          sync.RWMutex
	apiSecret   string
	handler     Handler
	middlewares []Middleware
	mux         *http.ServeMux
	validator   *Validator
	options     ServerOptions
}

// ServerOptions 服务器配置选项
type ServerOptions struct {
	// 路由配置
	WebhookPath string
	HealthPath  string

	// 安全配置
	EnableSignatureValidation bool
	TimestampToleranceSeconds int64

	// 性能配置
	MaxRequestBodySize int64
	RequestTimeout     time.Duration

	// 功能配置
	EnableHealthCheck bool
	EnableMetrics     bool
	EnableLogging     bool

	// 错误处理配置
	EnableGracefulError bool
	CustomErrorHandler  func(w http.ResponseWriter, r *http.Request, err error)
}

// ServerOption 服务器配置函数
type ServerOption func(*ServerOptions)

// WithWebhookPath 设置webhook路径
func WithWebhookPath(path string) ServerOption {
	return func(opts *ServerOptions) {
		opts.WebhookPath = path
	}
}

// WithHealthPath 设置健康检查路径
func WithHealthPath(path string) ServerOption {
	return func(opts *ServerOptions) {
		opts.HealthPath = path
	}
}

// WithSignatureValidation 启用签名验证
func WithSignatureValidation(enabled bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.EnableSignatureValidation = enabled
	}
}

// WithTimestampTolerance 设置时间戳容差
func WithTimestampTolerance(seconds int64) ServerOption {
	return func(opts *ServerOptions) {
		opts.TimestampToleranceSeconds = seconds
	}
}

// WithMaxRequestBodySize 设置最大请求体大小
func WithMaxRequestBodySize(size int64) ServerOption {
	return func(opts *ServerOptions) {
		opts.MaxRequestBodySize = size
	}
}

// WithRequestTimeout 设置请求超时时间
func WithRequestTimeout(timeout time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		opts.RequestTimeout = timeout
	}
}

// WithHealthCheck 启用健康检查
func WithHealthCheck(enabled bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.EnableHealthCheck = enabled
	}
}

// WithMetrics 启用指标收集
func WithMetrics(enabled bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.EnableMetrics = enabled
	}
}

// WithLogging 启用日志记录
func WithLogging(enabled bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.EnableLogging = enabled
	}
}

// WithGracefulError 启用优雅错误处理
func WithGracefulError(enabled bool) ServerOption {
	return func(opts *ServerOptions) {
		opts.EnableGracefulError = enabled
	}
}

// WithCustomErrorHandler 设置自定义错误处理器
func WithCustomErrorHandler(handler func(w http.ResponseWriter, r *http.Request, err error)) ServerOption {
	return func(opts *ServerOptions) {
		opts.CustomErrorHandler = handler
	}
}

// defaultServerOptions 默认服务器配置
func defaultServerOptions() ServerOptions {
	return ServerOptions{
		WebhookPath:               "/webhook",
		HealthPath:                "/health",
		EnableSignatureValidation: true,
		TimestampToleranceSeconds: 300, // 5分钟
		MaxRequestBodySize:        1024 * 1024, // 1MB
		RequestTimeout:            30 * time.Second,
		EnableHealthCheck:         true,
		EnableMetrics:             false,
		EnableLogging:             false,
		EnableGracefulError:       true,
	}
}

// NewServer 创建新的回调服务器
func NewServer(apiSecret string, handler Handler, opts ...ServerOption) *Server {
	options := defaultServerOptions()

	// 应用配置选项
	for _, opt := range opts {
		opt(&options)
	}

	server := &Server{
		apiSecret:   apiSecret,
		handler:     handler,
		middlewares: []Middleware{},
		mux:         http.NewServeMux(),
		validator:   NewValidator(apiSecret),
		options:     options,
	}

	// 注册路由
	server.setupRoutes()

	return server
}

// AddMiddleware 添加中间件
func (s *Server) AddMiddleware(middleware Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middlewares = append(s.middlewares, middleware)
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 注册webhook处理器
	s.mux.HandleFunc(s.options.WebhookPath, s.handleWebhook)

	// 注册健康检查处理器
	if s.options.EnableHealthCheck {
		s.mux.HandleFunc(s.options.HealthPath, s.handleHealth)
	}
}

// ServeHTTP 实现 http.Handler 接口
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置请求超时
	if s.options.RequestTimeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), s.options.RequestTimeout)
		defer cancel()
		r = r.WithContext(ctx)
	}

	s.mux.ServeHTTP(w, r)
}

// handleWebhook 处理webhook回调
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// 只接受POST请求
	if r.Method != http.MethodPost {
		s.handleError(w, r, NewMethodNotAllowedError("Only POST method is allowed"))
		return
	}

	// 限制请求体大小
	if s.options.MaxRequestBodySize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, s.options.MaxRequestBodySize)
	}

	// 执行前置中间件
	s.mu.RLock()
	middlewares := make([]Middleware, len(s.middlewares))
	copy(middlewares, s.middlewares)
	s.mu.RUnlock()

	for _, middleware := range middlewares {
		if err := middleware.Before(w, r); err != nil {
			s.handleError(w, r, err)
			return
		}
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.handleError(w, r, NewValidationError("Failed to read request body: "+err.Error()))
		return
	}
	defer r.Body.Close()

	// 验证签名
	if s.options.EnableSignatureValidation {
		if err := s.validator.ValidateSignature(r, body); err != nil {
			s.handleError(w, r, err)
			return
		}
	}

	// 解析回调事件
	event, err := s.validator.ParseEvent(body)
	if err != nil {
		s.handleError(w, r, err)
		return
	}

	// 调用业务处理器
	handlerErr := s.callEventHandler(event)
	if handlerErr != nil {
		s.handleError(w, r, handlerErr)
		return
	}

	// 执行后置中间件
	for _, middleware := range middlewares {
		if err := middleware.After(w, r, event); err != nil {
			// 后置中间件错误只记录，不影响响应
			if s.options.EnableLogging {
				// 这里可以添加日志记录
			}
		}
	}

	// 返回成功响应
	s.sendSuccessResponse(w, event)
}

// callEventHandler 调用对应的事件处理器
func (s *Server) callEventHandler(event *CallbackEvent) error {
	switch event.EventType {
	case "task.created":
		return s.handler.HandleTaskCreated(event)
	case "task.started":
		return s.handler.HandleTaskStarted(event)
	case "task.completed":
		return s.handler.HandleTaskCompleted(event)
	case "task.failed":
		return s.handler.HandleTaskFailed(event)
	default:
		return NewValidationError(fmt.Sprintf("Unknown event type: %s", event.EventType))
	}
}

// handleHealth 健康检查处理
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// 只接受GET请求
	if r.Method != http.MethodGet {
		s.handleError(w, r, NewMethodNotAllowedError("Only GET method is allowed for health check"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "TaskCenter Callback Server",
		"version":   Version,
	}

	json.NewEncoder(w).Encode(response)
}

// handleError 错误处理
func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	// 使用自定义错误处理器
	if s.options.CustomErrorHandler != nil {
		s.options.CustomErrorHandler(w, r, err)
		return
	}

	// 默认错误处理
	var statusCode int
	var errorCode string
	var message string

	if sdkErr, ok := err.(Error); ok {
		statusCode = sdkErr.StatusCode()
		errorCode = sdkErr.Code()
		message = sdkErr.Error()
	} else {
		statusCode = http.StatusInternalServerError
		errorCode = CodeServerError
		message = err.Error()

		// 在生产环境中，我们可能不想暴露内部错误详情
		if s.options.EnableGracefulError {
			message = "Internal server error occurred"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ErrorResponse{
		Success: false,
		Message: message,
		Code:    errorCode,
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// sendSuccessResponse 发送成功响应
func (s *Server) sendSuccessResponse(w http.ResponseWriter, event *CallbackEvent) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"success":    true,
		"message":    "Event processed successfully",
		"event_type": event.EventType,
		"task_id":    event.TaskID,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// Start 启动服务器
func (s *Server) Start(addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return server.ListenAndServe()
}

// StartTLS 启动HTTPS服务器
func (s *Server) StartTLS(addr, certFile, keyFile string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}

// StartWithServer 使用自定义服务器启动
func (s *Server) StartWithServer(server *http.Server) error {
	server.Handler = s
	return server.ListenAndServe()
}

// GetHandler 获取HTTP处理器（用于集成到现有的HTTP服务器）
func (s *Server) GetHandler() http.Handler {
	return s
}

// GetMux 获取路由多路复用器（用于注册到其他路由器）
func (s *Server) GetMux() *http.ServeMux {
	return s.mux
}

// UpdateOptions 更新服务器选项
func (s *Server) UpdateOptions(opts ...ServerOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, opt := range opts {
		opt(&s.options)
	}

	// 重新设置验证器选项
	s.validator.SetTimestampTolerance(s.options.TimestampToleranceSeconds)
}

// GetOptions 获取当前服务器选项
func (s *Server) GetOptions() ServerOptions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.options
}