package async

import (
	"context"
	"sync"
	"time"

	"task-center/sdk/task"
)

// AsyncClient 异步任务客户端
type AsyncClient struct {
	client    *task.Client
	workers   int
	taskChan  chan *AsyncTask
	resultMap sync.Map
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	started   bool
	mu        sync.RWMutex
}

// AsyncTask 异步任务
type AsyncTask struct {
	ID       string
	Request  *task.CreateRequest
	Callback func(*TaskResult)
	Created  time.Time
}

// TaskResult 任务结果
type TaskResult struct {
	Task  *task.Task
	Error error
}

// Future 异步任务的未来结果
type Future struct {
	id     string
	client *AsyncClient
	done   chan struct{}
	result *TaskResult
	mu     sync.RWMutex
}

// AsyncClientConfig 异步客户端配置
type AsyncClientConfig struct {
	Workers    int           // 工作线程数
	BufferSize int           // 任务缓冲区大小
	Timeout    time.Duration // 默认任务超时时间
}

// DefaultAsyncConfig 默认异步客户端配置
func DefaultAsyncConfig() *AsyncClientConfig {
	return &AsyncClientConfig{
		Workers:    10,
		BufferSize: 100,
		Timeout:    30 * time.Second,
	}
}

// NewAsyncClient 创建异步任务客户端
func NewAsyncClient(client *task.Client, config *AsyncClientConfig) *AsyncClient {
	if config == nil {
		config = DefaultAsyncConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncClient{
		client:   client,
		workers:  config.Workers,
		taskChan: make(chan *AsyncTask, config.BufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动异步客户端
func (ac *AsyncClient) Start() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.started {
		return
	}

	ac.started = true

	// 启动工作线程
	for i := 0; i < ac.workers; i++ {
		ac.wg.Add(1)
		go ac.worker()
	}
}

// Stop 停止异步客户端
func (ac *AsyncClient) Stop() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.started {
		return
	}

	ac.cancel()
	close(ac.taskChan)
	ac.wg.Wait()
	ac.started = false
}

// worker 工作线程
func (ac *AsyncClient) worker() {
	defer ac.wg.Done()

	for {
		select {
		case asyncTask, ok := <-ac.taskChan:
			if !ok {
				return
			}
			ac.processTask(asyncTask)

		case <-ac.ctx.Done():
			return
		}
	}
}

// processTask 处理异步任务
func (ac *AsyncClient) processTask(asyncTask *AsyncTask) {
	ctx, cancel := context.WithTimeout(ac.ctx, 30*time.Second)
	defer cancel()

	// 执行任务创建
	task, err := ac.client.CreateTask(ctx, asyncTask.Request)

	result := &TaskResult{
		Task:  task,
		Error: err,
	}

	// 存储结果
	ac.resultMap.Store(asyncTask.ID, result)

	// 调用回调函数
	if asyncTask.Callback != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 忽略回调函数中的panic
				}
			}()
			asyncTask.Callback(result)
		}()
	}

	// 通知Future完成
	if futureInterface, ok := ac.resultMap.Load("future_" + asyncTask.ID); ok {
		if future, ok := futureInterface.(*Future); ok {
			future.mu.Lock()
			future.result = result
			close(future.done)
			future.mu.Unlock()
		}
	}
}

// CreateTaskAsync 异步创建任务
func (ac *AsyncClient) CreateTaskAsync(request *task.CreateRequest, callback func(*TaskResult)) (string, error) {
	ac.mu.RLock()
	started := ac.started
	ac.mu.RUnlock()

	if !started {
		ac.Start()
	}

	id := generateTaskID()
	asyncTask := &AsyncTask{
		ID:       id,
		Request:  request,
		Callback: callback,
		Created:  time.Now(),
	}

	select {
	case ac.taskChan <- asyncTask:
		return id, nil
	case <-ac.ctx.Done():
		return "", ac.ctx.Err()
	default:
		return "", ErrTaskQueueFull
	}
}

// CreateTaskFuture 创建任务并返回Future
func (ac *AsyncClient) CreateTaskFuture(request *task.CreateRequest) *Future {
	id := generateTaskID()
	future := &Future{
		id:     id,
		client: ac,
		done:   make(chan struct{}),
	}

	// 存储Future引用
	ac.resultMap.Store("future_"+id, future)

	// 异步创建任务
	ac.CreateTaskAsync(request, nil)

	return future
}

// GetResult 获取异步任务结果
func (ac *AsyncClient) GetResult(id string) (*TaskResult, bool) {
	if result, ok := ac.resultMap.Load(id); ok {
		return result.(*TaskResult), true
	}
	return nil, false
}

// GetAllResults 获取所有结果
func (ac *AsyncClient) GetAllResults() map[string]*TaskResult {
	results := make(map[string]*TaskResult)
	ac.resultMap.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok && keyStr != "" && keyStr[0:7] != "future_" {
			results[keyStr] = value.(*TaskResult)
		}
		return true
	})
	return results
}

// ClearResults 清理结果
func (ac *AsyncClient) ClearResults() {
	ac.resultMap.Range(func(key, value interface{}) bool {
		ac.resultMap.Delete(key)
		return true
	})
}

