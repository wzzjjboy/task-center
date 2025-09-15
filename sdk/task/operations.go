package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"task-center/sdk"
)

// BatchOperations 批量操作接口
type BatchOperations interface {
	// BatchCreate 批量创建任务
	BatchCreate(ctx context.Context, req *BatchCreateRequest) (*BatchCreateResponse, error)
	// BatchUpdate 批量更新任务
	BatchUpdate(ctx context.Context, updates []BatchUpdateItem) (*BatchUpdateResponse, error)
	// BatchCancel 批量取消任务
	BatchCancel(ctx context.Context, taskIDs []int64) (*BatchCancelResponse, error)
	// BatchRetry 批量重试任务
	BatchRetry(ctx context.Context, taskIDs []int64) (*BatchRetryResponse, error)
	// BatchDelete 批量删除任务
	BatchDelete(ctx context.Context, taskIDs []int64) (*BatchDeleteResponse, error)
}

// BatchUpdateItem 批量更新项
type BatchUpdateItem struct {
	TaskID  int64          `json:"task_id"`
	Request *UpdateRequest `json:"request"`
}

// BatchUpdateResponse 批量更新响应
type BatchUpdateResponse struct {
	Succeeded []*Task            `json:"succeeded"`
	Failed    []*BatchTaskError  `json:"failed"`
}

// BatchCancelResponse 批量取消响应
type BatchCancelResponse struct {
	Succeeded []*Task           `json:"succeeded"`
	Failed    []*BatchTaskError `json:"failed"`
}

// BatchRetryResponse 批量重试响应
type BatchRetryResponse struct {
	Succeeded []*Task           `json:"succeeded"`
	Failed    []*BatchTaskError `json:"failed"`
}

// BatchDeleteResponse 批量删除响应
type BatchDeleteResponse struct {
	Succeeded []int64           `json:"succeeded"`
	Failed    []*BatchTaskError `json:"failed"`
}

// Operations 扩展操作集合
type Operations struct {
	client *Client
}

// NewOperations 创建操作集合
func NewOperations(client *Client) *Operations {
	return &Operations{client: client}
}

// BatchCreate 批量创建任务
func (ops *Operations) BatchCreate(ctx context.Context, req *BatchCreateRequest) (*BatchCreateResponse, error) {
	if req == nil {
		return nil, sdk.NewValidationError("batch create request cannot be nil")
	}

	if len(req.Requests) == 0 {
		return nil, sdk.NewValidationError("batch create request must contain at least one task")
	}

	// 验证所有请求
	for i, taskReq := range req.Requests {
		if err := taskReq.Validate(); err != nil {
			return nil, fmt.Errorf("task %d validation failed: %w", i, err)
		}
	}

	// 发送批量创建请求
	resp, err := ops.client.sdkClient.DoRequest(ctx, "POST", "/api/v1/tasks/batch", req.ToSDK())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ops.parseBatchCreateResponse(resp)
}

