package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"task-center/sdk/async"
	"task-center/sdk/batch"
	"task-center/sdk/builder"
	"task-center/sdk/task"
)

// TaskCenterSDK 高级SDK封装，提供便捷接口
type TaskCenterSDK struct {
	client       *task.Client
	asyncClient  *async.AsyncClient
	batchClient  *batch.BatchClient
	taskBuilder  *builder.TaskBuilder
	queryBuilder *builder.QueryBuilder
	quickBuilder *builder.QuickBuilder
	quickQuery   *builder.QuickQuery
}

// NewTaskCenterSDK 创建高级SDK实例
func NewTaskCenterSDK(config *Config) (*TaskCenterSDK, error) {
	// 创建基础客户端
	sdkClient, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}

	taskClient := task.NewClient(sdkClient)

	// 创建各个模块的客户端
	asyncClient := async.NewAsyncClient(taskClient, nil)
	batchClient := batch.NewBatchClient(taskClient, nil)

	return &TaskCenterSDK{
		client:       taskClient,
		asyncClient:  asyncClient,
		batchClient:  batchClient,
		taskBuilder:  builder.NewTaskBuilder(taskClient),
		queryBuilder: builder.NewQueryBuilder(taskClient),
		quickBuilder: builder.NewQuickBuilder(taskClient),
		quickQuery:   builder.NewQuickQuery(taskClient),
	}, nil
}

// Close 关闭SDK
func (sdk *TaskCenterSDK) Close() error {
	if sdk.asyncClient != nil {
		sdk.asyncClient.Stop()
	}
	return sdk.client.Close()
}

// ========== 便捷任务创建方法 ==========

// CreateSimpleTask 创建简单任务
func (sdk *TaskCenterSDK) CreateSimpleTask(ctx context.Context, name, taskType string, payload interface{}) (*task.Task, error) {
	return sdk.taskBuilder.
		Reset().
		WithContext(ctx).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		Build()
}

// CreateDelayedTask 创建延迟任务
func (sdk *TaskCenterSDK) CreateDelayedTask(ctx context.Context, name, taskType string, payload interface{}, delay time.Duration) (*task.Task, error) {
	return sdk.taskBuilder.
		Reset().
		WithContext(ctx).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithDelay(delay).
		Build()
}

// CreateScheduledTask 创建定时任务
func (sdk *TaskCenterSDK) CreateScheduledTask(ctx context.Context, name, taskType string, payload interface{}, scheduledAt time.Time) (*task.Task, error) {
	return sdk.taskBuilder.
		Reset().
		WithContext(ctx).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithScheduledTime(scheduledAt).
		Build()
}

// CreateHighPriorityTask 创建高优先级任务
func (sdk *TaskCenterSDK) CreateHighPriorityTask(ctx context.Context, name, taskType string, payload interface{}) (*task.Task, error) {
	return sdk.taskBuilder.
		Reset().
		WithContext(ctx).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithHighPriority().
		Build()
}

// CreateTaskWithCallback 创建带回调的任务
func (sdk *TaskCenterSDK) CreateTaskWithCallback(ctx context.Context, name, taskType string, payload interface{}, callbackURL string) (*task.Task, error) {
	return sdk.taskBuilder.
		Reset().
		WithContext(ctx).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithCallback(callbackURL).
		Build()
}

// CreateTaskWithRetry 创建带重试策略的任务
func (sdk *TaskCenterSDK) CreateTaskWithRetry(ctx context.Context, name, taskType string, payload interface{}, maxRetries int, retryInterval time.Duration) (*task.Task, error) {
	return sdk.taskBuilder.
		Reset().
		WithContext(ctx).
		WithName(name).
		WithType(taskType).
		WithPayload(payload).
		WithRetryPolicy(maxRetries, retryInterval).
		Build()
}

// ========== 便捷查询方法 ==========

