package batch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"task-center/sdk/task"
)

// BatchClient 批量操作客户端
type BatchClient struct {
	client      *task.Client
	concurrency int
	timeout     time.Duration
}

// BatchConfig 批量操作配置
type BatchConfig struct {
	Concurrency int           // 并发数
	Timeout     time.Duration // 操作超时时间
	BatchSize   int           // 批次大小
}

// DefaultBatchConfig 默认批量操作配置
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		Concurrency: 10,
		Timeout:     5 * time.Minute,
		BatchSize:   100,
	}
}

// NewBatchClient 创建批量操作客户端
func NewBatchClient(client *task.Client, config *BatchConfig) *BatchClient {
	if config == nil {
		config = DefaultBatchConfig()
	}

	return &BatchClient{
		client:      client,
		concurrency: config.Concurrency,
		timeout:     config.Timeout,
	}
}

// BatchCreateRequest 批量创建请求
type BatchCreateRequest struct {
	Tasks []*task.CreateRequest
}

// BatchCreateResult 批量创建结果
type BatchCreateResult struct {
	Success []*task.Task
	Failed  []BatchError
	Total   int
}

// BatchError 批量操作错误
type BatchError struct {
	Index   int
	Request *task.CreateRequest
	Error   error
}

// BatchUpdateRequest 批量更新请求
type BatchUpdateRequest struct {
	Updates []BatchUpdateItem
}

// BatchUpdateItem 批量更新项
type BatchUpdateItem struct {
	TaskID  int64
	Request *task.UpdateRequest
}

// BatchUpdateResult 批量更新结果
type BatchUpdateResult struct {
	Success []*task.Task
	Failed  []BatchError
	Total   int
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	TaskIDs []int64
}

// BatchDeleteResult 批量删除结果
type BatchDeleteResult struct {
	Success []int64
	Failed  []BatchError
	Total   int
}

// BatchQueryRequest 批量查询请求
type BatchQueryRequest struct {
	TaskIDs []int64
}

// BatchQueryResult 批量查询结果
type BatchQueryResult struct {
	Success []*task.Task
	Failed  []BatchError
	Total   int
}

// CreateTasks 批量创建任务
func (bc *BatchClient) CreateTasks(ctx context.Context, requests []*task.CreateRequest) (*BatchCreateResult, error) {
	if len(requests) == 0 {
		return &BatchCreateResult{Total: 0}, nil
	}

	result := &BatchCreateResult{
		Total: len(requests),
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, bc.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 处理每个任务
	for i, req := range requests {
		wg.Add(1)
		go func(index int, request *task.CreateRequest) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建带超时的上下文
			taskCtx, cancel := context.WithTimeout(ctx, bc.timeout)
			defer cancel()

			// 执行任务创建
			createdTask, err := bc.client.CreateTask(taskCtx, request)

			// 保存结果
			mu.Lock()
			if err != nil {
				result.Failed = append(result.Failed, BatchError{
					Index:   index,
					Request: request,
					Error:   err,
				})
			} else {
				result.Success = append(result.Success, createdTask)
			}
			mu.Unlock()
		}(i, req)
	}

	// 等待所有任务完成
	wg.Wait()

	return result, nil
}

// UpdateTasks 批量更新任务
func (bc *BatchClient) UpdateTasks(ctx context.Context, updates []BatchUpdateItem) (*BatchUpdateResult, error) {
	if len(updates) == 0 {
		return &BatchUpdateResult{Total: 0}, nil
	}

	result := &BatchUpdateResult{
		Total: len(updates),
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, bc.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 处理每个更新
	for i, update := range updates {
		wg.Add(1)
		go func(index int, item BatchUpdateItem) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建带超时的上下文
			taskCtx, cancel := context.WithTimeout(ctx, bc.timeout)
			defer cancel()

			// 执行任务更新
			updatedTask, err := bc.client.UpdateTask(taskCtx, item.TaskID, item.Request)

			// 保存结果
			mu.Lock()
			if err != nil {
				result.Failed = append(result.Failed, BatchError{
					Index: index,
					Error: err,
				})
			} else {
				result.Success = append(result.Success, updatedTask)
			}
			mu.Unlock()
		}(i, update)
	}

	// 等待所有任务完成
	wg.Wait()

	return result, nil
}

