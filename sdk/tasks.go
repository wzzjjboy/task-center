package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// TaskService 任务服务接口
type TaskService interface {
	Create(ctx context.Context, req *CreateTaskRequest) (*Task, error)
	Get(ctx context.Context, taskID int64) (*Task, error)
	GetByBusinessUniqueID(ctx context.Context, businessUniqueID string) (*Task, error)
	Update(ctx context.Context, taskID int64, req *UpdateTaskRequest) (*Task, error)
	Delete(ctx context.Context, taskID int64) error
	List(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error)
	Stats(ctx context.Context) (*TaskStatsResponse, error)
	BatchCreate(ctx context.Context, req *BatchCreateTasksRequest) (*BatchCreateTasksResponse, error)
	Cancel(ctx context.Context, taskID int64) error
	Retry(ctx context.Context, taskID int64) error
}

// taskService 任务服务实现
type taskService struct {
	client *Client
}

// newTaskService 创建任务服务实例
func newTaskService(client *Client) TaskService {
	return &taskService{client: client}
}

// Create 创建任务
func (s *taskService) Create(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := s.client.doRequest(ctx, "POST", "/api/v1/tasks", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	taskData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// Get 根据ID获取任务
func (s *taskService) Get(ctx context.Context, taskID int64) (*Task, error) {
	path := fmt.Sprintf("/api/v1/tasks/%d", taskID)
	resp, err := s.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewNotFoundError("task")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	taskData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// GetByBusinessUniqueID 根据业务唯一ID获取任务
func (s *taskService) GetByBusinessUniqueID(ctx context.Context, businessUniqueID string) (*Task, error) {
	path := fmt.Sprintf("/api/v1/tasks/by-business-id/%s", url.QueryEscape(businessUniqueID))
	resp, err := s.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewNotFoundError("task")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	taskData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// Update 更新任务
func (s *taskService) Update(ctx context.Context, taskID int64, req *UpdateTaskRequest) (*Task, error) {
	path := fmt.Sprintf("/api/v1/tasks/%d", taskID)
	resp, err := s.client.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewNotFoundError("task")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	taskData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// Delete 删除任务
func (s *taskService) Delete(ctx context.Context, taskID int64) error {
	path := fmt.Sprintf("/api/v1/tasks/%d", taskID)
	resp, err := s.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return NewNotFoundError("task")
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return s.handleErrorResponse(resp)
	}

	return nil
}

// List 获取任务列表
func (s *taskService) List(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error) {
	if req == nil {
		req = &ListTasksRequest{}
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("page", strconv.Itoa(req.Page))
	params.Set("page_size", strconv.Itoa(req.PageSize))

	if len(req.Status) > 0 {
		statuses := make([]string, len(req.Status))
		for i, status := range req.Status {
			statuses[i] = strconv.Itoa(int(status))
		}
		params.Set("status", strings.Join(statuses, ","))
	}

	if len(req.Tags) > 0 {
		params.Set("tags", strings.Join(req.Tags, ","))
	}

	if req.Priority != nil {
		params.Set("priority", strconv.Itoa(int(*req.Priority)))
	}

	if req.CreatedFrom != nil {
		params.Set("created_from", req.CreatedFrom.Format("2006-01-02T15:04:05Z"))
	}

	if req.CreatedTo != nil {
		params.Set("created_to", req.CreatedTo.Format("2006-01-02T15:04:05Z"))
	}

	path := "/api/v1/tasks"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := s.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	listData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list data: %w", err)
	}

	var listResp ListTasksResponse
	if err := json.Unmarshal(listData, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	return &listResp, nil
}

// Stats 获取任务统计信息
func (s *taskService) Stats(ctx context.Context) (*TaskStatsResponse, error) {
	resp, err := s.client.doRequest(ctx, "GET", "/api/v1/tasks/stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	statsData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stats data: %w", err)
	}

	var stats TaskStatsResponse
	if err := json.Unmarshal(statsData, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}

// BatchCreate 批量创建任务
func (s *taskService) BatchCreate(ctx context.Context, req *BatchCreateTasksRequest) (*BatchCreateTasksResponse, error) {
	if req == nil || len(req.Tasks) == 0 {
		return nil, NewValidationError("at least one task is required")
	}

	// 验证所有任务
	for i, task := range req.Tasks {
		if err := task.Validate(); err != nil {
			return nil, NewValidationErrorWithDetails(
				fmt.Sprintf("validation failed for task at index %d", i),
				err.Error(),
			)
		}
	}

	resp, err := s.client.doRequest(ctx, "POST", "/api/v1/tasks/batch", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, s.handleErrorResponse(resp)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	batchData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch data: %w", err)
	}

	var batchResp BatchCreateTasksResponse
	if err := json.Unmarshal(batchData, &batchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch response: %w", err)
	}

	return &batchResp, nil
}

// Cancel 取消任务
func (s *taskService) Cancel(ctx context.Context, taskID int64) error {
	path := fmt.Sprintf("/api/v1/tasks/%d/cancel", taskID)
	resp, err := s.client.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return NewNotFoundError("task")
	}

	if resp.StatusCode != http.StatusOK {
		return s.handleErrorResponse(resp)
	}

	return nil
}

// Retry 重试任务
func (s *taskService) Retry(ctx context.Context, taskID int64) error {
	path := fmt.Sprintf("/api/v1/tasks/%d/retry", taskID)
	resp, err := s.client.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return NewNotFoundError("task")
	}

	if resp.StatusCode != http.StatusOK {
		return s.handleErrorResponse(resp)
	}

	return nil
}

// handleErrorResponse 处理错误响应
func (s *taskService) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	return ParseHTTPError(resp.StatusCode, body)
}

// Tasks 返回任务服务实例
func (c *Client) Tasks() TaskService {
	return newTaskService(c)
}