// GetPendingTasks 获取待执行任务
func (sdk *TaskCenterSDK) GetPendingTasks(ctx context.Context, limit int) ([]*task.Task, error) {
	result, err := sdk.queryBuilder.
		Reset().
		WithContext(ctx).
		WithPendingStatus().
		WithLimit(limit).
		Execute()
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// GetFailedTasks 获取失败任务
func (sdk *TaskCenterSDK) GetFailedTasks(ctx context.Context, limit int) ([]*task.Task, error) {
	result, err := sdk.queryBuilder.
		Reset().
		WithContext(ctx).
		WithFailedStatus().
		WithLimit(limit).
		Execute()
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// GetTasksByTag 根据标签获取任务
func (sdk *TaskCenterSDK) GetTasksByTag(ctx context.Context, tag string, limit int) ([]*task.Task, error) {
	result, err := sdk.queryBuilder.
		Reset().
		WithContext(ctx).
		WithTag(tag).
		WithLimit(limit).
		Execute()
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// GetRecentTasks 获取最近创建的任务
func (sdk *TaskCenterSDK) GetRecentTasks(ctx context.Context, duration time.Duration, limit int) ([]*task.Task, error) {
	since := time.Now().Add(-duration)
	result, err := sdk.queryBuilder.
		Reset().
		WithContext(ctx).
		WithCreatedAfter(since).
		WithLimit(limit).
		Execute()
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// SearchTasks 搜索任务
func (sdk *TaskCenterSDK) SearchTasks(ctx context.Context, query string, limit int) ([]*task.Task, error) {
	listReq := task.NewListRequest().WithPagination(1, limit)
	result, err := sdk.client.SearchTasks(ctx, query, listReq)
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// ========== 异步操作方法 ==========

// CreateTaskAsync 异步创建任务
func (sdk *TaskCenterSDK) CreateTaskAsync(name, taskType string, payload interface{}, callback func(*async.TaskResult)) (string, error) {
	request := task.NewCreateRequest().
		WithName(name).
		WithType(taskType).
		WithPayload(payload)

	if !sdk.asyncClient.IsStarted() {
		sdk.asyncClient.Start()
	}

	return sdk.asyncClient.CreateTaskAsync(request, callback)
}

// CreateTaskFuture 创建任务并返回Future
func (sdk *TaskCenterSDK) CreateTaskFuture(name, taskType string, payload interface{}) *async.Future {
	request := task.NewCreateRequest().
		WithName(name).
		WithType(taskType).
		WithPayload(payload)

	if !sdk.asyncClient.IsStarted() {
		sdk.asyncClient.Start()
	}

	return sdk.asyncClient.CreateTaskFuture(request)
}

// ========== 批量操作方法 ==========

// CreateTasksBatch 批量创建任务
func (sdk *TaskCenterSDK) CreateTasksBatch(ctx context.Context, requests []*task.CreateRequest) (*batch.BatchCreateResult, error) {
	return sdk.batchClient.CreateTasks(ctx, requests)
}

// DeleteTasksBatch 批量删除任务
func (sdk *TaskCenterSDK) DeleteTasksBatch(ctx context.Context, taskIDs []int64) (*batch.BatchDeleteResult, error) {
	return sdk.batchClient.DeleteTasks(ctx, taskIDs)
}

// CancelTasksBatch 批量取消任务
func (sdk *TaskCenterSDK) CancelTasksBatch(ctx context.Context, taskIDs []int64) (*batch.BatchUpdateResult, error) {
	return sdk.batchClient.CancelTasks(ctx, taskIDs)
}

// RetryTasksBatch 批量重试任务
func (sdk *TaskCenterSDK) RetryTasksBatch(ctx context.Context, taskIDs []int64) (*batch.BatchUpdateResult, error) {
	return sdk.batchClient.RetryTasks(ctx, taskIDs)
}

// ========== 工具函数 ==========

// TaskExists 检查任务是否存在
func (sdk *TaskCenterSDK) TaskExists(ctx context.Context, taskID int64) (bool, error) {
	return sdk.client.CheckTaskExists(ctx, taskID)
}

// WaitForTask 等待任务完成
func (sdk *TaskCenterSDK) WaitForTask(ctx context.Context, taskID int64, checkInterval time.Duration) (*task.Task, error) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			taskObj, err := sdk.client.GetTask(ctx, taskID)
			if err != nil {
				return nil, err
			}

			// 检查任务是否完成
			if IsTaskCompleted(taskObj.Status) {
				return taskObj, nil
			}
		}
	}
}

// WaitForTasks 等待多个任务完成
func (sdk *TaskCenterSDK) WaitForTasks(ctx context.Context, taskIDs []int64, checkInterval time.Duration) ([]*task.Task, error) {
	results := make([]*task.Task, len(taskIDs))
	completed := make([]bool, len(taskIDs))
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			allCompleted := true

			for i, taskID := range taskIDs {
				if completed[i] {
					continue
				}

				taskObj, err := sdk.client.GetTask(ctx, taskID)
				if err != nil {
					return nil, err
				}

				if IsTaskCompleted(taskObj.Status) {
					results[i] = taskObj
					completed[i] = true
				} else {
					allCompleted = false
				}
			}

			if allCompleted {
				return results, nil
			}
		}
	}
}

