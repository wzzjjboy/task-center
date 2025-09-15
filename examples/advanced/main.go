package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/your-org/task-center/sdk"
)

func main() {
	// 创建客户端
	client, err := createClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("🚀 TaskCenter Advanced Example")

	// 1. 创建复杂任务
	fmt.Println("\n1. Creating advanced tasks with different priorities...")
	createAdvancedTasks(client)

	// 2. 批量创建任务
	fmt.Println("\n2. Batch creating multiple tasks...")
	batchCreateTasks(client)

	// 3. 任务管理操作
	fmt.Println("\n3. Task management operations...")
	taskManagement(client)

	// 4. 错误处理演示
	fmt.Println("\n4. Error handling demonstration...")
	errorHandlingDemo(client)

	// 5. 获取统计信息
	fmt.Println("\n5. Getting task statistics...")
	getTaskStats(client)

	fmt.Println("\n✅ Advanced example completed!")
}

func createClient() (*sdk.Client, error) {
	// 从环境变量或设置默认值
	apiURL := getEnvOrDefault("TASKCENTER_API_URL", "http://localhost:8080")
	apiKey := getEnvOrDefault("TASKCENTER_API_KEY", "demo-api-key")
	businessIDStr := getEnvOrDefault("TASKCENTER_BUSINESS_ID", "1")

	businessID, err := strconv.ParseInt(businessIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID: %w", err)
	}

	// 创建自定义配置
	config := sdk.DefaultConfig()
	config.BaseURL = apiURL
	config.APIKey = apiKey
	config.BusinessID = businessID
	config.Timeout = 60 * time.Second // 增加超时时间

	// 自定义重试策略
	config.RetryPolicy = &sdk.RetryPolicy{
		MaxRetries:      5,
		InitialInterval: 2 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		RetryableErrors: []int{429, 500, 502, 503, 504},
	}

	client, err := sdk.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

func createAdvancedTasks(client *sdk.Client) {
	ctx := context.Background()
	timestamp := time.Now().Format("20060102-150405")

	// 创建高优先级支付任务
	paymentTask := sdk.NewTask(
		"payment-"+timestamp,
		"https://api.example.com/webhook/payment",
	).
		WithPriority(sdk.TaskPriorityHighest).
		WithTimeout(30).                              // 30秒超时
		WithRetries(5, sdk.FastRetryIntervals).      // 快速重试
		WithTags("payment", "critical", "financial"). // 多个标签
		WithHeaders(map[string]string{
			"Authorization": "Bearer " + getEnvOrDefault("WEBHOOK_TOKEN", "demo-token"),
			"Content-Type":  "application/json",
			"X-Source":      "taskcenter-advanced-example",
		}).
		WithBody(`{
			"transaction_id": "txn_` + timestamp + `",
			"amount": 299.99,
			"currency": "USD",
			"payment_method": "credit_card"
		}`).
		WithMetadata(map[string]interface{}{
			"user_id":      12345,
			"merchant_id":  "merchant_abc",
			"order_total":  299.99,
			"created_from": "advanced_example",
		})

	createdPayment, err := client.Tasks().Create(ctx, paymentTask)
	if err != nil {
		log.Printf("❌ Failed to create payment task: %v", err)
	} else {
		fmt.Printf("💳 Payment task created: ID=%d, Priority=%d\n",
			createdPayment.ID, int(createdPayment.Priority))
	}

	// 创建定时营销任务
	marketingTime := time.Now().Add(2 * time.Minute)
	marketingTask := sdk.NewScheduledTask(
		"marketing-"+timestamp,
		"https://api.example.com/webhook/marketing",
		marketingTime,
	).
		WithPriority(sdk.TaskPriorityNormal).
		WithTags("marketing", "newsletter", "automation").
		WithMetadata(map[string]interface{}{
			"campaign_id":   "newsletter_weekly",
			"target_users":  []int{1001, 1002, 1003},
			"template_name": "weekly_digest",
		})

	createdMarketing, err := client.Tasks().Create(ctx, marketingTask)
	if err != nil {
		log.Printf("❌ Failed to create marketing task: %v", err)
	} else {
		fmt.Printf("📧 Marketing task scheduled: ID=%d, Execute at=%s\n",
			createdMarketing.ID, createdMarketing.ScheduledAt.Format("15:04:05"))
	}

	// 创建低优先级清理任务
	cleanupTask := sdk.NewTask(
		"cleanup-"+timestamp,
		"https://api.example.com/webhook/cleanup",
	).
		WithPriority(sdk.TaskPriorityLow).
		WithTimeout(300).                        // 5分钟超时
		WithRetries(2, sdk.SlowRetryIntervals). // 慢速重试
		WithTags("maintenance", "cleanup", "background")

	createdCleanup, err := client.Tasks().Create(ctx, cleanupTask)
	if err != nil {
		log.Printf("❌ Failed to create cleanup task: %v", err)
	} else {
		fmt.Printf("🧹 Cleanup task created: ID=%d, Priority=%d\n",
			createdCleanup.ID, int(createdCleanup.Priority))
	}
}

func batchCreateTasks(client *sdk.Client) {
	ctx := context.Background()
	timestamp := time.Now().Format("20060102-150405")

	// 创建多个数据处理任务
	tasks := []sdk.CreateTaskRequest{}
	for i := 1; i <= 5; i++ {
		task := *sdk.NewTask(
			fmt.Sprintf("batch-data-processing-%s-%d", timestamp, i),
			"https://api.example.com/webhook/data-processing",
		).
			WithPriority(sdk.TaskPriorityNormal).
			WithTags("batch", "data-processing", fmt.Sprintf("batch-%d", i)).
			WithMetadata(map[string]interface{}{
				"batch_id":    fmt.Sprintf("batch_%s_%d", timestamp, i),
				"data_source": "user_analytics",
				"chunk_size":  1000,
			})

		tasks = append(tasks, task)
	}

	batchReq := &sdk.BatchCreateTasksRequest{
		Tasks: tasks,
	}

	response, err := client.Tasks().BatchCreate(ctx, batchReq)
	if err != nil {
		log.Printf("❌ Failed to batch create tasks: %v", err)
		return
	}

	fmt.Printf("📦 Batch creation results:\n")
	fmt.Printf("   ✅ Successfully created: %d tasks\n", len(response.Succeeded))
	fmt.Printf("   ❌ Failed to create: %d tasks\n", len(response.Failed))

	for _, task := range response.Succeeded {
		fmt.Printf("      - Task %d: %s\n", task.ID, task.BusinessUniqueID)
	}

	for _, failure := range response.Failed {
		fmt.Printf("      - Index %d failed: %s\n", failure.Index, failure.Error)
	}
}

func taskManagement(client *sdk.Client) {
	ctx := context.Background()

	// 查询活跃任务
	activeReq := sdk.NewListTasksRequest().
		WithStatus(sdk.TaskStatusPending, sdk.TaskStatusRunning).
		WithPagination(1, 5)

	activeResponse, err := client.Tasks().List(ctx, activeReq)
	if err != nil {
		log.Printf("❌ Failed to list active tasks: %v", err)
		return
	}

	fmt.Printf("🔄 Active tasks found: %d\n", len(activeResponse.Tasks))

	// 如果有活跃任务，演示管理操作
	if len(activeResponse.Tasks) > 0 {
		taskID := activeResponse.Tasks[0].ID
		fmt.Printf("   Operating on task ID: %d\n", taskID)

		// 更新任务优先级
		updateReq := &sdk.UpdateTaskRequest{
			Priority: &sdk.TaskPriorityHigh,
		}

		updatedTask, err := client.Tasks().Update(ctx, taskID, updateReq)
		if err != nil {
			log.Printf("   ❌ Failed to update task: %v", err)
		} else {
			fmt.Printf("   ✅ Updated task priority to: %d\n", int(updatedTask.Priority))
		}

		// 尝试取消任务（如果状态允许）
		if activeResponse.Tasks[0].Status == sdk.TaskStatusPending {
			err = client.Tasks().Cancel(ctx, taskID)
			if err != nil {
				log.Printf("   ❌ Failed to cancel task: %v", err)
			} else {
				fmt.Printf("   🚫 Task cancelled successfully\n")
			}
		}
	}

	// 查询带特定标签的任务
	taggedReq := sdk.NewListTasksRequest().
		WithTagsFilter("batch").
		WithPagination(1, 3)

	taggedResponse, err := client.Tasks().List(ctx, taggedReq)
	if err != nil {
		log.Printf("❌ Failed to list tagged tasks: %v", err)
	} else {
		fmt.Printf("🏷️  Tasks with 'batch' tag: %d\n", len(taggedResponse.Tasks))
	}
}

func errorHandlingDemo(client *sdk.Client) {
	ctx := context.Background()

	// 尝试创建一个无效的任务来演示错误处理
	invalidTask := &sdk.CreateTaskRequest{
		BusinessUniqueID: "", // 空的业务ID，应该导致验证错误
		CallbackURL:      "invalid-url",
	}

	_, err := client.Tasks().Create(ctx, invalidTask)
	if err != nil {
		fmt.Printf("🔍 Error handling demonstration:\n")

		switch {
		case sdk.IsValidationError(err):
			fmt.Printf("   ✅ Caught validation error: %s\n", err.Error())

		case sdk.IsAuthenticationError(err):
			fmt.Printf("   🔐 Authentication error: %s\n", err.Error())

		case sdk.IsNetworkError(err):
			fmt.Printf("   🌐 Network error: %s\n", err.Error())

		default:
			fmt.Printf("   ❓ Other error: %s\n", err.Error())
		}

		// 检查错误详情
		if sdkErr, ok := err.(sdk.Error); ok {
			fmt.Printf("   Error code: %s\n", sdkErr.Code())
			fmt.Printf("   HTTP status: %d\n", sdkErr.StatusCode())
		}
	} else {
		fmt.Printf("⚠️  Expected validation error but task was created\n")
	}

	// 尝试查询不存在的任务
	_, err = client.Tasks().Get(ctx, 999999)
	if err != nil {
		if sdk.IsNotFoundError(err) {
			fmt.Printf("   ✅ Correctly caught not found error for non-existent task\n")
		} else {
			fmt.Printf("   ❓ Unexpected error for non-existent task: %s\n", err.Error())
		}
	}
}

func getTaskStats(client *sdk.Client) {
	ctx := context.Background()

	stats, err := client.Tasks().Stats(ctx)
	if err != nil {
		log.Printf("❌ Failed to get stats: %v", err)
		return
	}

	fmt.Printf("📊 Task Statistics:\n")
	fmt.Printf("   Total tasks: %d\n", stats.TotalTasks)

	fmt.Printf("   Status distribution:\n")
	for status, count := range stats.StatusCounts {
		fmt.Printf("      %s: %d\n", status.String(), count)
	}

	fmt.Printf("   Priority distribution:\n")
	for priority, count := range stats.PriorityCounts {
		fmt.Printf("      Priority %d: %d tasks\n", int(priority), count)
	}

	if len(stats.TagCounts) > 0 {
		fmt.Printf("   Top tags:\n")
		count := 0
		for tag, freq := range stats.TagCounts {
			if count >= 5 {
				break
			}
			fmt.Printf("      %s: %d\n", tag, freq)
			count++
		}
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}