// BatchUpdate 批量更新任务
func (ops *Operations) BatchUpdate(ctx context.Context, updates []BatchUpdateItem) (*BatchUpdateResponse, error) {
	if len(updates) == 0 {
		return nil, sdk.NewValidationError("batch update must contain at least one item")
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"updates": updates,
	}

	resp, err := ops.client.sdkClient.DoRequest(ctx, "PUT", "/api/v1/tasks/batch", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ops.parseBatchUpdateResponse(resp)
}

// BatchCancel 批量取消任务
func (ops *Operations) BatchCancel(ctx context.Context, taskIDs []int64) (*BatchCancelResponse, error) {
	if len(taskIDs) == 0 {
		return nil, sdk.NewValidationError("task IDs cannot be empty")
	}

	requestBody := map[string]interface{}{
		"task_ids": taskIDs,
	}

	resp, err := ops.client.sdkClient.DoRequest(ctx, "POST", "/api/v1/tasks/batch/cancel", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ops.parseBatchCancelResponse(resp)
}

// BatchRetry 批量重试任务
func (ops *Operations) BatchRetry(ctx context.Context, taskIDs []int64) (*BatchRetryResponse, error) {
	if len(taskIDs) == 0 {
		return nil, sdk.NewValidationError("task IDs cannot be empty")
	}

	requestBody := map[string]interface{}{
		"task_ids": taskIDs,
	}

	resp, err := ops.client.sdkClient.DoRequest(ctx, "POST", "/api/v1/tasks/batch/retry", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ops.parseBatchRetryResponse(resp)
}

// BatchDelete 批量删除任务
func (ops *Operations) BatchDelete(ctx context.Context, taskIDs []int64) (*BatchDeleteResponse, error) {
	if len(taskIDs) == 0 {
		return nil, sdk.NewValidationError("task IDs cannot be empty")
	}

	requestBody := map[string]interface{}{
		"task_ids": taskIDs,
	}

	resp, err := ops.client.sdkClient.DoRequest(ctx, "DELETE", "/api/v1/tasks/batch", requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ops.parseBatchDeleteResponse(resp)
}

// ConcurrentOperations 并发操作
type ConcurrentOperations struct {
	client     *Client
	maxWorkers int
	timeout    time.Duration
}

// NewConcurrentOperations 创建并发操作
func NewConcurrentOperations(client *Client, maxWorkers int, timeout time.Duration) *ConcurrentOperations {
	if maxWorkers <= 0 {
		maxWorkers = 10 // 默认10个并发
	}
	if timeout <= 0 {
		timeout = 30 * time.Second // 默认30秒超时
	}

	return &ConcurrentOperations{
		client:     client,
		maxWorkers: maxWorkers,
		timeout:    timeout,
	}
}

// ConcurrentCreateResult 并发创建结果
type ConcurrentCreateResult struct {
	Index int
	Task  *Task
	Error error
}

// ConcurrentCreate 并发创建任务
func (cop *ConcurrentOperations) ConcurrentCreate(ctx context.Context, requests []*CreateRequest) []ConcurrentCreateResult {
	if len(requests) == 0 {
		return nil
	}

	results := make([]ConcurrentCreateResult, len(requests))
	jobs := make(chan int, len(requests))
	var wg sync.WaitGroup

	// 创建工作池
	for w := 0; w < cop.maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				taskCtx, cancel := context.WithTimeout(ctx, cop.timeout)
				task, err := cop.client.CreateTask(taskCtx, requests[i])
				cancel()

				results[i] = ConcurrentCreateResult{
					Index: i,
					Task:  task,
					Error: err,
				}
			}
		}()
	}

	// 提交任务
	for i := range requests {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results
}

// ConcurrentUpdateResult 并发更新结果
type ConcurrentUpdateResult struct {
	TaskID int64
	Task   *Task
	Error  error
}

// ConcurrentUpdate 并发更新任务
func (cop *ConcurrentOperations) ConcurrentUpdate(ctx context.Context, updates map[int64]*UpdateRequest) []ConcurrentUpdateResult {
	if len(updates) == 0 {
		return nil
	}

	results := make([]ConcurrentUpdateResult, 0, len(updates))
	jobs := make(chan map[int64]*UpdateRequest, len(updates))
	resultsChan := make(chan ConcurrentUpdateResult, len(updates))
	var wg sync.WaitGroup

	// 创建工作池
	for w := 0; w < cop.maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for update := range jobs {
				for taskID, req := range update {
					taskCtx, cancel := context.WithTimeout(ctx, cop.timeout)
					task, err := cop.client.UpdateTask(taskCtx, taskID, req)
					cancel()

					resultsChan <- ConcurrentUpdateResult{
						TaskID: taskID,
						Task:   task,
						Error:  err,
					}
				}
			}
		}()
	}

	// 提交任务
	for taskID, req := range updates {
		jobs <- map[int64]*UpdateRequest{taskID: req}
	}
	close(jobs)

	// 收集结果
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// TaskWatcher 任务监控器
type TaskWatcher struct {
	client   *Client
	interval time.Duration
	stopChan chan struct{}
	mu       sync.RWMutex
	watchers map[int64]chan *Task
}

// NewTaskWatcher 创建任务监控器
func NewTaskWatcher(client *Client, interval time.Duration) *TaskWatcher {
	if interval <= 0 {
		interval = 5 * time.Second // 默认5秒检查一次
	}

	return &TaskWatcher{
		client:   client,
		interval: interval,
		stopChan: make(chan struct{}),
		watchers: make(map[int64]chan *Task),
	}
}

// WatchTask 监控任务状态变化
func (tw *TaskWatcher) WatchTask(ctx context.Context, taskID int64) <-chan *Task {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	taskChan := make(chan *Task, 10)
	tw.watchers[taskID] = taskChan

	return taskChan
}

// StopWatching 停止监控任务
func (tw *TaskWatcher) StopWatching(taskID int64) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if taskChan, exists := tw.watchers[taskID]; exists {
		close(taskChan)
		delete(tw.watchers, taskID)
	}
}

