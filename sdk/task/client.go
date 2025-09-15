package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"task-center/sdk"
)

// Client 任务管理客户端
type Client struct {
	sdkClient *sdk.Client
}

// NewClient 创建任务管理客户端
func NewClient(sdkClient *sdk.Client) *Client {
	return &Client{
		sdkClient: sdkClient,
	}
}

// NewClientWithConfig 使用配置创建任务管理客户端
func NewClientWithConfig(config *sdk.Config) (*Client, error) {
	sdkClient, err := sdk.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}
	return NewClient(sdkClient), nil
}

// CreateTask 创建单个任务
func (c *Client) CreateTask(ctx context.Context, req *CreateRequest) (*Task, error) {
	if req == nil {
		return nil, sdk.NewValidationError("create request cannot be nil")
	}

	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 发送HTTP请求
	resp, err := c.sdkClient.DoRequest(ctx, "POST", "/api/v1/tasks", req.CreateTaskRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	return c.parseTaskResponse(resp)
}

// GetTask 根据ID获取任务
func (c *Client) GetTask(ctx context.Context, taskID int64) (*Task, error) {
	if taskID <= 0 {
		return nil, sdk.NewValidationError("task ID must be greater than 0")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseTaskResponse(resp)
}

// GetTaskByBusinessID 根据业务唯一ID获取任务
func (c *Client) GetTaskByBusinessID(ctx context.Context, businessUniqueID string) (*Task, error) {
	if businessUniqueID == "" {
		return nil, sdk.NewValidationError("business unique ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/tasks/business/%s", url.PathEscape(businessUniqueID))
	resp, err := c.sdkClient.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseTaskResponse(resp)
}

// UpdateTask 更新任务
func (c *Client) UpdateTask(ctx context.Context, taskID int64, req *UpdateRequest) (*Task, error) {
	if taskID <= 0 {
		return nil, sdk.NewValidationError("task ID must be greater than 0")
	}
	if req == nil {
		return nil, sdk.NewValidationError("update request cannot be nil")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "PUT", path, req.UpdateTaskRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseTaskResponse(resp)
}

// DeleteTask 删除任务
func (c *Client) DeleteTask(ctx context.Context, taskID int64) error {
	if taskID <= 0 {
		return sdk.NewValidationError("task ID must be greater than 0")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseErrorResponse(resp)
	}

	return nil
}

// CancelTask 取消任务
func (c *Client) CancelTask(ctx context.Context, taskID int64) (*Task, error) {
	if taskID <= 0 {
		return nil, sdk.NewValidationError("task ID must be greater than 0")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d/cancel", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseTaskResponse(resp)
}

// RetryTask 重试任务
func (c *Client) RetryTask(ctx context.Context, taskID int64) (*Task, error) {
	if taskID <= 0 {
		return nil, sdk.NewValidationError("task ID must be greater than 0")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d/retry", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseTaskResponse(resp)
}

// ListTasks 查询任务列表
func (c *Client) ListTasks(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	if req == nil {
		req = NewListRequest()
	}

	// 构建查询参数
	query := c.buildListQuery(req)
	path := "/api/v1/tasks"
	if query != "" {
		path += "?" + query
	}

	resp, err := c.sdkClient.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseListResponse(resp)
}

// GetTaskStats 获取任务统计信息
func (c *Client) GetTaskStats(ctx context.Context) (*StatsResponse, error) {
	resp, err := c.sdkClient.DoRequest(ctx, "GET", "/api/v1/tasks/stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseStatsResponse(resp)
}

// SearchTasks 搜索任务
func (c *Client) SearchTasks(ctx context.Context, query string, filters *ListRequest) (*ListResponse, error) {
	if query == "" {
		return nil, sdk.NewValidationError("search query cannot be empty")
	}

	if filters == nil {
		filters = NewListRequest()
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("q", query)

	// 添加过滤条件
	filterQuery := c.buildListQuery(filters)
	if filterQuery != "" {
		// 解析已有的查询参数并合并
		existingParams, _ := url.ParseQuery(filterQuery)
		for key, values := range existingParams {
			for _, value := range values {
				params.Add(key, value)
			}
		}
	}

	path := "/api/v1/tasks/search?" + params.Encode()
	resp, err := c.sdkClient.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseListResponse(resp)
}

// GetTaskHistory 获取任务执行历史
func (c *Client) GetTaskHistory(ctx context.Context, taskID int64) ([]*Task, error) {
	if taskID <= 0 {
		return nil, sdk.NewValidationError("task ID must be greater than 0")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d/history", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseTaskListResponse(resp)
}

// GetTasksByStatus 根据状态获取任务列表
func (c *Client) GetTasksByStatus(ctx context.Context, status TaskStatus, page, pageSize int) (*ListResponse, error) {
	req := NewListRequest().
		WithStatus(status).
		WithPagination(page, pageSize)

	return c.ListTasks(ctx, req)
}

// GetTasksByTag 根据标签获取任务列表
func (c *Client) GetTasksByTag(ctx context.Context, tag string, page, pageSize int) (*ListResponse, error) {
	req := NewListRequest().
		WithTags(tag).
		WithPagination(page, pageSize)

	return c.ListTasks(ctx, req)
}

// GetPendingTasks 获取待执行任务
func (c *Client) GetPendingTasks(ctx context.Context, page, pageSize int) (*ListResponse, error) {
	return c.GetTasksByStatus(ctx, StatusPending, page, pageSize)
}

// GetRunningTasks 获取正在执行的任务
func (c *Client) GetRunningTasks(ctx context.Context, page, pageSize int) (*ListResponse, error) {
	return c.GetTasksByStatus(ctx, StatusRunning, page, pageSize)
}

// GetCompletedTasks 获取已完成的任务
func (c *Client) GetCompletedTasks(ctx context.Context, page, pageSize int) (*ListResponse, error) {
	req := NewListRequest().
		WithStatus(StatusSucceeded, StatusFailed, StatusCancelled, StatusExpired).
		WithPagination(page, pageSize)

	return c.ListTasks(ctx, req)
}

// GetFailedTasks 获取失败的任务
func (c *Client) GetFailedTasks(ctx context.Context, page, pageSize int) (*ListResponse, error) {
	return c.GetTasksByStatus(ctx, StatusFailed, page, pageSize)
}

// CheckTaskExists 检查任务是否存在
func (c *Client) CheckTaskExists(ctx context.Context, taskID int64) (bool, error) {
	if taskID <= 0 {
		return false, sdk.NewValidationError("task ID must be greater than 0")
	}

	path := fmt.Sprintf("/api/v1/tasks/%d/exists", taskID)
	resp, err := c.sdkClient.DoRequest(ctx, "HEAD", path, nil)
	if err != nil {
		if sdk.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// CheckTaskExistsByBusinessID 检查业务任务是否存在
func (c *Client) CheckTaskExistsByBusinessID(ctx context.Context, businessUniqueID string) (bool, error) {
	if businessUniqueID == "" {
		return false, sdk.NewValidationError("business unique ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/tasks/business/%s/exists", url.PathEscape(businessUniqueID))
	resp, err := c.sdkClient.DoRequest(ctx, "HEAD", path, nil)
	if err != nil {
		if sdk.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// 内部方法：构建列表查询参数
func (c *Client) buildListQuery(req *ListRequest) string {
	params := url.Values{}

	// 状态过滤
	if len(req.Status) > 0 {
		statuses := make([]string, len(req.Status))
		for i, status := range req.Status {
			statuses[i] = strconv.Itoa(int(status))
		}
		params.Set("status", strings.Join(statuses, ","))
	}

	// 标签过滤
	if len(req.Tags) > 0 {
		params.Set("tags", strings.Join(req.Tags, ","))
	}

	// 优先级过滤
	if req.Priority != nil {
		params.Set("priority", strconv.Itoa(int(*req.Priority)))
	}

	// 时间范围过滤
	if req.CreatedFrom != nil {
		params.Set("created_from", req.CreatedFrom.Format("2006-01-02T15:04:05Z"))
	}
	if req.CreatedTo != nil {
		params.Set("created_to", req.CreatedTo.Format("2006-01-02T15:04:05Z"))
	}

	// 分页参数
	if req.Page > 0 {
		params.Set("page", strconv.Itoa(req.Page))
	}
	if req.PageSize > 0 {
		params.Set("page_size", strconv.Itoa(req.PageSize))
	}

	return params.Encode()
}

// 内部方法：解析任务响应
func (c *Client) parseTaskResponse(resp *http.Response) (*Task, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp)
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

	// 解析任务数据
	taskData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var sdkTask sdk.Task
	if err := json.Unmarshal(taskData, &sdkTask); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return NewTaskFromSDK(&sdkTask), nil
}

// 内部方法：解析任务列表响应
func (c *Client) parseListResponse(resp *http.Response) (*ListResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp)
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

	// 解析列表数据
	listData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list data: %w", err)
	}

	var sdkResp sdk.ListTasksResponse
	if err := json.Unmarshal(listData, &sdkResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task list: %w", err)
	}

	return NewListResponseFromSDK(&sdkResp), nil
}

// 内部方法：解析任务列表响应（不包装在ApiResponse中）
func (c *Client) parseTaskListResponse(resp *http.Response) ([]*Task, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var sdkTasks []sdk.Task
	if err := json.Unmarshal(body, &sdkTasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task list: %w", err)
	}

	tasks := make([]*Task, len(sdkTasks))
	for i, sdkTask := range sdkTasks {
		tasks[i] = NewTaskFromSDK(&sdkTask)
	}

	return tasks, nil
}

// 内部方法：解析统计响应
func (c *Client) parseStatsResponse(resp *http.Response) (*StatsResponse, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp)
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

	// 解析统计数据
	statsData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stats data: %w", err)
	}

	var sdkStats sdk.TaskStatsResponse
	if err := json.Unmarshal(statsData, &sdkStats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return NewStatsResponseFromSDK(&sdkStats), nil
}

// 内部方法：解析错误响应
func (c *Client) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	return sdk.ParseHTTPError(resp.StatusCode, body)
}

// GetClient 获取底层 SDK 客户端（用于高级用法）
func (c *Client) GetClient() *sdk.Client {
	return c.sdkClient
}

// Close 关闭客户端
func (c *Client) Close() error {
	return c.sdkClient.Close()
}