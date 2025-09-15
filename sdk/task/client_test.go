package task

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"task-center/sdk"
)

// mockServer 创建模拟HTTP服务器
func mockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// createTestClient 创建测试客户端
func createTestClient(t *testing.T, server *httptest.Server) *Client {
	config := &sdk.Config{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		BusinessID: 123,
		Timeout:    5 * time.Second,
	}

	client, err := NewClientWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

func TestClient_CreateTask(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tasks" {
			t.Errorf("Expected path /api/v1/tasks, got %s", r.URL.Path)
		}

		// 验证请求头
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header with Bearer token")
		}

		if r.Header.Get("X-Business-ID") != "123" {
			t.Errorf("Expected X-Business-ID header with value 123")
		}

		// 验证请求体
		var req sdk.CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if req.BusinessUniqueID != "test-business-id" {
			t.Errorf("Expected business ID test-business-id, got %s", req.BusinessUniqueID)
		}

		// 返回成功响应
		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:               1,
				BusinessUniqueID: req.BusinessUniqueID,
				CallbackURL:      req.CallbackURL,
				Status:           sdk.TaskStatusPending,
				Priority:         req.Priority,
				CreatedAt:        time.Now(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	req := NewCreateRequest("test-business-id", "https://example.com/callback").
		WithPriority(PriorityHigh).
		WithTags("test")

	ctx := context.Background()
	task, err := client.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if task.ID != 1 {
		t.Errorf("Expected task ID 1, got %d", task.ID)
	}

	if task.BusinessUniqueID != "test-business-id" {
		t.Errorf("Expected business ID test-business-id, got %s", task.BusinessUniqueID)
	}

	if task.Status != StatusPending {
		t.Errorf("Expected pending status, got %d", task.Status)
	}
}