// Start 启动监控
func (tw *TaskWatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(tw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tw.stopChan:
			return
		case <-ticker.C:
			tw.checkTasks(ctx)
		}
	}
}

// Stop 停止监控
func (tw *TaskWatcher) Stop() {
	close(tw.stopChan)

	tw.mu.Lock()
	defer tw.mu.Unlock()

	for _, taskChan := range tw.watchers {
		close(taskChan)
	}
	tw.watchers = make(map[int64]chan *Task)
}

// checkTasks 检查任务状态
func (tw *TaskWatcher) checkTasks(ctx context.Context) {
	tw.mu.RLock()
	taskIDs := make([]int64, 0, len(tw.watchers))
	for taskID := range tw.watchers {
		taskIDs = append(taskIDs, taskID)
	}
	tw.mu.RUnlock()

	for _, taskID := range taskIDs {
		task, err := tw.client.GetTask(ctx, taskID)
		if err != nil {
			continue
		}

		tw.mu.RLock()
		taskChan, exists := tw.watchers[taskID]
		tw.mu.RUnlock()

		if exists {
			select {
			case taskChan <- task:
			default:
				// 如果通道已满，跳过这次更新
			}
		}
	}
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	client *Client
}

// NewTaskScheduler 创建任务调度器
func NewTaskScheduler(client *Client) *TaskScheduler {
	return &TaskScheduler{client: client}
}

// ScheduleTask 调度任务在指定时间执行
func (ts *TaskScheduler) ScheduleTask(ctx context.Context, req *CreateRequest, scheduledAt time.Time) (*Task, error) {
	req.WithScheduledAt(scheduledAt)
	return ts.client.CreateTask(ctx, req)
}

// ScheduleTaskAfter 调度任务在指定时间后执行
func (ts *TaskScheduler) ScheduleTaskAfter(ctx context.Context, req *CreateRequest, duration time.Duration) (*Task, error) {
	scheduledAt := time.Now().Add(duration)
	return ts.ScheduleTask(ctx, req, scheduledAt)
}

// ScheduleCronTask 创建定时任务（需要服务端支持）
func (ts *TaskScheduler) ScheduleCronTask(ctx context.Context, req *CreateRequest, cronExpr string) (*Task, error) {
	// 在元数据中添加 cron 表达式
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["cron_expression"] = cronExpr
	req.Metadata["task_type"] = "cron"

	return ts.client.CreateTask(ctx, req)
}

// 内部方法：解析批量创建响应
func (ops *Operations) parseBatchCreateResponse(resp *http.Response) (*BatchCreateResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ops.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp sdk.ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.Success {
		return nil, sdk.NewServerError(apiResp.Message)
	}

	// 解析批量创建数据
	batchData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch data: %w", err)
	}

	var sdkResp sdk.BatchCreateTasksResponse
	if err := json.Unmarshal(batchData, &sdkResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch create response: %w", err)
	}

	return NewBatchCreateResponseFromSDK(&sdkResp), nil
}

