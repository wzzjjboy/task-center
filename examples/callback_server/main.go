package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/task-center/sdk"
)

func main() {
	fmt.Println("🚀 TaskCenter Callback Server Example")

	// 检查环境变量
	apiSecret := getEnvOrDefault("TASKCENTER_API_SECRET", "demo-secret-key-for-development")
	port := getEnvOrDefault("CALLBACK_PORT", "8080")

	// 创建回调处理器
	handler := createCallbackHandler()

	// 创建带中间件的回调服务器
	server := sdk.NewCallbackServer(
		apiSecret,
		handler,
		sdk.WithCallbackMiddleware(createLoggingMiddleware()),
		sdk.WithCallbackMiddleware(createMetricsMiddleware()),
	)

	// 设置HTTP服务器
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		fmt.Printf("📡 Starting callback server on port %s\n", port)
		fmt.Printf("🔗 Webhook URL: http://localhost:%s/webhook\n", port)
		fmt.Printf("💚 Health check: http://localhost:%s/health\n", port)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号来优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down callback server...")

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("✅ Callback server stopped")
}

func createCallbackHandler() sdk.CallbackHandler {
	return &sdk.DefaultCallbackHandler{
		OnTaskCreated: func(event *sdk.CallbackEvent) error {
			fmt.Printf("\n📝 Task Created Event\n")
			fmt.Printf("   Task ID: %d\n", event.TaskID)
			fmt.Printf("   Business ID: %s\n", event.Task.BusinessUniqueID)
			fmt.Printf("   Callback URL: %s\n", event.Task.CallbackURL)
			fmt.Printf("   Priority: %d\n", int(event.Task.Priority))
			fmt.Printf("   Tags: %v\n", event.Task.Tags)
			fmt.Printf("   Event Time: %s\n", event.EventTime.Format(time.RFC3339))

			// 这里可以添加业务逻辑，比如：
			// - 发送通知
			// - 更新数据库记录
			// - 记录审计日志
			logTaskEvent("CREATED", event)

			return nil
		},

		OnTaskStarted: func(event *sdk.CallbackEvent) error {
			fmt.Printf("\n🚀 Task Started Event\n")
			fmt.Printf("   Task ID: %d\n", event.TaskID)
			fmt.Printf("   Business ID: %s\n", event.Task.BusinessUniqueID)
			fmt.Printf("   Started At: %s\n", event.Task.ExecutedAt.Format(time.RFC3339))
			fmt.Printf("   Current Retry: %d/%d\n", event.Task.CurrentRetry, event.Task.MaxRetries)

			// 检查是否为重试执行
			if event.Task.CurrentRetry > 0 {
				fmt.Printf("   ⚠️  This is a retry attempt\n")
			}

			logTaskEvent("STARTED", event)

			return nil
		},

		OnTaskCompleted: func(event *sdk.CallbackEvent) error {
			fmt.Printf("\n✅ Task Completed Event\n")
			fmt.Printf("   Task ID: %d\n", event.TaskID)
			fmt.Printf("   Business ID: %s\n", event.Task.BusinessUniqueID)
			fmt.Printf("   Completed At: %s\n", event.Task.CompletedAt.Format(time.RFC3339))

			// 计算执行时间
			if event.Task.ExecutedAt != nil && event.Task.CompletedAt != nil {
				duration := event.Task.CompletedAt.Sub(*event.Task.ExecutedAt)
				fmt.Printf("   Duration: %s\n", duration)
			}

			// 根据任务类型执行不同的完成处理
			return handleTaskCompletion(event)
		},

		OnTaskFailed: func(event *sdk.CallbackEvent) error {
			fmt.Printf("\n❌ Task Failed Event\n")
			fmt.Printf("   Task ID: %d\n", event.TaskID)
			fmt.Printf("   Business ID: %s\n", event.Task.BusinessUniqueID)
			fmt.Printf("   Error: %s\n", event.Task.ErrorMessage)
			fmt.Printf("   Retry Count: %d/%d\n", event.Task.CurrentRetry, event.Task.MaxRetries)

			// 检查是否还会重试
			if event.Task.CurrentRetry < event.Task.MaxRetries {
				fmt.Printf("   🔄 Will retry automatically\n")
				if event.Task.NextExecuteAt != nil {
					fmt.Printf("   Next attempt at: %s\n", event.Task.NextExecuteAt.Format(time.RFC3339))
				}
			} else {
				fmt.Printf("   🚫 No more retries - task permanently failed\n")
				// 发送告警通知
				sendFailureAlert(event)
			}

			logTaskEvent("FAILED", event)

			return nil
		},
	}
}

