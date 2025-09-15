package task

import (
	"encoding/json"
	"time"

	"task-center/sdk"
)

// TaskStatus 任务状态枚举
type TaskStatus = sdk.TaskStatus

// TaskPriority 任务优先级
type TaskPriority = sdk.TaskPriority

// 复用 SDK 包中的常量定义
const (
	StatusPending   = sdk.TaskStatusPending
	StatusRunning   = sdk.TaskStatusRunning
	StatusSucceeded = sdk.TaskStatusSucceeded
	StatusFailed    = sdk.TaskStatusFailed
	StatusCancelled = sdk.TaskStatusCancelled
	StatusExpired   = sdk.TaskStatusExpired

	PriorityHighest = sdk.TaskPriorityHighest
	PriorityHigh    = sdk.TaskPriorityHigh
	PriorityNormal  = sdk.TaskPriorityNormal
	PriorityLow     = sdk.TaskPriorityLow
	PriorityLowest  = sdk.TaskPriorityLowest
)

// Task 任务结构，继承自 SDK 基础类型
type Task struct {
	*sdk.Task
}

// NewTask 创建新任务实例
func NewTask() *Task {
	return &Task{
		Task: &sdk.Task{},
	}
}

// NewTaskFromSDK 从 SDK Task 创建
func NewTaskFromSDK(sdkTask *sdk.Task) *Task {
	return &Task{Task: sdkTask}
}

// ToSDK 转换为 SDK Task
func (t *Task) ToSDK() *sdk.Task {
	return t.Task
}

// CreateRequest 创建任务请求结构
type CreateRequest struct {
	*sdk.CreateTaskRequest
}

// NewCreateRequest 创建新的任务创建请求
func NewCreateRequest(businessUniqueID, callbackURL string) *CreateRequest {
	return &CreateRequest{
		CreateTaskRequest: &sdk.CreateTaskRequest{
			BusinessUniqueID: businessUniqueID,
			CallbackURL:      callbackURL,
			CallbackMethod:   "POST",
			Priority:         PriorityNormal,
			Timeout:          300,
			MaxRetries:       3,
		},
	}
}

// WithMethod 设置回调方法
func (r *CreateRequest) WithMethod(method string) *CreateRequest {
	r.CallbackMethod = method
	return r
}

// WithHeaders 设置回调头
func (r *CreateRequest) WithHeaders(headers map[string]string) *CreateRequest {
	r.CallbackHeaders = headers
	return r
}

// WithBody 设置回调体
func (r *CreateRequest) WithBody(body string) *CreateRequest {
	r.CallbackBody = body
	return r
}

// WithRetry 设置重试配置
func (r *CreateRequest) WithRetry(maxRetries int, intervals ...int) *CreateRequest {
	r.MaxRetries = maxRetries
	if len(intervals) > 0 {
		r.RetryIntervals = intervals
	}
	return r
}

// WithPriority 设置优先级
func (r *CreateRequest) WithPriority(priority TaskPriority) *CreateRequest {
	r.Priority = priority
	return r
}

// WithTags 设置标签
func (r *CreateRequest) WithTags(tags ...string) *CreateRequest {
	r.Tags = tags
	return r
}

// WithTimeout 设置超时时间（秒）
func (r *CreateRequest) WithTimeout(timeout int) *CreateRequest {
	r.Timeout = timeout
	return r
}

// WithScheduledAt 设置计划执行时间
func (r *CreateRequest) WithScheduledAt(t time.Time) *CreateRequest {
	r.ScheduledAt = &t
	return r
}

// WithMetadata 设置元数据
func (r *CreateRequest) WithMetadata(metadata map[string]interface{}) *CreateRequest {
	r.Metadata = metadata
	return r
}

// UpdateRequest 更新任务请求结构
type UpdateRequest struct {
	*sdk.UpdateTaskRequest
}

// NewUpdateRequest 创建新的任务更新请求
func NewUpdateRequest() *UpdateRequest {
	return &UpdateRequest{
		UpdateTaskRequest: &sdk.UpdateTaskRequest{},
	}
}

// WithCallbackURL 设置回调URL
func (r *UpdateRequest) WithCallbackURL(url string) *UpdateRequest {
	r.CallbackURL = &url
	return r
}

// WithCallbackMethod 设置回调方法
func (r *UpdateRequest) WithCallbackMethod(method string) *UpdateRequest {
	r.CallbackMethod = &method
	return r
}