// DeleteTasks 批量删除任务
func (bc *BatchClient) DeleteTasks(ctx context.Context, taskIDs []int64) (*BatchDeleteResult, error) {
	if len(taskIDs) == 0 {
		return &BatchDeleteResult{Total: 0}, nil
	}

	result := &BatchDeleteResult{
		Total: len(taskIDs),
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, bc.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 处理每个删除
	for i, taskID := range taskIDs {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建带超时的上下文
			taskCtx, cancel := context.WithTimeout(ctx, bc.timeout)
			defer cancel()

			// 执行任务删除
			err := bc.client.DeleteTask(taskCtx, id)

			// 保存结果
			mu.Lock()
			if err != nil {
				result.Failed = append(result.Failed, BatchError{
					Index: index,
					Error: err,
				})
			} else {
				result.Success = append(result.Success, id)
			}
			mu.Unlock()
		}(i, taskID)
	}

	// 等待所有任务完成
	wg.Wait()

	return result, nil
}

// QueryTasks 批量查询任务
func (bc *BatchClient) QueryTasks(ctx context.Context, taskIDs []int64) (*BatchQueryResult, error) {
	if len(taskIDs) == 0 {
		return &BatchQueryResult{Total: 0}, nil
	}

	result := &BatchQueryResult{
		Total: len(taskIDs),
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, bc.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 处理每个查询
	for i, taskID := range taskIDs {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建带超时的上下文
			taskCtx, cancel := context.WithTimeout(ctx, bc.timeout)
			defer cancel()

			// 执行任务查询
			taskObj, err := bc.client.GetTask(taskCtx, id)

			// 保存结果
			mu.Lock()
			if err != nil {
				result.Failed = append(result.Failed, BatchError{
					Index: index,
					Error: err,
				})
			} else {
				result.Success = append(result.Success, taskObj)
			}
			mu.Unlock()
		}(i, taskID)
	}

	// 等待所有任务完成
	wg.Wait()

	return result, nil
}

// CancelTasks 批量取消任务
func (bc *BatchClient) CancelTasks(ctx context.Context, taskIDs []int64) (*BatchUpdateResult, error) {
	if len(taskIDs) == 0 {
		return &BatchUpdateResult{Total: 0}, nil
	}

	result := &BatchUpdateResult{
		Total: len(taskIDs),
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, bc.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 处理每个取消
	for i, taskID := range taskIDs {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建带超时的上下文
			taskCtx, cancel := context.WithTimeout(ctx, bc.timeout)
			defer cancel()

			// 执行任务取消
			cancelledTask, err := bc.client.CancelTask(taskCtx, id)

			// 保存结果
			mu.Lock()
			if err != nil {
				result.Failed = append(result.Failed, BatchError{
					Index: index,
					Error: err,
				})
			} else {
				result.Success = append(result.Success, cancelledTask)
			}
			mu.Unlock()
		}(i, taskID)
	}

	// 等待所有任务完成
	wg.Wait()

	return result, nil
}

// RetryTasks 批量重试任务
func (bc *BatchClient) RetryTasks(ctx context.Context, taskIDs []int64) (*BatchUpdateResult, error) {
	if len(taskIDs) == 0 {
		return &BatchUpdateResult{Total: 0}, nil
	}

	result := &BatchUpdateResult{
		Total: len(taskIDs),
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, bc.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 处理每个重试
	for i, taskID := range taskIDs {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 创建带超时的上下文
			taskCtx, cancel := context.WithTimeout(ctx, bc.timeout)
			defer cancel()

			// 执行任务重试
			retriedTask, err := bc.client.RetryTask(taskCtx, id)

			// 保存结果
			mu.Lock()
			if err != nil {
				result.Failed = append(result.Failed, BatchError{
					Index: index,
					Error: err,
				})
			} else {
				result.Success = append(result.Success, retriedTask)
			}
			mu.Unlock()
		}(i, taskID)
	}

	// 等待所有任务完成
	wg.Wait()

	return result, nil
}

// BatchProcessor 批处理器，用于处理大量数据
type BatchProcessor struct {
	client    *BatchClient
	batchSize int
	processor func(batch []*task.CreateRequest) error
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor(client *BatchClient, batchSize int) *BatchProcessor {
	return &BatchProcessor{
		client:    client,
		batchSize: batchSize,
	}
}

// ProcessCreateRequests 批量处理创建请求
func (bp *BatchProcessor) ProcessCreateRequests(ctx context.Context, requests []*task.CreateRequest, processor func(*BatchCreateResult) error) error {
	if len(requests) == 0 {
		return nil
	}

	// 分批处理
	for i := 0; i < len(requests); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(requests) {
			end = len(requests)
		}

		batch := requests[i:end]
		result, err := bp.client.CreateTasks(ctx, batch)
		if err != nil {
			return fmt.Errorf("batch %d-%d failed: %w", i, end-1, err)
		}

		if processor != nil {
			if err := processor(result); err != nil {
				return fmt.Errorf("processor failed for batch %d-%d: %w", i, end-1, err)
			}
		}
	}

	return nil
}

