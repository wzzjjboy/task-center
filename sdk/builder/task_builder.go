package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"task-center/sdk/task"
)

// TaskBuilder 任务构建器，提供链式调用接口
type TaskBuilder struct {
	client  *task.Client
	request *task.CreateRequest
	context context.Context
}

// NewTaskBuilder 创建新的任务构建器
func NewTaskBuilder(client *task.Client) *TaskBuilder {
	return &TaskBuilder{
		client:  client,
		request: task.NewCreateRequest(),
		context: context.Background(),
	}
}

// WithContext 设置上下文
func (b *TaskBuilder) WithContext(ctx context.Context) *TaskBuilder {
	b.context = ctx
	return b
}

// WithName 设置任务名称
func (b *TaskBuilder) WithName(name string) *TaskBuilder {
	b.request.Name = name
	return b
}

// WithType 设置任务类型
func (b *TaskBuilder) WithType(taskType string) *TaskBuilder {
	b.request.Type = taskType
	return b
}

// WithPayload 设置任务负载
func (b *TaskBuilder) WithPayload(payload interface{}) *TaskBuilder {
	if payload != nil {
		if payloadBytes, err := json.Marshal(payload); err == nil {
			b.request.Payload = payloadBytes
		}
	}
	return b
}

// WithPayloadJSON 设置JSON格式的任务负载
func (b *TaskBuilder) WithPayloadJSON(jsonStr string) *TaskBuilder {
	b.request.Payload = []byte(jsonStr)
	return b
}

// WithPriority 设置任务优先级
func (b *TaskBuilder) WithPriority(priority task.TaskPriority) *TaskBuilder {
	b.request.Priority = priority
	return b
}

// WithHighPriority 设置高优先级
func (b *TaskBuilder) WithHighPriority() *TaskBuilder {
	return b.WithPriority(task.PriorityHigh)
}

// WithNormalPriority 设置普通优先级
func (b *TaskBuilder) WithNormalPriority() *TaskBuilder {
	return b.WithPriority(task.PriorityNormal)
}

// WithLowPriority 设置低优先级
func (b *TaskBuilder) WithLowPriority() *TaskBuilder {
	return b.WithPriority(task.PriorityLow)
}

// WithScheduledTime 设置调度时间
func (b *TaskBuilder) WithScheduledTime(scheduledAt time.Time) *TaskBuilder {
	b.request.ScheduledAt = &scheduledAt
	return b
}

// WithDelay 设置延迟执行（从现在开始）
func (b *TaskBuilder) WithDelay(delay time.Duration) *TaskBuilder {
	scheduledAt := time.Now().Add(delay)
	b.request.ScheduledAt = &scheduledAt
	return b
}

// WithExpiration 设置任务过期时间
func (b *TaskBuilder) WithExpiration(expiredAt time.Time) *TaskBuilder {
	b.request.ExpiredAt = &expiredAt
	return b
}

// WithTimeout 设置任务超时时间
func (b *TaskBuilder) WithTimeout(timeout time.Duration) *TaskBuilder {
	timeoutSec := int(timeout.Seconds())
	b.request.TimeoutSeconds = &timeoutSec
	return b
}

// WithRetryPolicy 设置重试策略
func (b *TaskBuilder) WithRetryPolicy(maxRetries int, retryInterval time.Duration) *TaskBuilder {
	b.request.MaxRetries = &maxRetries
	intervalSec := int(retryInterval.Seconds())
	b.request.RetryInterval = &intervalSec
	return b
}

// WithMaxRetries 设置最大重试次数
func (b *TaskBuilder) WithMaxRetries(maxRetries int) *TaskBuilder {
	b.request.MaxRetries = &maxRetries
	return b
}

// WithRetryInterval 设置重试间隔
func (b *TaskBuilder) WithRetryInterval(interval time.Duration) *TaskBuilder {
	intervalSec := int(interval.Seconds())
	b.request.RetryInterval = &intervalSec
	return b
}

// WithCallback 设置回调URL
func (b *TaskBuilder) WithCallback(callbackURL string) *TaskBuilder {
	b.request.CallbackURL = &callbackURL
	return b
}

// WithCallbackHeaders 设置回调请求头
func (b *TaskBuilder) WithCallbackHeaders(headers map[string]string) *TaskBuilder {
	if headers != nil {
		if b.request.CallbackHeaders == nil {
			b.request.CallbackHeaders = make(map[string]string)
		}
		for k, v := range headers {
			b.request.CallbackHeaders[k] = v
		}
	}
	return b
}

// WithCallbackHeader 添加单个回调请求头
func (b *TaskBuilder) WithCallbackHeader(key, value string) *TaskBuilder {
	if b.request.CallbackHeaders == nil {
		b.request.CallbackHeaders = make(map[string]string)
	}
	b.request.CallbackHeaders[key] = value
	return b
}