func handleTaskCompletion(event *sdk.CallbackEvent) error {
	// 根据任务标签或业务ID执行不同的处理逻辑
	tags := event.Task.Tags

	if contains(tags, "payment") {
		return handlePaymentCompletion(event)
	}

	if contains(tags, "marketing") {
		return handleMarketingCompletion(event)
	}

	if contains(tags, "data-processing") {
		return handleDataProcessingCompletion(event)
	}

	// 默认处理
	fmt.Printf("   📄 Generic task completion processing\n")
	return nil
}

func handlePaymentCompletion(event *sdk.CallbackEvent) error {
	fmt.Printf("   💳 Processing payment completion\n")

	// 模拟支付完成处理
	fmt.Printf("   • Updating payment status in database\n")
	fmt.Printf("   • Sending confirmation email to user\n")
	fmt.Printf("   • Triggering order fulfillment\n")

	// 这里可以添加实际的业务逻辑：
	// - 更新数据库中的支付状态
	// - 发送邮件通知
	// - 调用其他服务API
	// - 更新用户余额等

	return nil
}

func handleMarketingCompletion(event *sdk.CallbackEvent) error {
	fmt.Printf("   📧 Processing marketing task completion\n")

	// 模拟营销任务完成处理
	fmt.Printf("   • Updating campaign statistics\n")
	fmt.Printf("   • Recording email delivery metrics\n")
	fmt.Printf("   • Scheduling follow-up campaigns\n")

	return nil
}

func handleDataProcessingCompletion(event *sdk.CallbackEvent) error {
	fmt.Printf("   📊 Processing data processing completion\n")

	// 模拟数据处理完成
	fmt.Printf("   • Updating processing status\n")
	fmt.Printf("   • Generating report\n")
	fmt.Printf("   • Notifying stakeholders\n")

	return nil
}

func sendFailureAlert(event *sdk.CallbackEvent) {
	fmt.Printf("   🚨 Sending failure alert\n")
	fmt.Printf("   • Alert: Task %d (%s) permanently failed\n",
		event.TaskID, event.Task.BusinessUniqueID)
	fmt.Printf("   • Error: %s\n", event.Task.ErrorMessage)

	// 这里可以集成实际的告警系统：
	// - 发送邮件告警
	// - 发送Slack通知
	// - 调用监控系统API
	// - 写入错误日志
}

func logTaskEvent(eventType string, event *sdk.CallbackEvent) {
	// 模拟结构化日志记录
	fmt.Printf("   📝 Logging event: %s for task %d\n", eventType, event.TaskID)

	// 实际应用中可以使用结构化日志库如 logrus, zap 等
	// log.WithFields(logrus.Fields{
	//     "event_type": eventType,
	//     "task_id": event.TaskID,
	//     "business_id": event.Task.BusinessUniqueID,
	//     "status": event.Task.Status,
	//     "tags": event.Task.Tags,
	// }).Info("Task event processed")
}

func createLoggingMiddleware() sdk.CallbackMiddleware {
	return &sdk.LoggingMiddleware{
		Logger: func(level, message string, fields map[string]interface{}) {
			fmt.Printf("[%s] %s", level, message)
			if len(fields) > 0 {
				fmt.Printf(" - %+v", fields)
			}
			fmt.Println()
		},
	}
}

func createMetricsMiddleware() sdk.CallbackMiddleware {
	// 简单的内存指标收集器
	metrics := make(map[string]int)

	return &sdk.MetricsMiddleware{
		IncCounter: func(name string, labels map[string]string) {
			key := name
			if eventType, ok := labels["event_type"]; ok {
				key = name + "_" + eventType
			}
			metrics[key]++

			// 每处理10个事件打印一次指标
			if metrics[key]%10 == 0 {
				fmt.Printf("📈 Metrics: %s = %d\n", key, metrics[key])
			}
		},
		RecordTime: func(name string, duration time.Duration, labels map[string]string) {
			fmt.Printf("⏱️  Timing: %s took %s\n", name, duration)
		},
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}