// ========== 数据转换和验证工具 ==========

// IsTaskCompleted 检查任务是否已完成
func IsTaskCompleted(status task.TaskStatus) bool {
	return status == task.StatusSucceeded ||
		status == task.StatusFailed ||
		status == task.StatusCancelled ||
		status == task.StatusExpired
}

// IsTaskSuccessful 检查任务是否成功完成
func IsTaskSuccessful(status task.TaskStatus) bool {
	return status == task.StatusSucceeded
}

// IsTaskFailed 检查任务是否失败
func IsTaskFailed(status task.TaskStatus) bool {
	return status == task.StatusFailed
}

// IsTaskActive 检查任务是否处于活跃状态
func IsTaskActive(status task.TaskStatus) bool {
	return status == task.StatusPending || status == task.StatusRunning
}

// PayloadToStruct 将任务负载转换为结构体
func PayloadToStruct(payload []byte, target interface{}) error {
	if payload == nil {
		return nil
	}
	return json.Unmarshal(payload, target)
}

// StructToPayload 将结构体转换为任务负载
func StructToPayload(source interface{}) ([]byte, error) {
	if source == nil {
		return nil, nil
	}
	return json.Marshal(source)
}

// ValidateTaskRequest 验证任务请求
func ValidateTaskRequest(req *task.CreateRequest) error {
	if req == nil {
		return NewValidationError("request cannot be nil")
	}
	if req.Name == "" {
		return NewValidationError("task name cannot be empty")
	}
	if req.Type == "" {
		return NewValidationError("task type cannot be empty")
	}
	return nil
}

// ========== 统计和监控工具 ==========

// TaskStats 任务统计信息
type TaskStats struct {
	Total     int                       `json:"total"`
	ByStatus  map[task.TaskStatus]int   `json:"by_status"`
	ByPriority map[task.TaskPriority]int `json:"by_priority"`
	ByType    map[string]int            `json:"by_type"`
}

// CalculateTaskStats 计算任务统计信息
func CalculateTaskStats(tasks []*task.Task) *TaskStats {
	stats := &TaskStats{
		Total:      len(tasks),
		ByStatus:   make(map[task.TaskStatus]int),
		ByPriority: make(map[task.TaskPriority]int),
		ByType:     make(map[string]int),
	}

	for _, t := range tasks {
		stats.ByStatus[t.Status]++
		stats.ByPriority[t.Priority]++
		stats.ByType[t.Type]++
	}

	return stats
}

// GetTaskStatistics 获取任务统计信息
func (sdk *TaskCenterSDK) GetTaskStatistics(ctx context.Context) (*task.StatsResponse, error) {
	return sdk.client.GetTaskStats(ctx)
}

// ========== 模板和预定义工具 ==========

// EmailTaskTemplate 邮件任务模板
type EmailTaskTemplate struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	CC      string `json:"cc,omitempty"`
	BCC     string `json:"bcc,omitempty"`
}

// CreateEmailTask 创建邮件任务
func (sdk *TaskCenterSDK) CreateEmailTask(ctx context.Context, to, subject, body string) (*task.Task, error) {
	return sdk.quickBuilder.Email(to, subject, body).
		WithContext(ctx).
		Build()
}

// ReportTaskTemplate 报表任务模板
type ReportTaskTemplate struct {
	Type     string                 `json:"type"`
	Format   string                 `json:"format"`
	Params   map[string]interface{} `json:"params"`
	OutputTo string                 `json:"output_to,omitempty"`
}