// WithTags 设置任务标签
func (b *TaskBuilder) WithTags(tags ...string) *TaskBuilder {
	b.request.Tags = append(b.request.Tags, tags...)
	return b
}

// WithTag 添加单个标签
func (b *TaskBuilder) WithTag(tag string) *TaskBuilder {
	return b.WithTags(tag)
}

// WithBusinessUniqueID 设置业务唯一ID
func (b *TaskBuilder) WithBusinessUniqueID(businessUniqueID string) *TaskBuilder {
	b.request.BusinessUniqueID = &businessUniqueID
	return b
}

// WithDescription 设置任务描述
func (b *TaskBuilder) WithDescription(description string) *TaskBuilder {
	b.request.Description = &description
	return b
}

// Build 构建并创建任务
func (b *TaskBuilder) Build() (*task.Task, error) {
	return b.client.CreateTask(b.context, b.request)
}

// BuildAsync 异步构建并创建任务
func (b *TaskBuilder) BuildAsync() (<-chan *BuildResult, error) {
	resultChan := make(chan *BuildResult, 1)

	go func() {
		defer close(resultChan)

		result, err := b.Build()
		resultChan <- &BuildResult{
			Task:  result,
			Error: err,
		}
	}()

	return resultChan, nil
}

// BuildResult 构建结果
type BuildResult struct {
	Task  *task.Task
	Error error
}

// Clone 克隆构建器（深拷贝）
func (b *TaskBuilder) Clone() *TaskBuilder {
	newBuilder := &TaskBuilder{
		client:  b.client,
		request: task.NewCreateRequest(),
		context: b.context,
	}

	// 深拷贝请求对象
	if b.request.Name != "" {
		newBuilder.request.Name = b.request.Name
	}
	if b.request.Type != "" {
		newBuilder.request.Type = b.request.Type
	}
	if b.request.Payload != nil {
		newBuilder.request.Payload = make([]byte, len(b.request.Payload))
		copy(newBuilder.request.Payload, b.request.Payload)
	}
	newBuilder.request.Priority = b.request.Priority
	if b.request.ScheduledAt != nil {
		scheduledAt := *b.request.ScheduledAt
		newBuilder.request.ScheduledAt = &scheduledAt
	}
	if b.request.ExpiredAt != nil {
		expiredAt := *b.request.ExpiredAt
		newBuilder.request.ExpiredAt = &expiredAt
	}
	if b.request.TimeoutSeconds != nil {
		timeout := *b.request.TimeoutSeconds
		newBuilder.request.TimeoutSeconds = &timeout
	}
	if b.request.MaxRetries != nil {
		maxRetries := *b.request.MaxRetries
		newBuilder.request.MaxRetries = &maxRetries
	}
	if b.request.RetryInterval != nil {
		retryInterval := *b.request.RetryInterval
		newBuilder.request.RetryInterval = &retryInterval
	}
	if b.request.CallbackURL != nil {
		callbackURL := *b.request.CallbackURL
		newBuilder.request.CallbackURL = &callbackURL
	}
	if b.request.CallbackHeaders != nil {
		newBuilder.request.CallbackHeaders = make(map[string]string)
		for k, v := range b.request.CallbackHeaders {
			newBuilder.request.CallbackHeaders[k] = v
		}
	}
	if b.request.Tags != nil {
		newBuilder.request.Tags = make([]string, len(b.request.Tags))
		copy(newBuilder.request.Tags, b.request.Tags)
	}
	if b.request.BusinessUniqueID != nil {
		businessUniqueID := *b.request.BusinessUniqueID
		newBuilder.request.BusinessUniqueID = &businessUniqueID
	}
	if b.request.Description != nil {
		description := *b.request.Description
		newBuilder.request.Description = &description
	}

	return newBuilder
}

// Reset 重置构建器
func (b *TaskBuilder) Reset() *TaskBuilder {
	b.request = task.NewCreateRequest()
	b.context = context.Background()
	return b
}

// GetRequest 获取构建的请求对象（用于调试）
func (b *TaskBuilder) GetRequest() *task.CreateRequest {
	return b.request
}

// Validate 验证构建器设置
func (b *TaskBuilder) Validate() error {
	return b.request.Validate()
}

// Template 任务模板构建器
type Template struct {
	name            string
	taskType        string
	priority        task.TaskPriority
	timeout         *time.Duration
	maxRetries      *int
	retryInterval   *time.Duration
	callbackURL     *string
	callbackHeaders map[string]string
	tags            []string
	description     *string
}

// NewTemplate 创建任务模板
func NewTemplate(name, taskType string) *Template {
	return &Template{
		name:     name,
		taskType: taskType,
		priority: task.PriorityNormal,
	}
}

// SetPriority 设置模板优先级
func (t *Template) SetPriority(priority task.TaskPriority) *Template {
	t.priority = priority
	return t
}

