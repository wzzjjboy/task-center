package task

import (
	"encoding/json"
	"testing"
	"time"

	"task-center/sdk"
)

func TestTaskBuilder(t *testing.T) {
	builder := NewTaskBuilder("test-business-id", "https://example.com/callback")

	req := builder.
		Method("PUT").
		Headers(map[string]string{"Authorization": "Bearer token"}).
		Body(`{"test": "data"}`).
		Retry(5, 1, 2, 4).
		Priority(PriorityHigh).
		Tags("urgent", "test").
		Timeout(600).
		ScheduledAfter(time.Hour).
		MetadataValue("environment", "test").
		Build()

	if req.CallbackMethod != "PUT" {
		t.Errorf("Expected method PUT, got %s", req.CallbackMethod)
	}

	if req.CallbackHeaders["Authorization"] != "Bearer token" {
		t.Errorf("Expected authorization header, got %v", req.CallbackHeaders)
	}

	if req.CallbackBody != `{"test": "data"}` {
		t.Errorf("Expected body content, got %s", req.CallbackBody)
	}

	if req.MaxRetries != 5 {
		t.Errorf("Expected 5 max retries, got %d", req.MaxRetries)
	}

	if len(req.RetryIntervals) != 3 {
		t.Errorf("Expected 3 retry intervals, got %d", len(req.RetryIntervals))
	}

	if req.Priority != PriorityHigh {
		t.Errorf("Expected high priority, got %d", req.Priority)
	}

	if len(req.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(req.Tags))
	}

	if req.Timeout != 600 {
		t.Errorf("Expected timeout 600, got %d", req.Timeout)
	}

	if req.ScheduledAt == nil {
		t.Error("Expected scheduled time to be set")
	}

	if req.Metadata["environment"] != "test" {
		t.Errorf("Expected metadata environment=test, got %v", req.Metadata["environment"])
	}
}

func TestFilterBuilder(t *testing.T) {
	filter := NewFilterBuilder()

	req := filter.
		Status(StatusPending, StatusRunning).
		Tags("urgent", "test").
		Priority(PriorityHigh).
		CreatedToday().
		Pagination(2, 50).
		Build()

	if len(req.Status) != 2 {
		t.Errorf("Expected 2 status filters, got %d", len(req.Status))
	}

	if req.Status[0] != StatusPending || req.Status[1] != StatusRunning {
		t.Errorf("Expected pending and running status, got %v", req.Status)
	}

	if len(req.Tags) != 2 {
		t.Errorf("Expected 2 tag filters, got %d", len(req.Tags))
	}

	if *req.Priority != PriorityHigh {
		t.Errorf("Expected high priority filter, got %d", *req.Priority)
	}

	if req.Page != 2 {
		t.Errorf("Expected page 2, got %d", req.Page)
	}

	if req.PageSize != 50 {
		t.Errorf("Expected page size 50, got %d", req.PageSize)
	}

	if req.CreatedFrom == nil {
		t.Error("Expected created from time to be set")
	}

	if req.CreatedTo == nil {
		t.Error("Expected created to time to be set")
	}
}

