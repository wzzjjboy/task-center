package sdk

import (
	"encoding/json"
	"time"
)

// TaskStatus 任务状态枚举
type TaskStatus int

const (
	TaskStatusPending    TaskStatus = 0 // 待执行
	TaskStatusRunning    TaskStatus = 1 // 执行中
	TaskStatusSucceeded  TaskStatus = 2 // 成功
	TaskStatusFailed     TaskStatus = 3 // 失败
	TaskStatusCancelled  TaskStatus = 4 // 取消
	TaskStatusExpired    TaskStatus = 5 // 过期
)

// String 返回任务状态的字符串表示
func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusSucceeded:
		return "succeeded"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusCancelled:
		return "cancelled"
	case TaskStatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// TaskPriority 任务优先级
type TaskPriority int

const (
	TaskPriorityHighest TaskPriority = 1
	TaskPriorityHigh    TaskPriority = 3
	TaskPriorityNormal  TaskPriority = 5
	TaskPriorityLow     TaskPriority = 7
	TaskPriorityLowest  TaskPriority = 9
)

// Task 任务结构
type Task struct {
	ID               int64             `json:"id,omitempty"`
	BusinessUniqueID string            `json:"business_unique_id"`
	CallbackURL      string            `json:"callback_url"`
	CallbackMethod   string            `json:"callback_method,omitempty"`
	CallbackHeaders  map[string]string `json:"callback_headers,omitempty"`
	CallbackBody     string            `json:"callback_body,omitempty"`
	RetryIntervals   []int             `json:"retry_intervals,omitempty"`
	MaxRetries       int               `json:"max_retries,omitempty"`
	CurrentRetry     int               `json:"current_retry,omitempty"`
	Status           TaskStatus        `json:"status,omitempty"`
	Priority         TaskPriority      `json:"priority,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	Timeout          int               `json:"timeout,omitempty"`
	ScheduledAt      time.Time         `json:"scheduled_at,omitempty"`
	NextExecuteAt    *time.Time        `json:"next_execute_at,omitempty"`
	ExecutedAt       *time.Time        `json:"executed_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at,omitempty"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	BusinessUniqueID string                 `json:"business_unique_id"`
	CallbackURL      string                 `json:"callback_url"`
	CallbackMethod   string                 `json:"callback_method,omitempty"`
	CallbackHeaders  map[string]string      `json:"callback_headers,omitempty"`
	CallbackBody     string                 `json:"callback_body,omitempty"`
	RetryIntervals   []int                  `json:"retry_intervals,omitempty"`
	MaxRetries       int                    `json:"max_retries,omitempty"`
	Priority         TaskPriority           `json:"priority,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Timeout          int                    `json:"timeout,omitempty"`
	ScheduledAt      *time.Time             `json:"scheduled_at,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	CallbackURL     *string                `json:"callback_url,omitempty"`
	CallbackMethod  *string                `json:"callback_method,omitempty"`
	CallbackHeaders map[string]string      `json:"callback_headers,omitempty"`
	CallbackBody    *string                `json:"callback_body,omitempty"`
	RetryIntervals  []int                  `json:"retry_intervals,omitempty"`
	MaxRetries      *int                   `json:"max_retries,omitempty"`
	Priority        *TaskPriority          `json:"priority,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Timeout         *int                   `json:"timeout,omitempty"`
	ScheduledAt     *time.Time             `json:"scheduled_at,omitempty"`
	Status          *TaskStatus            `json:"status,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ListTasksRequest 查询任务列表请求
type ListTasksRequest struct {
	Status      []TaskStatus `json:"status,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Priority    *TaskPriority `json:"priority,omitempty"`
	CreatedFrom *time.Time   `json:"created_from,omitempty"`
	CreatedTo   *time.Time   `json:"created_to,omitempty"`
	Page        int          `json:"page,omitempty"`
	PageSize    int          `json:"page_size,omitempty"`
}

// ListTasksResponse 任务列表响应
type ListTasksResponse struct {
	Tasks      []Task `json:"tasks"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

// TaskStatsResponse 任务统计响应
type TaskStatsResponse struct {
	TotalTasks     int                    `json:"total_tasks"`
	StatusCounts   map[TaskStatus]int     `json:"status_counts"`
	PriorityCounts map[TaskPriority]int   `json:"priority_counts"`
	TagCounts      map[string]int         `json:"tag_counts"`
}

// ApiResponse 通用API响应结构
type ApiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Details interface{} `json:"details,omitempty"`
}

// CallbackEvent 回调事件结构
type CallbackEvent struct {
	EventType   string    `json:"event_type"`   // task.created, task.started, task.completed, task.failed
	EventTime   time.Time `json:"event_time"`
	TaskID      int64     `json:"task_id"`
	BusinessID  int64     `json:"business_id"`
	Task        Task      `json:"task"`
	Signature   string    `json:"signature"`    // 回调签名，用于验证
}

// BatchCreateTasksRequest 批量创建任务请求
type BatchCreateTasksRequest struct {
	Tasks []CreateTaskRequest `json:"tasks"`
}

// BatchCreateTasksResponse 批量创建任务响应
type BatchCreateTasksResponse struct {
	Succeeded []Task           `json:"succeeded"`
	Failed    []BatchTaskError `json:"failed"`
}

// BatchTaskError 批量操作中的任务错误
type BatchTaskError struct {
	Index   int    `json:"index"`
	Error   string `json:"error"`
	Code    string `json:"code"`
	Request CreateTaskRequest `json:"request"`
}

// Validate 验证创建任务请求
func (req *CreateTaskRequest) Validate() error {
	if req.BusinessUniqueID == "" {
		return NewValidationError("business_unique_id is required")
	}
	if req.CallbackURL == "" {
		return NewValidationError("callback_url is required")
	}
	if req.CallbackMethod == "" {
		req.CallbackMethod = "POST" // 默认POST方法
	}
	if req.Priority == 0 {
		req.Priority = TaskPriorityNormal // 默认正常优先级
	}
	if req.Timeout <= 0 {
		req.Timeout = 300 // 默认5分钟超时
	}
	if req.MaxRetries < 0 {
		req.MaxRetries = 3 // 默认重试3次
	}
	return nil
}

// MarshalJSON 自定义JSON序列化，处理时间格式
func (t *Task) MarshalJSON() ([]byte, error) {
	type Alias Task
	return json.Marshal(&struct {
		*Alias
		ScheduledAt   string  `json:"scheduled_at,omitempty"`
		NextExecuteAt *string `json:"next_execute_at,omitempty"`
		ExecutedAt    *string `json:"executed_at,omitempty"`
		CompletedAt   *string `json:"completed_at,omitempty"`
		CreatedAt     string  `json:"created_at,omitempty"`
		UpdatedAt     string  `json:"updated_at,omitempty"`
	}{
		Alias:       (*Alias)(t),
		ScheduledAt: t.ScheduledAt.Format(time.RFC3339),
		NextExecuteAt: func() *string {
			if t.NextExecuteAt != nil {
				s := t.NextExecuteAt.Format(time.RFC3339)
				return &s
			}
			return nil
		}(),
		ExecutedAt: func() *string {
			if t.ExecutedAt != nil {
				s := t.ExecutedAt.Format(time.RFC3339)
				return &s
			}
			return nil
		}(),
		CompletedAt: func() *string {
			if t.CompletedAt != nil {
				s := t.CompletedAt.Format(time.RFC3339)
				return &s
			}
			return nil
		}(),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	})
}