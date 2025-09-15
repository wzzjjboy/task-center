// +build integration

package sdk

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// 这些是集成测试，需要真实的 TaskCenter 服务实例
// 运行命令: go test -tags=integration

func getTestConfig() *Config {
	config := DefaultConfig()
	config.BaseURL = getEnvOrDefault("TASKCENTER_API_URL", "http://localhost:8080")
	config.APIKey = getEnvOrDefault("TASKCENTER_API_KEY", "test-api-key")

	businessIDStr := getEnvOrDefault("TASKCENTER_BUSINESS_ID", "1")
	businessID, err := strconv.ParseInt(businessIDStr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Invalid TASKCENTER_BUSINESS_ID: %v", err))
	}
	config.BusinessID = businessID

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestIntegration_ClientCreation(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Error("Client should not be nil")
	}
}

func TestIntegration_TaskCreation(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 创建测试任务
	task := NewTask(
		"integration-test-"+time.Now().Format("20060102-150405"),
		"https://httpbin.org/post",
	).WithTags("integration", "test")

	ctx := context.Background()
	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if createdTask.ID == 0 {
		t.Error("Created task should have a non-zero ID")
	}

	if createdTask.BusinessUniqueID != task.BusinessUniqueID {
		t.Errorf("Business unique ID mismatch: expected %s, got %s",
			task.BusinessUniqueID, createdTask.BusinessUniqueID)
	}

	if createdTask.CallbackURL != task.CallbackURL {
		t.Errorf("Callback URL mismatch: expected %s, got %s",
			task.CallbackURL, createdTask.CallbackURL)
	}

	if createdTask.Status != TaskStatusPending {
		t.Errorf("Expected status %s, got %s",
			TaskStatusPending.String(), createdTask.Status.String())
	}
}

func TestIntegration_TaskRetrieval(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 首先创建一个任务
	task := NewTask(
		"integration-get-test-"+time.Now().Format("20060102-150405"),
		"https://httpbin.org/post",
	)

	ctx := context.Background()
	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// 通过ID获取任务
	retrievedTask, err := client.Tasks().Get(ctx, createdTask.ID)
	if err != nil {
		t.Fatalf("Failed to get task by ID: %v", err)
	}

	if retrievedTask.ID != createdTask.ID {
		t.Errorf("ID mismatch: expected %d, got %d",
			createdTask.ID, retrievedTask.ID)
	}

	// 通过业务唯一ID获取任务
	retrievedByBusinessID, err := client.Tasks().GetByBusinessUniqueID(ctx, task.BusinessUniqueID)
	if err != nil {
		t.Fatalf("Failed to get task by business unique ID: %v", err)
	}

	if retrievedByBusinessID.ID != createdTask.ID {
		t.Errorf("ID mismatch when getting by business unique ID: expected %d, got %d",
			createdTask.ID, retrievedByBusinessID.ID)
	}
}

func TestIntegration_TaskListing(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 创建几个测试任务
	testTag := "integration-list-test-" + time.Now().Format("150405")
	for i := 0; i < 3; i++ {
		task := NewTask(
			fmt.Sprintf("integration-list-test-%s-%d", time.Now().Format("20060102-150405"), i),
			"https://httpbin.org/post",
		).WithTags(testTag, "integration")

		_, err := client.Tasks().Create(ctx, task)
		if err != nil {
			t.Fatalf("Failed to create test task %d: %v", i, err)
		}
	}

	// 等待一小段时间确保任务已创建
	time.Sleep(1 * time.Second)

	// 列出带有特定标签的任务
	listReq := NewListTasksRequest().
		WithTagsFilter(testTag).
		WithPagination(1, 10)

	response, err := client.Tasks().List(ctx, listReq)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(response.Tasks) < 3 {
		t.Errorf("Expected at least 3 tasks, got %d", len(response.Tasks))
	}

	// 验证所有返回的任务都有期望的标签
	for _, task := range response.Tasks {
		hasTag := false
		for _, tag := range task.Tags {
			if tag == testTag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("Task %d does not have expected tag %s", task.ID, testTag)
		}
	}
}

