package builder

import (
	"context"
	"time"

	"task-center/sdk/task"
)

// QueryBuilder 查询构建器，提供链式调用接口
type QueryBuilder struct {
	client  *task.Client
	request *task.ListRequest
	context context.Context
}

// NewQueryBuilder 创建新的查询构建器
func NewQueryBuilder(client *task.Client) *QueryBuilder {
	return &QueryBuilder{
		client:  client,
		request: task.NewListRequest(),
		context: context.Background(),
	}
}

// WithContext 设置查询上下文
func (q *QueryBuilder) WithContext(ctx context.Context) *QueryBuilder {
	q.context = ctx
	return q
}

// WithStatus 添加状态过滤条件
func (q *QueryBuilder) WithStatus(statuses ...task.TaskStatus) *QueryBuilder {
	q.request = q.request.WithStatus(statuses...)
	return q
}

// WithPendingStatus 查询待执行任务
func (q *QueryBuilder) WithPendingStatus() *QueryBuilder {
	return q.WithStatus(task.StatusPending)
}

// WithRunningStatus 查询正在执行的任务
func (q *QueryBuilder) WithRunningStatus() *QueryBuilder {
	return q.WithStatus(task.StatusRunning)
}

// WithSucceededStatus 查询成功完成的任务
func (q *QueryBuilder) WithSucceededStatus() *QueryBuilder {
	return q.WithStatus(task.StatusSucceeded)
}

// WithFailedStatus 查询失败的任务
func (q *QueryBuilder) WithFailedStatus() *QueryBuilder {
	return q.WithStatus(task.StatusFailed)
}

// WithCancelledStatus 查询已取消的任务
func (q *QueryBuilder) WithCancelledStatus() *QueryBuilder {
	return q.WithStatus(task.StatusCancelled)
}

// WithCompletedStatus 查询已完成的任务（成功、失败、取消、过期）
func (q *QueryBuilder) WithCompletedStatus() *QueryBuilder {
	return q.WithStatus(task.StatusSucceeded, task.StatusFailed, task.StatusCancelled, task.StatusExpired)
}

// WithActiveStatus 查询活跃任务（待执行、正在执行）
func (q *QueryBuilder) WithActiveStatus() *QueryBuilder {
	return q.WithStatus(task.StatusPending, task.StatusRunning)
}

// WithTags 添加标签过滤条件
func (q *QueryBuilder) WithTags(tags ...string) *QueryBuilder {
	q.request = q.request.WithTags(tags...)
	return q
}

// WithTag 添加单个标签过滤条件
func (q *QueryBuilder) WithTag(tag string) *QueryBuilder {
	return q.WithTags(tag)
}

// WithPriority 添加优先级过滤条件
func (q *QueryBuilder) WithPriority(priority task.TaskPriority) *QueryBuilder {
	q.request = q.request.WithPriority(priority)
	return q
}

// WithHighPriority 查询高优先级任务
func (q *QueryBuilder) WithHighPriority() *QueryBuilder {
	return q.WithPriority(task.PriorityHigh)
}

// WithNormalPriority 查询普通优先级任务
func (q *QueryBuilder) WithNormalPriority() *QueryBuilder {
	return q.WithPriority(task.PriorityNormal)
}

// WithLowPriority 查询低优先级任务
func (q *QueryBuilder) WithLowPriority() *QueryBuilder {
	return q.WithPriority(task.PriorityLow)
}

// WithCreatedTimeRange 添加创建时间范围过滤条件
func (q *QueryBuilder) WithCreatedTimeRange(from, to *time.Time) *QueryBuilder {
	q.request = q.request.WithCreatedTimeRange(from, to)
	return q
}

// WithCreatedAfter 查询指定时间之后创建的任务
func (q *QueryBuilder) WithCreatedAfter(t time.Time) *QueryBuilder {
	return q.WithCreatedTimeRange(&t, nil)
}

// WithCreatedBefore 查询指定时间之前创建的任务
func (q *QueryBuilder) WithCreatedBefore(t time.Time) *QueryBuilder {
	return q.WithCreatedTimeRange(nil, &t)
}

// WithCreatedToday 查询今天创建的任务
func (q *QueryBuilder) WithCreatedToday() *QueryBuilder {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return q.WithCreatedTimeRange(&startOfDay, &endOfDay)
}

// WithCreatedThisWeek 查询本周创建的任务
func (q *QueryBuilder) WithCreatedThisWeek() *QueryBuilder {
	now := time.Now()
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -int(weekday-1))
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
	endOfWeek := startOfWeek.Add(7 * 24 * time.Hour)
	return q.WithCreatedTimeRange(&startOfWeek, &endOfWeek)
}

// WithCreatedThisMonth 查询本月创建的任务
func (q *QueryBuilder) WithCreatedThisMonth() *QueryBuilder {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	return q.WithCreatedTimeRange(&startOfMonth, &endOfMonth)
}