// WithCallbackHeaders 设置回调头
func (r *UpdateRequest) WithCallbackHeaders(headers map[string]string) *UpdateRequest {
	r.CallbackHeaders = headers
	return r
}

// WithCallbackBody 设置回调体
func (r *UpdateRequest) WithCallbackBody(body string) *UpdateRequest {
	r.CallbackBody = &body
	return r
}

// WithRetry 设置重试配置
func (r *UpdateRequest) WithRetry(maxRetries int, intervals ...int) *UpdateRequest {
	r.MaxRetries = &maxRetries
	if len(intervals) > 0 {
		r.RetryIntervals = intervals
	}
	return r
}

// WithPriority 设置优先级
func (r *UpdateRequest) WithPriority(priority TaskPriority) *UpdateRequest {
	r.Priority = &priority
	return r
}

// WithTags 设置标签
func (r *UpdateRequest) WithTags(tags ...string) *UpdateRequest {
	r.Tags = tags
	return r
}

// WithTimeout 设置超时时间（秒）
func (r *UpdateRequest) WithTimeout(timeout int) *UpdateRequest {
	r.Timeout = &timeout
	return r
}

// WithScheduledAt 设置计划执行时间
func (r *UpdateRequest) WithScheduledAt(t time.Time) *UpdateRequest {
	r.ScheduledAt = &t
	return r
}

// WithStatus 设置任务状态
func (r *UpdateRequest) WithStatus(status TaskStatus) *UpdateRequest {
	r.Status = &status
	return r
}

// WithMetadata 设置元数据
func (r *UpdateRequest) WithMetadata(metadata map[string]interface{}) *UpdateRequest {
	r.Metadata = metadata
	return r
}

// ListRequest 查询任务列表请求
type ListRequest struct {
	*sdk.ListTasksRequest
}

// NewListRequest 创建新的任务列表请求
func NewListRequest() *ListRequest {
	return &ListRequest{
		ListTasksRequest: &sdk.ListTasksRequest{
			Page:     1,
			PageSize: 20,
		},
	}
}

// WithStatus 过滤任务状态
func (r *ListRequest) WithStatus(status ...TaskStatus) *ListRequest {
	r.Status = status
	return r
}

// WithTags 过滤标签
func (r *ListRequest) WithTags(tags ...string) *ListRequest {
	r.Tags = tags
	return r
}

// WithPriority 过滤优先级
func (r *ListRequest) WithPriority(priority TaskPriority) *ListRequest {
	r.Priority = &priority
	return r
}

// WithTimeRange 设置时间范围
func (r *ListRequest) WithTimeRange(from, to time.Time) *ListRequest {
	r.CreatedFrom = &from
	r.CreatedTo = &to
	return r
}

// WithCreatedFrom 设置创建时间起始
func (r *ListRequest) WithCreatedFrom(from time.Time) *ListRequest {
	r.CreatedFrom = &from
	return r
}

// WithCreatedTo 设置创建时间结束
func (r *ListRequest) WithCreatedTo(to time.Time) *ListRequest {
	r.CreatedTo = &to
	return r
}

// WithPagination 设置分页
func (r *ListRequest) WithPagination(page, pageSize int) *ListRequest {
	r.Page = page
	r.PageSize = pageSize
	return r
}

// ListResponse 任务列表响应
type ListResponse struct {
	*sdk.ListTasksResponse
	Tasks []*Task `json:"tasks"`
}

// NewListResponseFromSDK 从 SDK 响应创建列表响应
func NewListResponseFromSDK(sdkResp *sdk.ListTasksResponse) *ListResponse {
	resp := &ListResponse{
		ListTasksResponse: sdkResp,
		Tasks:             make([]*Task, len(sdkResp.Tasks)),
	}

	// 转换任务列表
	for i, sdkTask := range sdkResp.Tasks {
		resp.Tasks[i] = NewTaskFromSDK(&sdkTask)
	}

	return resp
}

// StatsResponse 任务统计响应
type StatsResponse struct {
	*sdk.TaskStatsResponse
}

// NewStatsResponseFromSDK 从 SDK 响应创建统计响应
func NewStatsResponseFromSDK(sdkResp *sdk.TaskStatsResponse) *StatsResponse {
	return &StatsResponse{
		TaskStatsResponse: sdkResp,
	}
}

// BatchCreateRequest 批量创建任务请求
type BatchCreateRequest struct {
	*sdk.BatchCreateTasksRequest
	Requests []*CreateRequest `json:"tasks"`
}