func TestIntegration_TaskUpdate(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 创建测试任务
	task := NewTask(
		"integration-update-test-"+time.Now().Format("20060102-150405"),
		"https://httpbin.org/post",
	).WithPriority(TaskPriorityNormal)

	ctx := context.Background()
	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// 更新任务优先级
	updateReq := &UpdateTaskRequest{
		Priority: &TaskPriorityHigh,
	}

	updatedTask, err := client.Tasks().Update(ctx, createdTask.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	if updatedTask.Priority != TaskPriorityHigh {
		t.Errorf("Priority not updated: expected %d, got %d",
			TaskPriorityHigh, updatedTask.Priority)
	}
}

func TestIntegration_BatchTaskCreation(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 创建批量任务
	timestamp := time.Now().Format("20060102-150405")
	tasks := []CreateTaskRequest{}

	for i := 0; i < 5; i++ {
		task := *NewTask(
			fmt.Sprintf("integration-batch-test-%s-%d", timestamp, i),
			"https://httpbin.org/post",
		).WithTags("batch", "integration")

		tasks = append(tasks, task)
	}

	batchReq := &BatchCreateTasksRequest{
		Tasks: tasks,
	}

	ctx := context.Background()
	response, err := client.Tasks().BatchCreate(ctx, batchReq)
	if err != nil {
		t.Fatalf("Failed to batch create tasks: %v", err)
	}

	if len(response.Succeeded) != 5 {
		t.Errorf("Expected 5 successful tasks, got %d", len(response.Succeeded))
	}

	if len(response.Failed) != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", len(response.Failed))
	}

	// 验证每个任务都有正确的业务ID
	for i, task := range response.Succeeded {
		expectedBusinessID := fmt.Sprintf("integration-batch-test-%s-%d", timestamp, i)
		if task.BusinessUniqueID != expectedBusinessID {
			t.Errorf("Task %d: expected business ID %s, got %s",
				i, expectedBusinessID, task.BusinessUniqueID)
		}
	}
}

func TestIntegration_TaskStats(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	stats, err := client.Tasks().Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get task stats: %v", err)
	}

	if stats.TotalTasks < 0 {
		t.Error("Total tasks should not be negative")
	}

	// 验证状态计数的总和等于总任务数
	var statusSum int
	for _, count := range stats.StatusCounts {
		statusSum += count
	}

	if statusSum != stats.TotalTasks {
		t.Errorf("Status counts sum (%d) does not match total tasks (%d)",
			statusSum, stats.TotalTasks)
	}
}

func TestIntegration_TaskDeletion(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 创建要删除的任务
	task := NewTask(
		"integration-delete-test-"+time.Now().Format("20060102-150405"),
		"https://httpbin.org/post",
	)

	ctx := context.Background()
	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// 删除任务
	err = client.Tasks().Delete(ctx, createdTask.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// 验证任务已被删除
	_, err = client.Tasks().Get(ctx, createdTask.ID)
	if err == nil {
		t.Error("Expected error when getting deleted task")
	}

	if !IsNotFoundError(err) {
		t.Errorf("Expected NotFoundError, got %v", err)
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	config := getTestConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试获取不存在的任务
	_, err = client.Tasks().Get(ctx, 999999)
	if err == nil {
		t.Error("Expected error when getting non-existent task")
	}

	if !IsNotFoundError(err) {
		t.Errorf("Expected NotFoundError, got %v", err)
	}

	// 测试创建无效任务
	invalidTask := &CreateTaskRequest{
		BusinessUniqueID: "", // 空的业务ID应该导致验证错误
		CallbackURL:      "https://example.com",
	}

	_, err = client.Tasks().Create(ctx, invalidTask)
	if err == nil {
		t.Error("Expected error when creating invalid task")
	}

	// 注意：这里可能会得到验证错误或其他类型的错误，取决于服务端实现
	t.Logf("Got expected error: %v", err)
}