// WithLast24Hours 查询最近24小时创建的任务
func (q *QueryBuilder) WithLast24Hours() *QueryBuilder {
	now := time.Now()
	past24Hours := now.Add(-24 * time.Hour)
	return q.WithCreatedTimeRange(&past24Hours, &now)
}

// WithLastHour 查询最近1小时创建的任务
func (q *QueryBuilder) WithLastHour() *QueryBuilder {
	now := time.Now()
	pastHour := now.Add(-1 * time.Hour)
	return q.WithCreatedTimeRange(&pastHour, &now)
}

// WithPagination 设置分页参数
func (q *QueryBuilder) WithPagination(page, pageSize int) *QueryBuilder {
	q.request = q.request.WithPagination(page, pageSize)
	return q
}

// WithPage 设置页码
func (q *QueryBuilder) WithPage(page int) *QueryBuilder {
	q.request.Page = page
	return q
}

// WithPageSize 设置每页大小
func (q *QueryBuilder) WithPageSize(pageSize int) *QueryBuilder {
	q.request.PageSize = pageSize
	return q
}

// WithLimit 设置查询限制（等同于设置pageSize为指定值，page为1）
func (q *QueryBuilder) WithLimit(limit int) *QueryBuilder {
	return q.WithPagination(1, limit)
}

// FirstPage 设置为第一页
func (q *QueryBuilder) FirstPage() *QueryBuilder {
	return q.WithPage(1)
}

// NextPage 移动到下一页
func (q *QueryBuilder) NextPage() *QueryBuilder {
	if q.request.Page <= 0 {
		q.request.Page = 1
	}
	q.request.Page++
	return q
}

// PrevPage 移动到上一页
func (q *QueryBuilder) PrevPage() *QueryBuilder {
	if q.request.Page > 1 {
		q.request.Page--
	}
	return q
}

// Execute 执行查询
func (q *QueryBuilder) Execute() (*task.ListResponse, error) {
	return q.client.ListTasks(q.context, q.request)
}

// Count 获取查询结果总数（执行查询并返回总数）
func (q *QueryBuilder) Count() (int, error) {
	// 保存原始分页设置
	originalPage := q.request.Page
	originalPageSize := q.request.PageSize

	// 设置为获取第一页，每页1条记录（只为了获取总数）
	q.request.Page = 1
	q.request.PageSize = 1

	resp, err := q.Execute()

	// 恢复原始分页设置
	q.request.Page = originalPage
	q.request.PageSize = originalPageSize

	if err != nil {
		return 0, err
	}

	return resp.Total, nil
}

// First 获取第一个匹配的任务
func (q *QueryBuilder) First() (*task.Task, error) {
	// 保存原始分页设置
	originalPage := q.request.Page
	originalPageSize := q.request.PageSize

	// 设置为获取第一页，每页1条记录
	q.request.Page = 1
	q.request.PageSize = 1

	resp, err := q.Execute()

	// 恢复原始分页设置
	q.request.Page = originalPage
	q.request.PageSize = originalPageSize

	if err != nil {
		return nil, err
	}

	if len(resp.Tasks) == 0 {
		return nil, nil
	}

	return resp.Tasks[0], nil
}

// GetAll 获取所有匹配的任务（自动分页获取）
func (q *QueryBuilder) GetAll() ([]*task.Task, error) {
	var allTasks []*task.Task
	page := 1
	pageSize := 100 // 每次获取100条记录

	for {
		// 保存原始分页设置
		originalPage := q.request.Page
		originalPageSize := q.request.PageSize

		// 设置当前分页
		q.request.Page = page
		q.request.PageSize = pageSize

		resp, err := q.Execute()

		// 恢复原始分页设置
		q.request.Page = originalPage
		q.request.PageSize = originalPageSize

		if err != nil {
			return nil, err
		}

		allTasks = append(allTasks, resp.Tasks...)

		// 如果已经获取完所有数据，退出循环
		if len(resp.Tasks) < pageSize || len(allTasks) >= resp.Total {
			break
		}

		page++
	}

	return allTasks, nil
}

// Exists 检查是否存在匹配的任务
func (q *QueryBuilder) Exists() (bool, error) {
	task, err := q.First()
	if err != nil {
		return false, err
	}
	return task != nil, nil
}