// CreateReportTask 创建报表任务
func (sdk *TaskCenterSDK) CreateReportTask(ctx context.Context, reportType string, params map[string]interface{}) (*task.Task, error) {
	return sdk.quickBuilder.Report(reportType, params).
		WithContext(ctx).
		Build()
}

// PaymentTaskTemplate 支付任务模板
type PaymentTaskTemplate struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Method   string  `json:"method,omitempty"`
	Gateway  string  `json:"gateway,omitempty"`
}

// CreatePaymentTask 创建支付任务
func (sdk *TaskCenterSDK) CreatePaymentTask(ctx context.Context, orderID string, amount float64, currency string) (*task.Task, error) {
	return sdk.quickBuilder.Payment(orderID, amount, currency).
		WithContext(ctx).
		Build()
}

// ========== 错误处理工具 ==========

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int
	Interval    time.Duration
	Backoff     float64
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		Interval:    1 * time.Second,
		Backoff:     2.0,
	}
}

// RetryWithBackoff 带退避的重试
func RetryWithBackoff(ctx context.Context, config *RetryConfig, operation func() error) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	interval := config.Interval

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
				interval = time.Duration(float64(interval) * config.Backoff)
			}
		}

		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// ========== 调试和日志工具 ==========

// TaskInfo 任务信息摘要
type TaskInfo struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      task.TaskStatus   `json:"status"`
	Priority    task.TaskPriority `json:"priority"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// GetTaskInfo 获取任务信息摘要
func GetTaskInfo(t *task.Task) *TaskInfo {
	if t == nil {
		return nil
	}

	return &TaskInfo{
		ID:          t.ID,
		Name:        t.Name,
		Type:        t.Type,
		Status:      t.Status,
		Priority:    t.Priority,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		ScheduledAt: t.ScheduledAt,
		Tags:        t.Tags,
	}
}

// FormatTaskSummary 格式化任务摘要
func FormatTaskSummary(t *task.Task) string {
	if t == nil {
		return "Task: <nil>"
	}

	return fmt.Sprintf("Task{ID: %d, Name: %s, Type: %s, Status: %s, Priority: %s, Created: %s}",
		t.ID, t.Name, t.Type, t.Status, t.Priority, t.CreatedAt.Format("2006-01-02 15:04:05"))
}

// PrintTasksTable 打印任务表格（用于调试）
func PrintTasksTable(tasks []*task.Task) {
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	// 表头
	fmt.Printf("%-8s %-20s %-15s %-12s %-10s %-20s\n",
		"ID", "Name", "Type", "Status", "Priority", "Created")
	fmt.Println(strings.Repeat("-", 95))

	// 任务行
	for _, t := range tasks {
		name := t.Name
		if len(name) > 18 {
			name = name[:15] + "..."
		}

		taskType := t.Type
		if len(taskType) > 13 {
			taskType = taskType[:10] + "..."
		}

		fmt.Printf("%-8d %-20s %-15s %-12s %-10s %-20s\n",
			t.ID, name, taskType, t.Status, t.Priority,
			t.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

// ========== 反射和类型工具 ==========

// GetStructFields 获取结构体字段信息
func GetStructFields(v interface{}) map[string]string {
	fields := make(map[string]string)
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		rt = rt.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return fields
	}

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.IsExported() {
			fields[field.Name] = field.Type.String()
		}
	}

	return fields
}

// ConvertToMap 将结构体转换为map
func ConvertToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

// ========== 获取内部客户端的方法 ==========

// GetTaskClient 获取任务客户端
func (sdk *TaskCenterSDK) GetTaskClient() *task.Client {
	return sdk.client
}

// GetAsyncClient 获取异步客户端
func (sdk *TaskCenterSDK) GetAsyncClient() *async.AsyncClient {
	return sdk.asyncClient
}

// GetBatchClient 获取批量客户端
func (sdk *TaskCenterSDK) GetBatchClient() *batch.BatchClient {
	return sdk.batchClient
}

// GetTaskBuilder 获取任务构建器
func (sdk *TaskCenterSDK) GetTaskBuilder() *builder.TaskBuilder {
	return sdk.taskBuilder
}

// GetQueryBuilder 获取查询构建器
func (sdk *TaskCenterSDK) GetQueryBuilder() *builder.QueryBuilder {
	return sdk.queryBuilder
}