// GetQueueLength 获取当前队列长度
func (ac *AsyncClient) GetQueueLength() int {
	return len(ac.taskChan)
}

// IsStarted 检查是否已启动
func (ac *AsyncClient) IsStarted() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.started
}

// Future 方法实现

// Get 获取Future结果（阻塞等待）
func (f *Future) Get() (*TaskResult, error) {
	<-f.done
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.result, nil
}

// GetWithTimeout 带超时的获取Future结果
func (f *Future) GetWithTimeout(timeout time.Duration) (*TaskResult, error) {
	select {
	case <-f.done:
		f.mu.RLock()
		defer f.mu.RUnlock()
		return f.result, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}

// IsDone 检查Future是否完成
func (f *Future) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// Cancel 取消Future（实际上无法取消已提交的任务）
func (f *Future) Cancel() bool {
	// 这里我们只能从结果映射中删除，无法真正取消任务
	f.client.resultMap.Delete(f.id)
	f.client.resultMap.Delete("future_" + f.id)
	return false
}

// TaskGroup 任务组，用于批量管理异步任务
type TaskGroup struct {
	client  *AsyncClient
	futures []*Future
	mu      sync.RWMutex
}

// NewTaskGroup 创建任务组
func NewTaskGroup(client *AsyncClient) *TaskGroup {
	return &TaskGroup{
		client: client,
	}
}

// Add 添加任务到组
func (tg *TaskGroup) Add(request *task.CreateRequest) {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	future := tg.client.CreateTaskFuture(request)
	tg.futures = append(tg.futures, future)
}

// Wait 等待所有任务完成
func (tg *TaskGroup) Wait() []*TaskResult {
	tg.mu.RLock()
	futures := make([]*Future, len(tg.futures))
	copy(futures, tg.futures)
	tg.mu.RUnlock()

	results := make([]*TaskResult, len(futures))
	for i, future := range futures {
		result, _ := future.Get()
		results[i] = result
	}

	return results
}

// WaitWithTimeout 带超时的等待所有任务完成
func (tg *TaskGroup) WaitWithTimeout(timeout time.Duration) ([]*TaskResult, error) {
	tg.mu.RLock()
	futures := make([]*Future, len(tg.futures))
	copy(futures, tg.futures)
	tg.mu.RUnlock()

	results := make([]*TaskResult, len(futures))
	deadline := time.Now().Add(timeout)

	for i, future := range futures {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, ErrTimeout
		}

		result, err := future.GetWithTimeout(remaining)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

// Size 获取任务组大小
func (tg *TaskGroup) Size() int {
	tg.mu.RLock()
	defer tg.mu.RUnlock()
	return len(tg.futures)
}

// Clear 清空任务组
func (tg *TaskGroup) Clear() {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.futures = tg.futures[:0]
}

// Pipeline 任务管道，用于流式处理
type Pipeline struct {
	client *AsyncClient
	stages []func(*task.CreateRequest) *task.CreateRequest
	mu     sync.RWMutex
}

// NewPipeline 创建任务管道
func NewPipeline(client *AsyncClient) *Pipeline {
	return &Pipeline{
		client: client,
	}
}

// AddStage 添加处理阶段
func (p *Pipeline) AddStage(stage func(*task.CreateRequest) *task.CreateRequest) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, stage)
	return p
}

// Process 处理任务请求
func (p *Pipeline) Process(request *task.CreateRequest) *Future {
	p.mu.RLock()
	stages := make([]func(*task.CreateRequest) *task.CreateRequest, len(p.stages))
	copy(stages, p.stages)
	p.mu.RUnlock()

	// 依次经过所有阶段处理
	for _, stage := range stages {
		request = stage(request)
		if request == nil {
			// 如果某个阶段返回nil，创建一个立即失败的Future
			future := &Future{
				id:     generateTaskID(),
				client: p.client,
				done:   make(chan struct{}),
				result: &TaskResult{Error: ErrPipelineStageError},
			}
			close(future.done)
			return future
		}
	}

	return p.client.CreateTaskFuture(request)
}

// WorkerPool 工作池，用于控制并发度
type WorkerPool struct {
	client     *AsyncClient
	semaphore  chan struct{}
	activeTasks sync.WaitGroup
}

// NewWorkerPool 创建工作池
func NewWorkerPool(client *AsyncClient, maxConcurrency int) *WorkerPool {
	return &WorkerPool{
		client:    client,
		semaphore: make(chan struct{}, maxConcurrency),
	}
}

// Submit 提交任务到工作池
func (wp *WorkerPool) Submit(request *task.CreateRequest, callback func(*TaskResult)) error {
	// 获取信号量
	select {
	case wp.semaphore <- struct{}{}:
		wp.activeTasks.Add(1)

		go func() {
			defer func() {
				<-wp.semaphore
				wp.activeTasks.Done()
			}()

			// 创建包装的回调函数
			wrappedCallback := func(result *TaskResult) {
				if callback != nil {
					callback(result)
				}
			}

			wp.client.CreateTaskAsync(request, wrappedCallback)
		}()

		return nil
	default:
		return ErrWorkerPoolFull
	}
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	wp.activeTasks.Wait()
}

// Close 关闭工作池
func (wp *WorkerPool) Close() {
	wp.Wait()
	close(wp.semaphore)
}