// Clone 克隆查询构建器
func (q *QueryBuilder) Clone() *QueryBuilder {
	newBuilder := &QueryBuilder{
		client:  q.client,
		context: q.context,
	}

	// 深拷贝查询请求
	newBuilder.request = &task.ListRequest{
		Page:     q.request.Page,
		PageSize: q.request.PageSize,
	}

	if q.request.Status != nil {
		newBuilder.request.Status = make([]task.TaskStatus, len(q.request.Status))
		copy(newBuilder.request.Status, q.request.Status)
	}

	if q.request.Tags != nil {
		newBuilder.request.Tags = make([]string, len(q.request.Tags))
		copy(newBuilder.request.Tags, q.request.Tags)
	}

	if q.request.Priority != nil {
		priority := *q.request.Priority
		newBuilder.request.Priority = &priority
	}

	if q.request.CreatedFrom != nil {
		createdFrom := *q.request.CreatedFrom
		newBuilder.request.CreatedFrom = &createdFrom
	}

	if q.request.CreatedTo != nil {
		createdTo := *q.request.CreatedTo
		newBuilder.request.CreatedTo = &createdTo
	}

	return newBuilder
}

// Reset 重置查询构建器
func (q *QueryBuilder) Reset() *QueryBuilder {
	q.request = task.NewListRequest()
	q.context = context.Background()
	return q
}

// GetRequest 获取构建的查询请求（用于调试）
func (q *QueryBuilder) GetRequest() *task.ListRequest {
	return q.request
}

// Search 基于查询字符串搜索任务
func (q *QueryBuilder) Search(query string) (*task.ListResponse, error) {
	return q.client.SearchTasks(q.context, query, q.request)
}

// 预定义查询模板
type QueryTemplate struct {
	name    string
	builder func(*QueryBuilder) *QueryBuilder
}

// 常用查询模板
var (
	// PendingTasksTemplate 待执行任务查询模板
	PendingTasksTemplate = &QueryTemplate{
		name: "pending_tasks",
		builder: func(q *QueryBuilder) *QueryBuilder {
			return q.WithPendingStatus().WithPageSize(50)
		},
	}

	// FailedTasksTemplate 失败任务查询模板
	FailedTasksTemplate = &QueryTemplate{
		name: "failed_tasks",
		builder: func(q *QueryBuilder) *QueryBuilder {
			return q.WithFailedStatus().WithPageSize(50)
		},
	}

	// RecentTasksTemplate 最近任务查询模板
	RecentTasksTemplate = &QueryTemplate{
		name: "recent_tasks",
		builder: func(q *QueryBuilder) *QueryBuilder {
			return q.WithLast24Hours().WithPageSize(50)
		},
	}

	// HighPriorityTasksTemplate 高优先级任务查询模板
	HighPriorityTasksTemplate = &QueryTemplate{
		name: "high_priority_tasks",
		builder: func(q *QueryBuilder) *QueryBuilder {
			return q.WithHighPriority().WithActiveStatus().WithPageSize(50)
		},
	}
)

// Apply 应用查询模板
func (t *QueryTemplate) Apply(builder *QueryBuilder) *QueryBuilder {
	return t.builder(builder)
}

// CreateBuilder 基于模板创建查询构建器
func (t *QueryTemplate) CreateBuilder(client *task.Client) *QueryBuilder {
	builder := NewQueryBuilder(client)
	return t.Apply(builder)
}

// QuickQuery 便捷查询方法集
type QuickQuery struct {
	client *task.Client
}

// NewQuickQuery 创建便捷查询
func NewQuickQuery(client *task.Client) *QuickQuery {
	return &QuickQuery{client: client}
}

// Pending 获取待执行任务
func (q *QuickQuery) Pending() *QueryBuilder {
	return PendingTasksTemplate.CreateBuilder(q.client)
}

// Failed 获取失败任务
func (q *QuickQuery) Failed() *QueryBuilder {
	return FailedTasksTemplate.CreateBuilder(q.client)
}

// Recent 获取最近任务
func (q *QuickQuery) Recent() *QueryBuilder {
	return RecentTasksTemplate.CreateBuilder(q.client)
}

// HighPriority 获取高优先级任务
func (q *QuickQuery) HighPriority() *QueryBuilder {
	return HighPriorityTasksTemplate.CreateBuilder(q.client)
}

// ByStatus 根据状态查询
func (q *QuickQuery) ByStatus(status task.TaskStatus) *QueryBuilder {
	return NewQueryBuilder(q.client).WithStatus(status)
}

// ByTag 根据标签查询
func (q *QuickQuery) ByTag(tag string) *QueryBuilder {
	return NewQueryBuilder(q.client).WithTag(tag)
}

// ByPriority 根据优先级查询
func (q *QuickQuery) ByPriority(priority task.TaskPriority) *QueryBuilder {
	return NewQueryBuilder(q.client).WithPriority(priority)
}

// Today 获取今天的任务
func (q *QuickQuery) Today() *QueryBuilder {
	return NewQueryBuilder(q.client).WithCreatedToday()
}

// ThisWeek 获取本周的任务
func (q *QuickQuery) ThisWeek() *QueryBuilder {
	return NewQueryBuilder(q.client).WithCreatedThisWeek()
}

// ThisMonth 获取本月的任务
func (q *QuickQuery) ThisMonth() *QueryBuilder {
	return NewQueryBuilder(q.client).WithCreatedThisMonth()
}