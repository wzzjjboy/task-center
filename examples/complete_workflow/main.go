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

// E-commerce workflow 示例：从订单创建到完成的完整流程
func main() {
	fmt.Println("🛒 TaskCenter Complete E-commerce Workflow Example")

	client, err := createClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 模拟电商订单处理工作流
	orderID := "order-" + time.Now().Format("20060102-150405")
	userID := "user-12345"

	fmt.Printf("🆕 Starting workflow for order: %s\n", orderID)

	// 步骤1：创建支付处理任务
	fmt.Println("\n1️⃣ Creating payment processing task...")
	paymentTaskID := createPaymentTask(client, orderID, userID)

	// 步骤2：创建库存预留任务
	fmt.Println("\n2️⃣ Creating inventory reservation task...")
	inventoryTaskID := createInventoryTask(client, orderID)

	// 步骤3：创建订单确认邮件任务（依赖支付完成）
	fmt.Println("\n3️⃣ Creating order confirmation email task...")
	emailTaskID := createEmailTask(client, orderID, userID)

	// 步骤4：创建延迟的物流配送任务（1小时后）
	fmt.Println("\n4️⃣ Creating shipping task (scheduled for 1 hour later)...")
	shippingTaskID := createShippingTask(client, orderID)

	// 步骤5：创建数据分析任务（低优先级）
	fmt.Println("\n5️⃣ Creating analytics task...")
	analyticsTaskID := createAnalyticsTask(client, orderID, userID)

	// 步骤6：监控所有任务状态
	fmt.Println("\n6️⃣ Monitoring task progress...")
	monitorWorkflow(client, map[string]int64{
		"payment":   paymentTaskID,
		"inventory": inventoryTaskID,
		"email":     emailTaskID,
		"shipping":  shippingTaskID,
		"analytics": analyticsTaskID,
	})

	// 步骤7：演示批量任务管理
	fmt.Println("\n7️⃣ Demonstrating batch operations...")
	demonstrateBatchOperations(client, orderID)

	// 步骤8：显示工作流统计
	fmt.Println("\n8️⃣ Workflow statistics...")
	showWorkflowStats(client)

	fmt.Println("\n✅ Complete workflow example finished!")
}

func createClient() (*sdk.Client, error) {
	apiURL := getEnvOrDefault("TASKCENTER_API_URL", "http://localhost:8080")
	apiKey := getEnvOrDefault("TASKCENTER_API_KEY", "demo-api-key")
	businessIDStr := getEnvOrDefault("TASKCENTER_BUSINESS_ID", "1")

	businessID, err := strconv.ParseInt(businessIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID: %w", err)
	}

	config := sdk.DefaultConfig()
	config.BaseURL = apiURL
	config.APIKey = apiKey
	config.BusinessID = businessID

	return sdk.NewClient(config)
}

func createPaymentTask(client *sdk.Client, orderID, userID string) int64 {
	ctx := context.Background()

	task := sdk.NewTask(
		orderID+"-payment",
		"https://api.example.com/webhook/payment",
	).
		WithPriority(sdk.TaskPriorityHighest). // 最高优先级
		WithTimeout(30).                       // 30秒超时
		WithRetries(3, sdk.FastRetryIntervals). // 快速重试
		WithTags("payment", "critical", "ecommerce", "order").
		WithHeaders(map[string]string{
			"Authorization": "Bearer " + getEnvOrDefault("PAYMENT_API_TOKEN", "demo-token"),
			"Content-Type":  "application/json",
			"Idempotency-Key": orderID + "-payment",
		}).
		WithBody(fmt.Sprintf(`{
			"order_id": "%s",
			"user_id": "%s",
			"amount": 99.99,
			"currency": "USD",
			"payment_method": "credit_card",
			"card_token": "tok_visa_4242"
		}`, orderID, userID)).
		WithMetadata(map[string]interface{}{
			"order_id":       orderID,
			"user_id":        userID,
			"payment_amount": 99.99,
			"step":           "payment_processing",
			"workflow_type":  "ecommerce_order",
		})

	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create payment task: %v", err)
		return 0
	}

	fmt.Printf("   💳 Payment task created: ID=%d\n", createdTask.ID)
	return createdTask.ID
}