// 内部方法：解析批量更新响应
func (ops *Operations) parseBatchUpdateResponse(resp *http.Response) (*BatchUpdateResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ops.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result BatchUpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch update response: %w", err)
	}

	return &result, nil
}

// 内部方法：解析批量取消响应
func (ops *Operations) parseBatchCancelResponse(resp *http.Response) (*BatchCancelResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ops.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result BatchCancelResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch cancel response: %w", err)
	}

	return &result, nil
}

// 内部方法：解析批量重试响应
func (ops *Operations) parseBatchRetryResponse(resp *http.Response) (*BatchRetryResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ops.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result BatchRetryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch retry response: %w", err)
	}

	return &result, nil
}

// 内部方法：解析批量删除响应
func (ops *Operations) parseBatchDeleteResponse(resp *http.Response) (*BatchDeleteResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ops.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result BatchDeleteResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch delete response: %w", err)
	}

	return &result, nil
}

// 内部方法：解析错误响应
func (ops *Operations) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	return sdk.ParseHTTPError(resp.StatusCode, body)
}

// TaskQuery 任务查询构建器
type TaskQuery struct {
	client *Client
	filter *FilterBuilder
}

// NewTaskQuery 创建任务查询构建器
func NewTaskQuery(client *Client) *TaskQuery {
	return &TaskQuery{
		client: client,
		filter: NewFilterBuilder(),
	}
}

// Status 过滤任务状态
func (tq *TaskQuery) Status(status ...TaskStatus) *TaskQuery {
	tq.filter.Status(status...)
	return tq
}

// Tags 过滤标签
func (tq *TaskQuery) Tags(tags ...string) *TaskQuery {
	tq.filter.Tags(tags...)
	return tq
}

// Priority 过滤优先级
func (tq *TaskQuery) Priority(priority TaskPriority) *TaskQuery {
	tq.filter.Priority(priority)
	return tq
}

// TimeRange 设置时间范围
func (tq *TaskQuery) TimeRange(from, to time.Time) *TaskQuery {
	tq.filter.TimeRange(from, to)
	return tq
}

// Pagination 设置分页
func (tq *TaskQuery) Pagination(page, pageSize int) *TaskQuery {
	tq.filter.Pagination(page, pageSize)
	return tq
}

// Execute 执行查询
func (tq *TaskQuery) Execute(ctx context.Context) (*ListResponse, error) {
	return tq.client.ListTasks(ctx, tq.filter.Build())
}

// Count 获取查询结果数量
func (tq *TaskQuery) Count(ctx context.Context) (int, error) {
	// 设置分页为1条记录，只获取总数
	req := tq.filter.Build()
	req.WithPagination(1, 1)

	resp, err := tq.client.ListTasks(ctx, req)
	if err != nil {
		return 0, err
	}

	return resp.Total, nil
}

// First 获取第一个匹配的任务
func (tq *TaskQuery) First(ctx context.Context) (*Task, error) {
	req := tq.filter.Build()
	req.WithPagination(1, 1)

	resp, err := tq.client.ListTasks(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Tasks) == 0 {
		return nil, sdk.NewNotFoundError("task")
	}

	return resp.Tasks[0], nil
}

// All 获取所有匹配的任务
func (tq *TaskQuery) All(ctx context.Context) ([]*Task, error) {
	var allTasks []*Task
	page := 1
	pageSize := 100

	for {
		req := tq.filter.Build()
		req.WithPagination(page, pageSize)

		resp, err := tq.client.ListTasks(ctx, req)
		if err != nil {
			return nil, err
		}

		allTasks = append(allTasks, resp.Tasks...)

		if len(resp.Tasks) < pageSize {
			break
		}

		page++
	}

	return allTasks, nil
}