// NewBatchCreateRequest 创建批量创建请求
func NewBatchCreateRequest() *BatchCreateRequest {
	return &BatchCreateRequest{
		BatchCreateTasksRequest: &sdk.BatchCreateTasksRequest{},
		Requests:                make([]*CreateRequest, 0),
	}
}

// AddTask 添加任务到批量请求
func (r *BatchCreateRequest) AddTask(req *CreateRequest) *BatchCreateRequest {
	r.Requests = append(r.Requests, req)
	return r
}

// AddTasks 添加多个任务到批量请求
func (r *BatchCreateRequest) AddTasks(reqs ...*CreateRequest) *BatchCreateRequest {
	r.Requests = append(r.Requests, reqs...)
	return r
}

// ToSDK 转换为 SDK 批量创建请求
func (r *BatchCreateRequest) ToSDK() *sdk.BatchCreateTasksRequest {
	sdkTasks := make([]sdk.CreateTaskRequest, len(r.Requests))
	for i, req := range r.Requests {
		sdkTasks[i] = *req.CreateTaskRequest
	}
	return &sdk.BatchCreateTasksRequest{
		Tasks: sdkTasks,
	}
}

// BatchCreateResponse 批量创建任务响应
type BatchCreateResponse struct {
	*sdk.BatchCreateTasksResponse
	Succeeded []*Task            `json:"succeeded"`
	Failed    []*BatchTaskError  `json:"failed"`
}

// BatchTaskError 批量操作中的任务错误
type BatchTaskError struct {
	*sdk.BatchTaskError
}

// NewBatchCreateResponseFromSDK 从 SDK 响应创建批量创建响应
func NewBatchCreateResponseFromSDK(sdkResp *sdk.BatchCreateTasksResponse) *BatchCreateResponse {
	resp := &BatchCreateResponse{
		BatchCreateTasksResponse: sdkResp,
		Succeeded:                make([]*Task, len(sdkResp.Succeeded)),
		Failed:                   make([]*BatchTaskError, len(sdkResp.Failed)),
	}

	// 转换成功的任务
	for i, sdkTask := range sdkResp.Succeeded {
		resp.Succeeded[i] = NewTaskFromSDK(&sdkTask)
	}

	// 转换失败的任务
	for i, sdkError := range sdkResp.Failed {
		resp.Failed[i] = &BatchTaskError{
			BatchTaskError: &sdkError,
		}
	}

	return resp
}

// TaskBuilder 任务构建器，用于链式构建任务
type TaskBuilder struct {
	request *CreateRequest
}

// NewTaskBuilder 创建任务构建器
func NewTaskBuilder(businessUniqueID, callbackURL string) *TaskBuilder {
	return &TaskBuilder{
		request: NewCreateRequest(businessUniqueID, callbackURL),
	}
}

// Method 设置回调方法
func (b *TaskBuilder) Method(method string) *TaskBuilder {
	b.request.WithMethod(method)
	return b
}

// Headers 设置回调头
func (b *TaskBuilder) Headers(headers map[string]string) *TaskBuilder {
	b.request.WithHeaders(headers)
	return b
}

// Body 设置回调体
func (b *TaskBuilder) Body(body string) *TaskBuilder {
	b.request.WithBody(body)
	return b
}

// Retry 设置重试配置
func (b *TaskBuilder) Retry(maxRetries int, intervals ...int) *TaskBuilder {
	b.request.WithRetry(maxRetries, intervals...)
	return b
}

// Priority 设置优先级
func (b *TaskBuilder) Priority(priority TaskPriority) *TaskBuilder {
	b.request.WithPriority(priority)
	return b
}

// Tags 设置标签
func (b *TaskBuilder) Tags(tags ...string) *TaskBuilder {
	b.request.WithTags(tags...)
	return b
}

// Timeout 设置超时时间（秒）
func (b *TaskBuilder) Timeout(timeout int) *TaskBuilder {
	b.request.WithTimeout(timeout)
	return b
}

// ScheduledAt 设置计划执行时间
func (b *TaskBuilder) ScheduledAt(t time.Time) *TaskBuilder {
	b.request.WithScheduledAt(t)
	return b
}

// ScheduledAfter 设置在指定时间后执行
func (b *TaskBuilder) ScheduledAfter(duration time.Duration) *TaskBuilder {
	t := time.Now().Add(duration)
	b.request.WithScheduledAt(t)
	return b
}