// SetTimeout 设置模板超时时间
func (t *Template) SetTimeout(timeout time.Duration) *Template {
	t.timeout = &timeout
	return t
}

// SetRetryPolicy 设置模板重试策略
func (t *Template) SetRetryPolicy(maxRetries int, retryInterval time.Duration) *Template {
	t.maxRetries = &maxRetries
	t.retryInterval = &retryInterval
	return t
}

// SetCallback 设置模板回调配置
func (t *Template) SetCallback(callbackURL string, headers map[string]string) *Template {
	t.callbackURL = &callbackURL
	t.callbackHeaders = headers
	return t
}

// SetTags 设置模板标签
func (t *Template) SetTags(tags ...string) *Template {
	t.tags = tags
	return t
}

// SetDescription 设置模板描述
func (t *Template) SetDescription(description string) *Template {
	t.description = &description
	return t
}

// Apply 将模板应用到构建器
func (t *Template) Apply(builder *TaskBuilder) *TaskBuilder {
	builder.WithName(t.name).WithType(t.taskType).WithPriority(t.priority)

	if t.timeout != nil {
		builder.WithTimeout(*t.timeout)
	}
	if t.maxRetries != nil && t.retryInterval != nil {
		builder.WithRetryPolicy(*t.maxRetries, *t.retryInterval)
	}
	if t.callbackURL != nil {
		builder.WithCallback(*t.callbackURL)
		if t.callbackHeaders != nil {
			builder.WithCallbackHeaders(t.callbackHeaders)
		}
	}
	if t.tags != nil {
		builder.WithTags(t.tags...)
	}
	if t.description != nil {
		builder.WithDescription(*t.description)
	}

	return builder
}

// CreateBuilder 基于模板创建构建器
func (t *Template) CreateBuilder(client *task.Client) *TaskBuilder {
	builder := NewTaskBuilder(client)
	return t.Apply(builder)
}

// 常用模板预定义
var (
	// EmailTemplate 邮件发送任务模板
	EmailTemplate = NewTemplate("send_email", "email").
			SetPriority(task.PriorityNormal).
			SetTimeout(30*time.Second).
			SetRetryPolicy(3, 5*time.Second).
			SetTags("email", "notification")

	// ReportTemplate 报表生成任务模板
	ReportTemplate = NewTemplate("generate_report", "report").
			   SetPriority(task.PriorityLow).
			   SetTimeout(5*time.Minute).
			   SetRetryPolicy(2, 10*time.Second).
			   SetTags("report", "background")

	// PaymentTemplate 支付处理任务模板
	PaymentTemplate = NewTemplate("process_payment", "payment").
			  SetPriority(task.PriorityHigh).
			  SetTimeout(10*time.Second).
			  SetRetryPolicy(5, 2*time.Second).
			  SetTags("payment", "critical")
)

// QuickBuilder 便捷构建器方法集
type QuickBuilder struct {
	client *task.Client
}

// NewQuickBuilder 创建便捷构建器
func NewQuickBuilder(client *task.Client) *QuickBuilder {
	return &QuickBuilder{client: client}
}

// Email 创建邮件任务
func (q *QuickBuilder) Email(to, subject, body string) *TaskBuilder {
	payload := map[string]string{
		"to":      to,
		"subject": subject,
		"body":    body,
	}
	return EmailTemplate.CreateBuilder(q.client).WithPayload(payload)
}

// Report 创建报表任务
func (q *QuickBuilder) Report(reportType string, params map[string]interface{}) *TaskBuilder {
	payload := map[string]interface{}{
		"type":   reportType,
		"params": params,
	}
	return ReportTemplate.CreateBuilder(q.client).WithPayload(payload)
}

// Payment 创建支付任务
func (q *QuickBuilder) Payment(orderID string, amount float64, currency string) *TaskBuilder {
	payload := map[string]interface{}{
		"order_id": orderID,
		"amount":   amount,
		"currency": currency,
	}
	return PaymentTemplate.CreateBuilder(q.client).WithPayload(payload)
}

// Simple 创建简单任务
func (q *QuickBuilder) Simple(name, taskType string, payload interface{}) *TaskBuilder {
	return NewTaskBuilder(q.client).
		WithName(name).
		WithType(taskType).
		WithPayload(payload)
}

// Delayed 创建延迟任务
func (q *QuickBuilder) Delayed(name, taskType string, payload interface{}, delay time.Duration) *TaskBuilder {
	return NewTaskBuilder(q.client).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithDelay(delay)
}

// Scheduled 创建定时任务
func (q *QuickBuilder) Scheduled(name, taskType string, payload interface{}, scheduledAt time.Time) *TaskBuilder {
	return NewTaskBuilder(q.client).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithScheduledTime(scheduledAt)
}