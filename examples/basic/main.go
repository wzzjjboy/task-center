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

	fmt.Println("🚀 TaskCenter Basic Example")

	// 1. 创建简单任务
	fmt.Println("\n1. Creating a simple task...")
	createSimpleTask(client)

	// 2. 查询任务
	fmt.Println("\n2. Querying tasks...")
	queryTasks(client)

	// 3. 创建定时任务
	fmt.Println("\n3. Creating a scheduled task...")
	createScheduledTask(client)

	// 4. 列出所有任务
	fmt.Println("\n4. Listing tasks...")
	listTasks(client)

	fmt.Println("\n✅ Basic example completed!")
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

	// 创建客户端配置
	config := sdk.DefaultConfig()
	config.BaseURL = apiURL
	config.APIKey = apiKey
	config.BusinessID = businessID

	client, err := sdk.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

func createSimpleTask(client *sdk.Client) {
	// 创建任务
	task := sdk.NewTask(
		"basic-example-"+time.Now().Format("20060102-150405"),
		"https://httpbin.org/post", // 使用 httpbin 作为测试回调
	)

	ctx := context.Background()
	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create task: %v", err)
		return
	}

	fmt.Printf("✅ Task created successfully!\n")
	fmt.Printf("   ID: %d\n", createdTask.ID)
	fmt.Printf("   Business ID: %s\n", createdTask.BusinessUniqueID)
	fmt.Printf("   Status: %s\n", createdTask.Status.String())
	fmt.Printf("   Callback URL: %s\n", createdTask.CallbackURL)
}

func queryTasks(client *sdk.Client) {
	ctx := context.Background()

	// 尝试查询一个任务（假设存在ID为1的任务）
	task, err := client.Tasks().Get(ctx, 1)
	if err != nil {
		if sdk.IsNotFoundError(err) {
			fmt.Println("📭 Task with ID 1 not found")
		} else {
			log.Printf("❌ Failed to get task: %v", err)
		}
		return
	}

	fmt.Printf("📋 Found task:\n")
	fmt.Printf("   ID: %d\n", task.ID)
	fmt.Printf("   Business ID: %s\n", task.BusinessUniqueID)
	fmt.Printf("   Status: %s\n", task.Status.String())
	fmt.Printf("   Created: %s\n", task.CreatedAt.Format(time.RFC3339))
}

func createScheduledTask(client *sdk.Client) {
	// 创建30秒后执行的任务
	scheduledTime := time.Now().Add(30 * time.Second)

	task := sdk.NewScheduledTask(
		"scheduled-example-"+time.Now().Format("20060102-150405"),
		"https://httpbin.org/post",
		scheduledTime,
	).WithTags("scheduled", "example")

	ctx := context.Background()
	createdTask, err := client.Tasks().Create(ctx, task)
	if err != nil {
		log.Printf("❌ Failed to create scheduled task: %v", err)
		return
	}

	fmt.Printf("⏰ Scheduled task created!\n")
	fmt.Printf("   ID: %d\n", createdTask.ID)
	fmt.Printf("   Will execute at: %s\n", createdTask.ScheduledAt.Format(time.RFC3339))
	fmt.Printf("   Tags: %v\n", createdTask.Tags)
}

func listTasks(client *sdk.Client) {
	// 查询最近创建的任务
	req := sdk.NewListTasksRequest().
		WithPagination(1, 10). // 第一页，10条记录
		WithDateRange(
			time.Now().Add(-24*time.Hour), // 过去24小时
			time.Now(),
		)

	ctx := context.Background()
	response, err := client.Tasks().List(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to list tasks: %v", err)
		return
	}

	fmt.Printf("📊 Task list (showing %d of %d total):\n", len(response.Tasks), response.Total)
	for i, task := range response.Tasks {
		fmt.Printf("   %d. ID=%d, Business=%s, Status=%s, Created=%s\n",
			i+1,
			task.ID,
			task.BusinessUniqueID,
			task.Status.String(),
			task.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}