// Metadata 设置元数据
func (b *TaskBuilder) Metadata(metadata map[string]interface{}) *TaskBuilder {
	b.request.WithMetadata(metadata)
	return b
}

// MetadataValue 设置单个元数据值
func (b *TaskBuilder) MetadataValue(key string, value interface{}) *TaskBuilder {
	if b.request.Metadata == nil {
		b.request.Metadata = make(map[string]interface{})
	}
	b.request.Metadata[key] = value
	return b
}

// Build 构建创建请求
func (b *TaskBuilder) Build() *CreateRequest {
	return b.request
}

// FilterBuilder 任务过滤构建器
type FilterBuilder struct {
	request *ListRequest
}

// NewFilterBuilder 创建过滤构建器
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		request: NewListRequest(),
	}
}

// Status 过滤任务状态
func (f *FilterBuilder) Status(status ...TaskStatus) *FilterBuilder {
	f.request.WithStatus(status...)
	return f
}

// Tags 过滤标签
func (f *FilterBuilder) Tags(tags ...string) *FilterBuilder {
	f.request.WithTags(tags...)
	return f
}

// Priority 过滤优先级
func (f *FilterBuilder) Priority(priority TaskPriority) *FilterBuilder {
	f.request.WithPriority(priority)
	return f
}

// TimeRange 设置时间范围
func (f *FilterBuilder) TimeRange(from, to time.Time) *FilterBuilder {
	f.request.WithTimeRange(from, to)
	return f
}

// CreatedToday 过滤今天创建的任务
func (f *FilterBuilder) CreatedToday() *FilterBuilder {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	f.request.WithTimeRange(today, tomorrow)
	return f
}

// CreatedLastWeek 过滤上周创建的任务
func (f *FilterBuilder) CreatedLastWeek() *FilterBuilder {
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	f.request.WithCreatedFrom(weekAgo)
	return f
}

// Pagination 设置分页
func (f *FilterBuilder) Pagination(page, pageSize int) *FilterBuilder {
	f.request.WithPagination(page, pageSize)
	return f
}

// Build 构建列表请求
func (f *FilterBuilder) Build() *ListRequest {
	return f.request
}

// ToJSON 将对象转换为JSON字符串
func (t *Task) ToJSON() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从JSON字符串创建任务
func FromJSON(jsonStr string) (*Task, error) {
	var sdkTask sdk.Task
	if err := json.Unmarshal([]byte(jsonStr), &sdkTask); err != nil {
		return nil, err
	}
	return NewTaskFromSDK(&sdkTask), nil
}

// IsCompleted 检查任务是否已完成
func (t *Task) IsCompleted() bool {
	return t.Status == StatusSucceeded || t.Status == StatusFailed || t.Status == StatusCancelled || t.Status == StatusExpired
}

// IsRunning 检查任务是否正在运行
func (t *Task) IsRunning() bool {
	return t.Status == StatusRunning
}

// IsPending 检查任务是否待执行
func (t *Task) IsPending() bool {
	return t.Status == StatusPending
}

// IsSucceeded 检查任务是否成功
func (t *Task) IsSucceeded() bool {
	return t.Status == StatusSucceeded
}

// IsFailed 检查任务是否失败
func (t *Task) IsFailed() bool {
	return t.Status == StatusFailed
}

// IsCancelled 检查任务是否被取消
func (t *Task) IsCancelled() bool {
	return t.Status == StatusCancelled
}

// IsExpired 检查任务是否过期
func (t *Task) IsExpired() bool {
	return t.Status == StatusExpired
}

// GetDuration 获取任务执行时长
func (t *Task) GetDuration() time.Duration {
	if t.ExecutedAt == nil {
		return 0
	}

	endTime := time.Now()
	if t.CompletedAt != nil {
		endTime = *t.CompletedAt
	}

	return endTime.Sub(*t.ExecutedAt)
}

// GetWaitTime 获取任务等待时长
func (t *Task) GetWaitTime() time.Duration {
	if t.ExecutedAt == nil {
		return time.Since(t.CreatedAt)
	}
	return t.ExecutedAt.Sub(t.CreatedAt)
}

// HasRetry 检查任务是否已重试
func (t *Task) HasRetry() bool {
	return t.CurrentRetry > 0
}

// CanRetry 检查任务是否可以重试
func (t *Task) CanRetry() bool {
	return t.CurrentRetry < t.MaxRetries && (t.Status == StatusFailed)
}