func createInventoryTask(client *sdk.Client, orderID string) int64 {
	ctx := context.Background()

	task := sdk.NewTask(
		orderID+"-inventory",
		"https://api.example.com/webhook/inventory",
	).
		WithPriority(sdk.TaskPriorityHigh).
		WithTimeout(60).
		WithRetries(2, sdk.StandardRetryIntervals).
		WithTags("inventory", "reservation", "ecommerce", "order").
		WithBody(fmt.Sprintf(`{
			"order_id": "%s",
			"items": [
				{"sku": "PROD-001", "quantity": 2},
				{"sku": "PROD-002", "quantity": 1}
			],
			"warehouse": "main"
		}`, orderID)).
		WithMetadata(map[string]interface{}{
			"order_id":      orderID,
			"step":          "inventory_reservation",
			"workflow_type": "ecommerce_order",
		})

	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create inventory task: %v", err)
		return 0
	}

	fmt.Printf("   📦 Inventory task created: ID=%d\n", createdTask.ID)
	return createdTask.ID
}

func createEmailTask(client *sdk.Client, orderID, userID string) int64 {
	ctx := context.Background()

	task := sdk.NewTask(
		orderID+"-email",
		"https://api.example.com/webhook/email",
	).
		WithPriority(sdk.TaskPriorityNormal).
		WithTimeout(30).
		WithRetries(2, sdk.StandardRetryIntervals).
		WithTags("email", "notification", "ecommerce", "order").
		WithBody(fmt.Sprintf(`{
			"order_id": "%s",
			"user_id": "%s",
			"template": "order_confirmation",
			"recipient": "user@example.com",
			"data": {
				"order_number": "%s",
				"total": "$99.99",
				"items_count": 3
			}
		}`, orderID, userID, orderID)).
		WithMetadata(map[string]interface{}{
			"order_id":      orderID,
			"user_id":       userID,
			"step":          "email_notification",
			"workflow_type": "ecommerce_order",
		})

	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create email task: %v", err)
		return 0
	}

	fmt.Printf("   📧 Email task created: ID=%d\n", createdTask.ID)
	return createdTask.ID
}

func createShippingTask(client *sdk.Client, orderID string) int64 {
	ctx := context.Background()

	// 1小时后执行
	scheduledTime := time.Now().Add(1 * time.Hour)

	task := sdk.NewScheduledTask(
		orderID+"-shipping",
		"https://api.example.com/webhook/shipping",
		scheduledTime,
	).
		WithPriority(sdk.TaskPriorityNormal).
		WithTimeout(120).
		WithRetries(3, sdk.StandardRetryIntervals).
		WithTags("shipping", "logistics", "ecommerce", "order").
		WithBody(fmt.Sprintf(`{
			"order_id": "%s",
			"shipping_address": {
				"street": "123 Main St",
				"city": "Anytown",
				"state": "CA",
				"zip": "12345"
			},
			"shipping_method": "standard",
			"carrier": "ups"
		}`, orderID)).
		WithMetadata(map[string]interface{}{
			"order_id":      orderID,
			"step":          "shipping_label",
			"workflow_type": "ecommerce_order",
		})

	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create shipping task: %v", err)
		return 0
	}

	fmt.Printf("   🚚 Shipping task scheduled: ID=%d, Execute at=%s\n",
		createdTask.ID, scheduledTime.Format("15:04:05"))
	return createdTask.ID
}

func createAnalyticsTask(client *sdk.Client, orderID, userID string) int64 {
	ctx := context.Background()

	task := sdk.NewTask(
		orderID+"-analytics",
		"https://api.example.com/webhook/analytics",
	).
		WithPriority(sdk.TaskPriorityLow). // 低优先级，不影响关键流程
		WithTimeout(300).
		WithRetries(1, sdk.SlowRetryIntervals).
		WithTags("analytics", "reporting", "ecommerce", "order").
		WithBody(fmt.Sprintf(`{
			"event": "order_created",
			"order_id": "%s",
			"user_id": "%s",
			"timestamp": "%s",
			"properties": {
				"total_amount": 99.99,
				"items_count": 3,
				"payment_method": "credit_card",
				"channel": "web"
			}
		}`, orderID, userID, time.Now().Format(time.RFC3339))).
		WithMetadata(map[string]interface{}{
			"order_id":      orderID,
			"user_id":       userID,
			"step":          "analytics_tracking",
			"workflow_type": "ecommerce_order",
		})

	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create analytics task: %v", err)
		return 0
	}

	fmt.Printf("   📊 Analytics task created: ID=%d\n", createdTask.ID)
	return createdTask.ID
}

