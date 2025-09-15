package sdk

import "time"

// SDK 版本信息
const (
	Version = "1.0.0"
	Name    = "TaskCenter Go SDK"
)

// 预定义的常量
const (
	DefaultTimeout      = 30 * time.Second
	DefaultMaxRetries   = 3
	DefaultPageSize     = 20
	DefaultUserAgent    = "TaskCenter-Go-SDK/" + Version
)

// 便捷的构造函数

// NewTask 创建新任务
func NewTask(businessUniqueID, callbackURL string) *CreateTaskRequest {
	return &CreateTaskRequest{
		BusinessUniqueID: businessUniqueID,
		CallbackURL:      callbackURL,
		CallbackMethod:   "POST",
		Priority:         TaskPriorityNormal,
		Timeout:          300, // 5分钟
		MaxRetries:       3,
	}
}

// NewTaskWithCallback 创建带回调配置的任务
func NewTaskWithCallback(businessUniqueID, callbackURL, method string, headers map[string]string, body string) *CreateTaskRequest {
	req := NewTask(businessUniqueID, callbackURL)
	req.CallbackMethod = method
	req.CallbackHeaders = headers
	req.CallbackBody = body
	return req
}

// NewScheduledTask 创建定时任务
func NewScheduledTask(businessUniqueID, callbackURL string, scheduledAt time.Time) *CreateTaskRequest {
	req := NewTask(businessUniqueID, callbackURL)
	req.ScheduledAt = &scheduledAt
	return req
}

// 链式调用方法

// WithPriority 设置任务优先级
func (req *CreateTaskRequest) WithPriority(priority TaskPriority) *CreateTaskRequest {
	req.Priority = priority
	return req
}

// WithTimeout 设置任务超时
func (req *CreateTaskRequest) WithTimeout(timeout int) *CreateTaskRequest {
	req.Timeout = timeout
	return req
}

// WithRetries 设置重试配置
func (req *CreateTaskRequest) WithRetries(maxRetries int, intervals []int) *CreateTaskRequest {
	req.MaxRetries = maxRetries
	req.RetryIntervals = intervals
	return req
}

// WithTags 设置任务标签
func (req *CreateTaskRequest) WithTags(tags ...string) *CreateTaskRequest {
	req.Tags = tags
	return req
}

// WithMetadata 设置元数据
func (req *CreateTaskRequest) WithMetadata(metadata map[string]interface{}) *CreateTaskRequest {
	req.Metadata = metadata
	return req
}

// WithHeaders 设置回调请求头
func (req *CreateTaskRequest) WithHeaders(headers map[string]string) *CreateTaskRequest {
	req.CallbackHeaders = headers
	return req
}

// WithBody 设置回调请求体
func (req *CreateTaskRequest) WithBody(body string) *CreateTaskRequest {
	req.CallbackBody = body
	return req
}

// WithSchedule 设置计划执行时间
func (req *CreateTaskRequest) WithSchedule(scheduledAt time.Time) *CreateTaskRequest {
	req.ScheduledAt = &scheduledAt
	return req
}

// 便捷的查询构造器

// NewListTasksRequest 创建任务列表查询请求
func NewListTasksRequest() *ListTasksRequest {
	return &ListTasksRequest{
		Page:     1,
		PageSize: DefaultPageSize,
	}
}

// WithStatus 过滤任务状态
func (req *ListTasksRequest) WithStatus(status ...TaskStatus) *ListTasksRequest {
	req.Status = status
	return req
}

// WithTagsFilter 过滤标签
func (req *ListTasksRequest) WithTagsFilter(tags ...string) *ListTasksRequest {
	req.Tags = tags
	return req
}

// WithPriorityFilter 过滤优先级
func (req *ListTasksRequest) WithPriorityFilter(priority TaskPriority) *ListTasksRequest {
	req.Priority = &priority
	return req
}

// WithDateRange 设置日期范围
func (req *ListTasksRequest) WithDateRange(from, to time.Time) *ListTasksRequest {
	req.CreatedFrom = &from
	req.CreatedTo = &to
	return req
}

// WithPagination 设置分页参数
func (req *ListTasksRequest) WithPagination(page, pageSize int) *ListTasksRequest {
	req.Page = page
	req.PageSize = pageSize
	return req
}

// 预定义的重试间隔配置

// StandardRetryIntervals 标准重试间隔 (1分钟, 5分钟, 15分钟)
var StandardRetryIntervals = []int{60, 300, 900}

// FastRetryIntervals 快速重试间隔 (10秒, 30秒, 60秒)
var FastRetryIntervals = []int{10, 30, 60}

// SlowRetryIntervals 慢速重试间隔 (5分钟, 30分钟, 2小时)
var SlowRetryIntervals = []int{300, 1800, 7200}

// ExponentialRetryIntervals 指数重试间隔 (2^n 分钟)
var ExponentialRetryIntervals = []int{60, 120, 240, 480, 960}

// 便捷的回调处理器

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

// 工具函数

// IsTaskActive 检查任务是否处于活跃状态
func IsTaskActive(status TaskStatus) bool {
	return status == TaskStatusPending || status == TaskStatusRunning
}

// IsTaskCompleted 检查任务是否已完成（成功或失败）
func IsTaskCompleted(status TaskStatus) bool {
	return status == TaskStatusSucceeded ||
		status == TaskStatusFailed ||
		status == TaskStatusCancelled ||
		status == TaskStatusExpired
}

// IsTaskSuccessful 检查任务是否成功完成
func IsTaskSuccessful(status TaskStatus) bool {
	return status == TaskStatusSucceeded
}

// CalculateRetryDelay 计算重试延迟时间
func CalculateRetryDelay(currentRetry int, intervals []int) time.Duration {
	if currentRetry >= len(intervals) {
		// 如果重试次数超过配置的间隔数组，使用最后一个间隔
		return time.Duration(intervals[len(intervals)-1]) * time.Second
	}
	return time.Duration(intervals[currentRetry]) * time.Second
}

// FormatTaskID 格式化任务ID用于显示
func FormatTaskID(businessUniqueID string, taskID int64) string {
	return businessUniqueID + "#" + string(rune(taskID))
}