// StreamProcessor 流式批处理器
type StreamProcessor struct {
	client    *BatchClient
	batchSize int
	timeout   time.Duration
	buffer    []*task.CreateRequest
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	resultCh  chan *BatchCreateResult
	errorCh   chan error
}

// NewStreamProcessor 创建流式批处理器
func NewStreamProcessor(client *BatchClient, batchSize int, timeout time.Duration) *StreamProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	sp := &StreamProcessor{
		client:    client,
		batchSize: batchSize,
		timeout:   timeout,
		ctx:       ctx,
		cancel:    cancel,
		resultCh:  make(chan *BatchCreateResult, 10),
		errorCh:   make(chan error, 10),
	}

	// 启动定时刷新
	go sp.periodicFlush()

	return sp
}

// Add 添加任务到流处理器
func (sp *StreamProcessor) Add(request *task.CreateRequest) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	select {
	case <-sp.ctx.Done():
		return sp.ctx.Err()
	default:
	}

	sp.buffer = append(sp.buffer, request)

	// 如果缓冲区满了，立即处理
	if len(sp.buffer) >= sp.batchSize {
		return sp.flushLocked()
	}

	return nil
}

// Flush 手动刷新缓冲区
func (sp *StreamProcessor) Flush() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.flushLocked()
}

// flushLocked 内部刷新方法（需要持有锁）
func (sp *StreamProcessor) flushLocked() error {
	if len(sp.buffer) == 0 {
		return nil
	}

	batch := make([]*task.CreateRequest, len(sp.buffer))
	copy(batch, sp.buffer)
	sp.buffer = sp.buffer[:0]

	// 异步处理批次
	go func() {
		result, err := sp.client.CreateTasks(sp.ctx, batch)
		if err != nil {
			select {
			case sp.errorCh <- err:
			case <-sp.ctx.Done():
			}
		} else {
			select {
			case sp.resultCh <- result:
			case <-sp.ctx.Done():
			}
		}
	}()

	return nil
}

// periodicFlush 定期刷新
func (sp *StreamProcessor) periodicFlush() {
	ticker := time.NewTicker(sp.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sp.Flush()
		case <-sp.ctx.Done():
			return
		}
	}
}

// Results 获取结果通道
func (sp *StreamProcessor) Results() <-chan *BatchCreateResult {
	return sp.resultCh
}

// Errors 获取错误通道
func (sp *StreamProcessor) Errors() <-chan error {
	return sp.errorCh
}

// Close 关闭流处理器
func (sp *StreamProcessor) Close() error {
	// 刷新剩余的缓冲区
	if err := sp.Flush(); err != nil {
		sp.cancel()
		return err
	}

	sp.cancel()
	close(sp.resultCh)
	close(sp.errorCh)
	return nil
}

// 批量操作统计
type BatchStats struct {
	TotalRequested int
	Successful     int
	Failed         int
	Duration       time.Duration
	ErrorRate      float64
}

// CalculateStats 计算批量操作统计
func CalculateStats(result interface{}, duration time.Duration) *BatchStats {
	stats := &BatchStats{
		Duration: duration,
	}

	switch r := result.(type) {
	case *BatchCreateResult:
		stats.TotalRequested = r.Total
		stats.Successful = len(r.Success)
		stats.Failed = len(r.Failed)
	case *BatchUpdateResult:
		stats.TotalRequested = r.Total
		stats.Successful = len(r.Success)
		stats.Failed = len(r.Failed)
	case *BatchDeleteResult:
		stats.TotalRequested = r.Total
		stats.Successful = len(r.Success)
		stats.Failed = len(r.Failed)
	case *BatchQueryResult:
		stats.TotalRequested = r.Total
		stats.Successful = len(r.Success)
		stats.Failed = len(r.Failed)
	}

	if stats.TotalRequested > 0 {
		stats.ErrorRate = float64(stats.Failed) / float64(stats.TotalRequested)
	}

	return stats
}

// String 返回统计信息的字符串表示
func (s *BatchStats) String() string {
	return fmt.Sprintf("BatchStats{Total: %d, Success: %d, Failed: %d, ErrorRate: %.2f%%, Duration: %v}",
		s.TotalRequested, s.Successful, s.Failed, s.ErrorRate*100, s.Duration)
}