func TestCreateRequestValidation(t *testing.T) {
	tests := []struct {
		name      string
		req       *CreateRequest
		wantError bool
	}{
		{
			name: "valid request",
			req: NewCreateRequest("test-id", "https://example.com/callback"),
			wantError: false,
		},
		{
			name: "missing business unique ID",
			req: &CreateRequest{
				CreateTaskRequest: &sdk.CreateTaskRequest{
					BusinessUniqueID: "",
					CallbackURL:      "https://example.com/callback",
				},
			},
			wantError: true,
		},
		{
			name: "missing callback URL",
			req: &CreateRequest{
				CreateTaskRequest: &sdk.CreateTaskRequest{
					BusinessUniqueID: "test-id",
					CallbackURL:      "",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTaskMethods(t *testing.T) {
	now := time.Now()
	executedAt := now.Add(-time.Hour)
	completedAt := now.Add(-30 * time.Minute)

	task := &Task{
		Task: &sdk.Task{
			ID:               1,
			BusinessUniqueID: "test-id",
			Status:           StatusSucceeded,
			CurrentRetry:     2,
			MaxRetries:       3,
			ExecutedAt:       &executedAt,
			CompletedAt:      &completedAt,
			CreatedAt:        now.Add(-2 * time.Hour),
		},
	}

	// Test status checks
	if !task.IsCompleted() {
		t.Error("Task should be completed")
	}

	if !task.IsSucceeded() {
		t.Error("Task should be succeeded")
	}

	if task.IsRunning() {
		t.Error("Task should not be running")
	}

	if task.IsPending() {
		t.Error("Task should not be pending")
	}

	// Test duration calculation
	duration := task.GetDuration()
	expected := completedAt.Sub(executedAt)
	if duration != expected {
		t.Errorf("Expected duration %v, got %v", expected, duration)
	}

	// Test wait time calculation
	waitTime := task.GetWaitTime()
	expectedWait := executedAt.Sub(task.CreatedAt)
	if waitTime != expectedWait {
		t.Errorf("Expected wait time %v, got %v", expectedWait, waitTime)
	}

	// Test retry checks
	if !task.HasRetry() {
		t.Error("Task should have retries")
	}

	if task.CanRetry() {
		t.Error("Task should not be able to retry (succeeded)")
	}
}

func TestTaskJSONSerialization(t *testing.T) {
	task := &Task{
		Task: &sdk.Task{
			ID:               1,
			BusinessUniqueID: "test-id",
			CallbackURL:      "https://example.com/callback",
			Status:           StatusPending,
			Priority:         PriorityNormal,
			Tags:             []string{"test"},
			CreatedAt:        time.Now(),
		},
	}

	// Test ToJSON
	jsonStr, err := task.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Test FromJSON
	parsedTask, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if parsedTask.ID != task.ID {
		t.Errorf("Expected ID %d, got %d", task.ID, parsedTask.ID)
	}

	if parsedTask.BusinessUniqueID != task.BusinessUniqueID {
		t.Errorf("Expected business ID %s, got %s", task.BusinessUniqueID, parsedTask.BusinessUniqueID)
	}
}

func TestBatchCreateRequest(t *testing.T) {
	batch := NewBatchCreateRequest()

	req1 := NewCreateRequest("id1", "https://example.com/callback1")
	req2 := NewCreateRequest("id2", "https://example.com/callback2")

	batch.AddTask(req1).AddTask(req2)

	if len(batch.Requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(batch.Requests))
	}

	// Test ToSDK conversion
	sdkReq := batch.ToSDK()
	if len(sdkReq.Tasks) != 2 {
		t.Errorf("Expected 2 SDK tasks, got %d", len(sdkReq.Tasks))
	}

	if sdkReq.Tasks[0].BusinessUniqueID != "id1" {
		t.Errorf("Expected first task ID id1, got %s", sdkReq.Tasks[0].BusinessUniqueID)
	}
}

func TestUpdateRequestChaining(t *testing.T) {
	timeout := 300
	priority := PriorityLow
	status := StatusCancelled
	scheduledAt := time.Now().Add(time.Hour)

	req := NewUpdateRequest().
		WithCallbackURL("https://new-callback.com").
		WithCallbackMethod("PUT").
		WithTimeout(timeout).
		WithPriority(priority).
		WithStatus(status).
		WithScheduledAt(scheduledAt).
		WithTags("updated", "test").
		WithMetadata(map[string]interface{}{"updated": true})

	if *req.CallbackURL != "https://new-callback.com" {
		t.Errorf("Expected new callback URL, got %s", *req.CallbackURL)
	}

	if *req.CallbackMethod != "PUT" {
		t.Errorf("Expected PUT method, got %s", *req.CallbackMethod)
	}

	if *req.Timeout != timeout {
		t.Errorf("Expected timeout %d, got %d", timeout, *req.Timeout)
	}

	if *req.Priority != priority {
		t.Errorf("Expected priority %d, got %d", priority, *req.Priority)
	}

	if *req.Status != status {
		t.Errorf("Expected status %d, got %d", status, *req.Status)
	}

	if req.ScheduledAt.Unix() != scheduledAt.Unix() {
		t.Errorf("Expected scheduled time %v, got %v", scheduledAt, *req.ScheduledAt)
	}

	if len(req.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(req.Tags))
	}

	if req.Metadata["updated"] != true {
		t.Errorf("Expected metadata updated=true, got %v", req.Metadata["updated"])
	}
}

func TestNewTaskFromSDK(t *testing.T) {
	sdkTask := &sdk.Task{
		ID:               123,
		BusinessUniqueID: "test-business-id",
		CallbackURL:      "https://example.com/callback",
		Status:           sdk.TaskStatusRunning,
		Priority:         sdk.TaskPriorityHigh,
		Tags:             []string{"test", "example"},
		CreatedAt:        time.Now(),
	}

	task := NewTaskFromSDK(sdkTask)

	if task.ID != 123 {
		t.Errorf("Expected ID 123, got %d", task.ID)
	}

	if task.BusinessUniqueID != "test-business-id" {
		t.Errorf("Expected business ID test-business-id, got %s", task.BusinessUniqueID)
	}

	if task.Status != StatusRunning {
		t.Errorf("Expected running status, got %d", task.Status)
	}

	if task.Priority != PriorityHigh {
		t.Errorf("Expected high priority, got %d", task.Priority)
	}

	// Test ToSDK conversion
	convertedSDK := task.ToSDK()
	if convertedSDK.ID != sdkTask.ID {
		t.Errorf("Expected converted ID %d, got %d", sdkTask.ID, convertedSDK.ID)
	}
}

func TestListResponseFromSDK(t *testing.T) {
	sdkTasks := []sdk.Task{
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
	}

	sdkResp := &sdk.ListTasksResponse{
		Tasks:      sdkTasks,
		Total:      100,
		Page:       1,
		PageSize:   20,
		TotalPages: 5,
	}

	resp := NewListResponseFromSDK(sdkResp)

	if len(resp.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(resp.Tasks))
	}

	if resp.Total != 100 {
		t.Errorf("Expected total 100, got %d", resp.Total)
	}

	if resp.Tasks[0].ID != 1 {
		t.Errorf("Expected first task ID 1, got %d", resp.Tasks[0].ID)
	}

	if resp.Tasks[1].Status != StatusRunning {
		t.Errorf("Expected second task running status, got %d", resp.Tasks[1].Status)
	}
}

func TestBatchCreateResponseFromSDK(t *testing.T) {
	succeededTasks := []sdk.Task{
		{ID: 1, BusinessUniqueID: "task1"},
		{ID: 2, BusinessUniqueID: "task2"},
	}

	failedTasks := []sdk.BatchTaskError{
		{
			Index: 2,
			Error: "validation failed",
			Code:  "VALIDATION_ERROR",
		},
	}

	sdkResp := &sdk.BatchCreateTasksResponse{
		Succeeded: succeededTasks,
		Failed:    failedTasks,
	}

	resp := NewBatchCreateResponseFromSDK(sdkResp)

	if len(resp.Succeeded) != 2 {
		t.Errorf("Expected 2 succeeded tasks, got %d", len(resp.Succeeded))
	}

	if len(resp.Failed) != 1 {
		t.Errorf("Expected 1 failed task, got %d", len(resp.Failed))
	}

	if resp.Succeeded[0].ID != 1 {
		t.Errorf("Expected first succeeded task ID 1, got %d", resp.Succeeded[0].ID)
	}

	if resp.Failed[0].Index != 2 {
		t.Errorf("Expected failed task index 2, got %d", resp.Failed[0].Index)
	}
}