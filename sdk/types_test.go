package sdk

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskStatus_String(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusRunning, "running"},
		{TaskStatusSucceeded, "succeeded"},
		{TaskStatusFailed, "failed"},
		{TaskStatusCancelled, "cancelled"},
		{TaskStatusExpired, "expired"},
		{TaskStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("TaskStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateTaskRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateTaskRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &CreateTaskRequest{
				BusinessUniqueID: "test-123",
				CallbackURL:      "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name: "missing business unique id",
			req: &CreateTaskRequest{
				BusinessUniqueID: "",
				CallbackURL:      "https://example.com/webhook",
			},
			wantErr: true,
		},
		{
			name: "missing callback url",
			req: &CreateTaskRequest{
				BusinessUniqueID: "test-123",
				CallbackURL:      "",
			},
			wantErr: true,
		},
		{
			name: "valid with defaults applied",
			req: &CreateTaskRequest{
				BusinessUniqueID: "test-123",
				CallbackURL:      "https://example.com/webhook",
				// 不设置 CallbackMethod, Priority, Timeout, MaxRetries
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalReq := *tt.req // 保存原始值用于比较

			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTaskRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 检查默认值是否已设置
				if tt.req.CallbackMethod == "" {
					t.Error("Expected CallbackMethod to be set to default value")
				}
				if originalReq.CallbackMethod == "" && tt.req.CallbackMethod != "POST" {
					t.Errorf("Expected default CallbackMethod 'POST', got %s", tt.req.CallbackMethod)
				}

				if originalReq.Priority == 0 && tt.req.Priority != TaskPriorityNormal {
					t.Errorf("Expected default Priority %d, got %d", TaskPriorityNormal, tt.req.Priority)
				}

				if originalReq.Timeout == 0 && tt.req.Timeout != 300 {
					t.Errorf("Expected default Timeout 300, got %d", tt.req.Timeout)
				}

				if originalReq.MaxRetries == 0 && tt.req.MaxRetries != 3 {
					t.Errorf("Expected default MaxRetries 3, got %d", tt.req.MaxRetries)
				}
			}
		})
	}
}

func TestTask_MarshalJSON(t *testing.T) {
	now := time.Now()
	later := now.Add(1 * time.Hour)

	task := &Task{
		ID:               123,
		BusinessUniqueID: "test-task",
		CallbackURL:      "https://example.com/webhook",
		Status:           TaskStatusPending,
		Priority:         TaskPriorityHigh,
		Tags:             []string{"test", "example"},
		ScheduledAt:      now,
		NextExecuteAt:    &later,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	// 解析JSON以验证格式
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal task JSON: %v", err)
	}

	// 检查时间字段是否为RFC3339格式
	scheduledAt, ok := parsed["scheduled_at"].(string)
	if !ok {
		t.Error("scheduled_at should be a string")
	} else {
		_, err := time.Parse(time.RFC3339, scheduledAt)
		if err != nil {
			t.Errorf("scheduled_at is not in RFC3339 format: %v", err)
		}
	}

	nextExecuteAt, ok := parsed["next_execute_at"].(string)
	if !ok {
		t.Error("next_execute_at should be a string")
	} else {
		_, err := time.Parse(time.RFC3339, nextExecuteAt)
		if err != nil {
			t.Errorf("next_execute_at is not in RFC3339 format: %v", err)
		}
	}

	// 检查其他字段
	if id, ok := parsed["id"].(float64); !ok || int64(id) != 123 {
		t.Errorf("Expected id 123, got %v", parsed["id"])
	}

	if businessID, ok := parsed["business_unique_id"].(string); !ok || businessID != "test-task" {
		t.Errorf("Expected business_unique_id 'test-task', got %v", parsed["business_unique_id"])
	}
}

func TestTaskHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		isActive bool
		isCompleted bool
		isSuccessful bool
	}{
		{"pending", TaskStatusPending, true, false, false},
		{"running", TaskStatusRunning, true, false, false},
		{"succeeded", TaskStatusSucceeded, false, true, true},
		{"failed", TaskStatusFailed, false, true, false},
		{"cancelled", TaskStatusCancelled, false, true, false},
		{"expired", TaskStatusExpired, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTaskActive(tt.status); got != tt.isActive {
				t.Errorf("IsTaskActive(%s) = %v, want %v", tt.name, got, tt.isActive)
			}

			if got := IsTaskCompleted(tt.status); got != tt.isCompleted {
				t.Errorf("IsTaskCompleted(%s) = %v, want %v", tt.name, got, tt.isCompleted)
			}

			if got := IsTaskSuccessful(tt.status); got != tt.isSuccessful {
				t.Errorf("IsTaskSuccessful(%s) = %v, want %v", tt.name, got, tt.isSuccessful)
			}
		})
	}
}