func monitorWorkflow(client *sdk.Client, tasks map[string]int64) {
	ctx := context.Background()

	fmt.Printf("   📋 Monitoring %d tasks...\n", len(tasks))

	for name, taskID := range tasks {
		if taskID == 0 {
			continue
		}

		task, err := client.Tasks().Get(ctx, taskID)
		if err != nil {
			log.Printf("   ❌ Failed to get %s task: %v", name, err)
			continue
		}

		statusIcon := getStatusIcon(task.Status)
		fmt.Printf("   %s %s: %s (Priority: %d)\n",
			statusIcon, name, task.Status.String(), int(task.Priority))

		// 如果任务失败，显示错误信息
		if task.Status == sdk.TaskStatusFailed && task.ErrorMessage != "" {
			fmt.Printf("      Error: %s\n", task.ErrorMessage)
		}

		// 如果是定时任务，显示执行时间
		if task.ScheduledAt.After(time.Now()) {
			fmt.Printf("      Scheduled for: %s\n", task.ScheduledAt.Format("15:04:05"))
		}
	}
}

func demonstrateBatchOperations(client *sdk.Client, orderID string) {
	ctx := context.Background()

	// 创建一组后续处理任务
	followUpTasks := []sdk.CreateTaskRequest{}

	// 创建多个营销相关任务
	marketingTasks := []string{
		"recommendation_email",
		"review_request",
		"loyalty_points",
		"cross_sell_campaign",
	}

	for _, taskType := range marketingTasks {
		task := *sdk.NewTask(
			fmt.Sprintf("%s-%s", orderID, taskType),
			"https://api.example.com/webhook/marketing",
		).
			WithPriority(sdk.TaskPriorityLow).
			WithTags("marketing", "follow-up", taskType).
			WithSchedule(time.Now().Add(24 * time.Hour)). // 24小时后执行
			WithMetadata(map[string]interface{}{
				"order_id":      orderID,
				"task_type":     taskType,
				"workflow_type": "post_order_marketing",
			})

		followUpTasks = append(followUpTasks, task)
	}

	// 批量创建任务
	batchReq := &sdk.BatchCreateTasksRequest{
		Tasks: followUpTasks,
	}

	response, err := client.Tasks().BatchCreate(ctx, batchReq)
	if err != nil {
		log.Printf("❌ Failed to batch create follow-up tasks: %v", err)
		return
	}

	fmt.Printf("   📦 Batch created %d follow-up tasks\n", len(response.Succeeded))
	fmt.Printf("   ❌ Failed to create %d tasks\n", len(response.Failed))

	// 显示创建成功的任务
	for _, task := range response.Succeeded {
		fmt.Printf("      ✅ %s (ID: %d)\n", task.BusinessUniqueID, task.ID)
	}
}

func showWorkflowStats(client *sdk.Client) {
	ctx := context.Background()

	// 获取整体统计
	stats, err := client.Tasks().Stats(ctx)
	if err != nil {
		log.Printf("❌ Failed to get stats: %v", err)
		return
	}

	fmt.Printf("   📊 Overall Statistics:\n")
	fmt.Printf("      Total tasks: %d\n", stats.TotalTasks)

	// 状态分布
	for status, count := range stats.StatusCounts {
		icon := getStatusIcon(status)
		fmt.Printf("      %s %s: %d\n", icon, status.String(), count)
	}

	// 查询电商相关任务
	ecommerceReq := sdk.NewListTasksRequest().
		WithTagsFilter("ecommerce").
		WithDateRange(time.Now().Add(-1*time.Hour), time.Now()).
		WithPagination(1, 50)

	ecommerceResponse, err := client.Tasks().List(ctx, ecommerceReq)
	if err != nil {
		log.Printf("❌ Failed to list ecommerce tasks: %v", err)
		return
	}

	fmt.Printf("   🛒 E-commerce tasks in last hour: %d\n", len(ecommerceResponse.Tasks))

	// 按优先级分组显示
	priorityGroups := make(map[sdk.TaskPriority][]sdk.Task)
	for _, task := range ecommerceResponse.Tasks {
		priorityGroups[task.Priority] = append(priorityGroups[task.Priority], task)
	}

	for priority, tasks := range priorityGroups {
		fmt.Printf("      Priority %d: %d tasks\n", int(priority), len(tasks))
	}
}

func getStatusIcon(status sdk.TaskStatus) string {
	switch status {
	case sdk.TaskStatusPending:
		return "⏳"
	case sdk.TaskStatusRunning:
		return "🔄"
	case sdk.TaskStatusSucceeded:
		return "✅"
	case sdk.TaskStatusFailed:
		return "❌"
	case sdk.TaskStatusCancelled:
		return "🚫"
	case sdk.TaskStatusExpired:
		return "⏰"
	default:
		return "❓"
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}