func TestClient_GetTask(t *testing.T) {
	taskID := int64(123)

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/api/v1/tasks/123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:               taskID,
				BusinessUniqueID: "test-business-id",
				CallbackURL:      "https://example.com/callback",
				Status:           sdk.TaskStatusRunning,
				Priority:         sdk.TaskPriorityNormal,
				CreatedAt:        time.Now(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	task, err := client.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if task.ID != taskID {
		t.Errorf("Expected task ID %d, got %d", taskID, task.ID)
	}

	if task.Status != StatusRunning {
		t.Errorf("Expected running status, got %d", task.Status)
	}
}

func TestClient_GetTaskByBusinessID(t *testing.T) {
	businessID := "test-business-id"

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		expectedPath := "/api/v1/tasks/business/test-business-id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:               456,
				BusinessUniqueID: businessID,
				CallbackURL:      "https://example.com/callback",
				Status:           sdk.TaskStatusSucceeded,
				CreatedAt:        time.Now(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	task, err := client.GetTaskByBusinessID(ctx, businessID)
	if err != nil {
		t.Fatalf("GetTaskByBusinessID() error = %v", err)
	}

	if task.ID != 456 {
		t.Errorf("Expected task ID 456, got %d", task.ID)
	}

	if task.BusinessUniqueID != businessID {
		t.Errorf("Expected business ID %s, got %s", businessID, task.BusinessUniqueID)
	}
}

func TestClient_UpdateTask(t *testing.T) {
	taskID := int64(789)

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		expectedPath := "/api/v1/tasks/789"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		var req sdk.UpdateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if *req.CallbackURL != "https://new-callback.com" {
			t.Errorf("Expected new callback URL, got %s", *req.CallbackURL)
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:          taskID,
				CallbackURL: *req.CallbackURL,
				Status:      *req.Status,
				UpdatedAt:   time.Now(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	updateReq := NewUpdateRequest().
		WithCallbackURL("https://new-callback.com").
		WithStatus(StatusCancelled)

	ctx := context.Background()
	task, err := client.UpdateTask(ctx, taskID, updateReq)
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	if task.ID != taskID {
		t.Errorf("Expected task ID %d, got %d", taskID, task.ID)
	}

	if task.CallbackURL != "https://new-callback.com" {
		t.Errorf("Expected updated callback URL, got %s", task.CallbackURL)
	}
}

func TestClient_DeleteTask(t *testing.T) {
	taskID := int64(999)

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		expectedPath := "/api/v1/tasks/999"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	err := client.DeleteTask(ctx, taskID)
	if err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
}

func TestClient_CancelTask(t *testing.T) {
	taskID := int64(111)

	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/api/v1/tasks/111/cancel"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.Task{
				ID:     taskID,
				Status: sdk.TaskStatusCancelled,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	task, err := client.CancelTask(ctx, taskID)
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}

	if task.Status != StatusCancelled {
		t.Errorf("Expected cancelled status, got %d", task.Status)
	}
}

func TestClient_ListTasks(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if !strings.HasPrefix(r.URL.Path, "/api/v1/tasks") {
			t.Errorf("Expected path to start with /api/v1/tasks, got %s", r.URL.Path)
		}

		// 验证查询参数
		query := r.URL.Query()
		if query.Get("status") != "0,1" {
			t.Errorf("Expected status query parameter 0,1, got %s", query.Get("status"))
		}

		if query.Get("page") != "1" {
			t.Errorf("Expected page query parameter 1, got %s", query.Get("page"))
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.ListTasksResponse{
				Tasks: []sdk.Task{
					{
						ID:               1,
						BusinessUniqueID: "task1",
						Status:           sdk.TaskStatusPending,
					},
					{
						ID:               2,
						BusinessUniqueID: "task2",
						Status:           sdk.TaskStatusRunning,
					},
				},
				Total:      2,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	req := NewListRequest().
		WithStatus(StatusPending, StatusRunning).
		WithPagination(1, 20)

	ctx := context.Background()
	resp, err := client.ListTasks(ctx, req)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}

	if len(resp.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(resp.Tasks))
	}

	if resp.Total != 2 {
		t.Errorf("Expected total 2, got %d", resp.Total)
	}

	if resp.Tasks[0].Status != StatusPending {
		t.Errorf("Expected first task pending status, got %d", resp.Tasks[0].Status)
	}
}

func TestClient_GetTaskStats(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tasks/stats" {
			t.Errorf("Expected path /api/v1/tasks/stats, got %s", r.URL.Path)
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.TaskStatsResponse{
				TotalTasks: 100,
				StatusCounts: map[sdk.TaskStatus]int{
					sdk.TaskStatusPending:   10,
					sdk.TaskStatusRunning:   5,
					sdk.TaskStatusSucceeded: 80,
					sdk.TaskStatusFailed:    5,
				},
				PriorityCounts: map[sdk.TaskPriority]int{
					sdk.TaskPriorityHigh:   20,
					sdk.TaskPriorityNormal: 70,
					sdk.TaskPriorityLow:    10,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	stats, err := client.GetTaskStats(ctx)
	if err != nil {
		t.Fatalf("GetTaskStats() error = %v", err)
	}

	if stats.TotalTasks != 100 {
		t.Errorf("Expected total tasks 100, got %d", stats.TotalTasks)
	}

	if stats.StatusCounts[StatusPending] != 10 {
		t.Errorf("Expected 10 pending tasks, got %d", stats.StatusCounts[StatusPending])
	}

	if stats.PriorityCounts[PriorityHigh] != 20 {
		t.Errorf("Expected 20 high priority tasks, got %d", stats.PriorityCounts[PriorityHigh])
	}
}

func TestClient_ValidationErrors(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// This handler should not be called for validation errors
		t.Error("Server handler should not be called for validation errors")
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()

	// Test invalid task ID
	_, err := client.GetTask(ctx, 0)
	if err == nil {
		t.Error("Expected validation error for invalid task ID")
	}

	// Test invalid business ID
	_, err = client.GetTaskByBusinessID(ctx, "")
	if err == nil {
		t.Error("Expected validation error for empty business ID")
	}

	// Test nil create request
	_, err = client.CreateTask(ctx, nil)
	if err == nil {
		t.Error("Expected validation error for nil create request")
	}

	// Test nil update request
	_, err = client.UpdateTask(ctx, 1, nil)
	if err == nil {
		t.Error("Expected validation error for nil update request")
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tasks/404":
			w.WriteHeader(http.StatusNotFound)
			errorResp := sdk.ErrorResponse{
				Success: false,
				Message: "Task not found",
				Code:    "NOT_FOUND_ERROR",
			}
			json.NewEncoder(w).Encode(errorResp)

		case "/api/v1/tasks/500":
			w.WriteHeader(http.StatusInternalServerError)
			errorResp := sdk.ErrorResponse{
				Success: false,
				Message: "Internal server error",
				Code:    "SERVER_ERROR",
			}
			json.NewEncoder(w).Encode(errorResp)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()

	// Test 404 error
	_, err := client.GetTask(ctx, 404)
	if err == nil {
		t.Error("Expected error for 404 response")
	}
	if !sdk.IsNotFoundError(err) {
		t.Errorf("Expected NotFoundError, got %T", err)
	}

	// Test 500 error
	_, err = client.GetTask(ctx, 500)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
	if !sdk.IsServerError(err) {
		t.Errorf("Expected ServerError, got %T", err)
	}
}

func TestClient_CheckTaskExists(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("Expected HEAD method, got %s", r.Method)
		}

		switch r.URL.Path {
		case "/api/v1/tasks/123/exists":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/tasks/404/exists":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()

	// Test existing task
	exists, err := client.CheckTaskExists(ctx, 123)
	if err != nil {
		t.Fatalf("CheckTaskExists() error = %v", err)
	}
	if !exists {
		t.Error("Expected task to exist")
	}

	// Test non-existing task
	exists, err = client.CheckTaskExists(ctx, 404)
	if err != nil {
		t.Fatalf("CheckTaskExists() error = %v", err)
	}
	if exists {
		t.Error("Expected task to not exist")
	}
}

func TestClient_SearchTasks(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		if !strings.HasPrefix(r.URL.Path, "/api/v1/tasks/search") {
			t.Errorf("Expected path to start with /api/v1/tasks/search, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("q") != "test query" {
			t.Errorf("Expected query parameter 'test query', got %s", query.Get("q"))
		}

		response := sdk.ApiResponse{
			Success: true,
			Data: sdk.ListTasksResponse{
				Tasks: []sdk.Task{
					{
						ID:               1,
						BusinessUniqueID: "search-result-1",
						Status:           sdk.TaskStatusPending,
					},
				},
				Total:      1,
				Page:       1,
				PageSize:   20,
				TotalPages: 1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createTestClient(t, server)
	defer client.Close()

	ctx := context.Background()
	resp, err := client.SearchTasks(ctx, "test query", nil)
	if err != nil {
		t.Fatalf("SearchTasks() error = %v", err)
	}

	if len(resp.Tasks) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(resp.Tasks))
	}

	if resp.Tasks[0].BusinessUniqueID != "search-result-1" {
		t.Errorf("Expected business ID search-result-1, got %s", resp.Tasks[0].BusinessUniqueID)
	}
}