func TestCalculateRetryDelay(t *testing.T) {
	intervals := []int{10, 30, 60, 120}

	tests := []struct {
		name         string
		currentRetry int
		intervals    []int
		want         time.Duration
	}{
		{"first retry", 0, intervals, 10 * time.Second},
		{"second retry", 1, intervals, 30 * time.Second},
		{"third retry", 2, intervals, 60 * time.Second},
		{"fourth retry", 3, intervals, 120 * time.Second},
		{"beyond intervals", 5, intervals, 120 * time.Second}, // 应该使用最后一个间隔
		{"empty intervals", 0, []int{}, 0 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.intervals) == 0 {
				// 测试空间隔数组的情况
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for empty intervals")
					}
				}()
			}

			got := CalculateRetryDelay(tt.currentRetry, tt.intervals)
			if got != tt.want {
				t.Errorf("CalculateRetryDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredefinedRetryIntervals(t *testing.T) {
	tests := []struct {
		name      string
		intervals []int
		expected  []int
	}{
		{"StandardRetryIntervals", StandardRetryIntervals, []int{60, 300, 900}},
		{"FastRetryIntervals", FastRetryIntervals, []int{10, 30, 60}},
		{"SlowRetryIntervals", SlowRetryIntervals, []int{300, 1800, 7200}},
		{"ExponentialRetryIntervals", ExponentialRetryIntervals, []int{60, 120, 240, 480, 960}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.intervals) != len(tt.expected) {
				t.Errorf("Expected %d intervals, got %d", len(tt.expected), len(tt.intervals))
				return
			}

			for i, expected := range tt.expected {
				if tt.intervals[i] != expected {
					t.Errorf("Interval[%d]: expected %d, got %d", i, expected, tt.intervals[i])
				}
			}
		})
	}
}

func TestNewTaskConstructors(t *testing.T) {
	t.Run("NewTask", func(t *testing.T) {
		task := NewTask("test-id", "https://example.com")

		if task.BusinessUniqueID != "test-id" {
			t.Errorf("Expected BusinessUniqueID 'test-id', got %s", task.BusinessUniqueID)
		}

		if task.CallbackURL != "https://example.com" {
			t.Errorf("Expected CallbackURL 'https://example.com', got %s", task.CallbackURL)
		}

		if task.CallbackMethod != "POST" {
			t.Errorf("Expected CallbackMethod 'POST', got %s", task.CallbackMethod)
		}

		if task.Priority != TaskPriorityNormal {
			t.Errorf("Expected Priority %d, got %d", TaskPriorityNormal, task.Priority)
		}
	})

	t.Run("NewTaskWithCallback", func(t *testing.T) {
		headers := map[string]string{"Authorization": "Bearer token"}
		body := `{"test": true}`

		task := NewTaskWithCallback("test-id", "https://example.com", "PUT", headers, body)

		if task.CallbackMethod != "PUT" {
			t.Errorf("Expected CallbackMethod 'PUT', got %s", task.CallbackMethod)
		}

		if task.CallbackHeaders["Authorization"] != "Bearer token" {
			t.Errorf("Expected Authorization header, got %v", task.CallbackHeaders)
		}

		if task.CallbackBody != body {
			t.Errorf("Expected CallbackBody %s, got %s", body, task.CallbackBody)
		}
	})

	t.Run("NewScheduledTask", func(t *testing.T) {
		scheduledTime := time.Now().Add(1 * time.Hour)
		task := NewScheduledTask("test-id", "https://example.com", scheduledTime)

		if task.ScheduledAt == nil {
			t.Error("Expected ScheduledAt to be set")
		} else if !task.ScheduledAt.Equal(scheduledTime) {
			t.Errorf("Expected ScheduledAt %v, got %v", scheduledTime, *task.ScheduledAt)
		}
	})
}

func TestCreateTaskRequest_ChainedMethods(t *testing.T) {
	scheduledTime := time.Now().Add(1 * time.Hour)
	headers := map[string]string{"Content-Type": "application/json"}
	metadata := map[string]interface{}{"key": "value"}

	task := NewTask("test-id", "https://example.com").
		WithPriority(TaskPriorityHigh).
		WithTimeout(60).
		WithRetries(5, []int{10, 20, 30}).
		WithTags("tag1", "tag2").
		WithMetadata(metadata).
		WithHeaders(headers).
		WithBody(`{"test": true}`).
		WithSchedule(scheduledTime)

	if task.Priority != TaskPriorityHigh {
		t.Errorf("Expected Priority %d, got %d", TaskPriorityHigh, task.Priority)
	}

	if task.Timeout != 60 {
		t.Errorf("Expected Timeout 60, got %d", task.Timeout)
	}

	if task.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", task.MaxRetries)
	}

	if len(task.RetryIntervals) != 3 {
		t.Errorf("Expected 3 retry intervals, got %d", len(task.RetryIntervals))
	}

	if len(task.Tags) != 2 || task.Tags[0] != "tag1" || task.Tags[1] != "tag2" {
		t.Errorf("Expected tags [tag1, tag2], got %v", task.Tags)
	}

	if task.Metadata["key"] != "value" {
		t.Errorf("Expected metadata key=value, got %v", task.Metadata)
	}

	if task.CallbackHeaders["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header, got %v", task.CallbackHeaders)
	}

	if task.CallbackBody != `{"test": true}` {
		t.Errorf("Expected body, got %s", task.CallbackBody)
	}

	if task.ScheduledAt == nil || !task.ScheduledAt.Equal(scheduledTime) {
		t.Errorf("Expected ScheduledAt %v, got %v", scheduledTime, task.ScheduledAt)
	}
}