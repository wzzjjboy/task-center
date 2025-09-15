package auth_test

import (
	"context"
	"fmt"
	"net/http"

	"task-center/sdk/auth"
)

// ExampleAPIKeyAuth 演示API Key认证的使用
func ExampleAPIKeyAuth() {
	// 创建API Key认证器
	auth := auth.NewAPIKeyAuth("your-api-key-here-1234567890")

	// 创建HTTP请求
	req, _ := http.NewRequest("GET", "https://api.example.com/tasks", nil)

	// 添加认证头
	if err := auth.Authenticate(req); err != nil {
		fmt.Printf("认证失败: %v\n", err)
		return
	}

	fmt.Printf("认证头已添加: %s\n", req.Header.Get("Authorization"))
	// Output: 认证头已添加: Bearer your-api-key-here-1234567890
}

// ExampleAPIKeyManager 演示API Key管理器的使用
func ExampleAPIKeyManager() {
	// 创建API Key管理器
	manager, err := auth.NewAPIKeyManager("your-api-key-here-1234567890")
	if err != nil {
		fmt.Printf("创建管理器失败: %v\n", err)
		return
	}

	// 检查凭证是否有效
	fmt.Printf("凭证有效: %t\n", manager.IsValid())

	// 获取认证器实例
	authenticator := manager.GetAuth()

	// 更新API Key
	err = manager.SetAPIKey("new-api-key-here-0987654321")
	if err != nil {
		fmt.Printf("更新API Key失败: %v\n", err)
		return
	}

	fmt.Printf("API Key已更新\n")
	fmt.Printf("新的凭证有效: %t\n", manager.IsValid())

	// 刷新凭证（对于API Key来说只是更新时间戳）
	err = manager.Refresh(context.Background())
	if err != nil {
		fmt.Printf("刷新失败: %v\n", err)
		return
	}

	fmt.Printf("凭证已刷新\n")

	// 获取凭证信息
	creds := manager.ToCredentials()
	fmt.Printf("API Key: %s...\n", creds.APIKey[:10])

	_ = authenticator // 使用认证器进行HTTP请求认证

	// Output:
	// 凭证有效: true
	// API Key已更新
	// 新的凭证有效: true
	// 凭证已刷新
	// API Key: new-api-ke...
}

// ExampleAuthManager 演示统一认证管理器的使用
func ExampleAuthManager() {
	// 使用构建器创建API Key认证管理器
	manager, err := auth.NewAuthManagerBuilder().
		WithAPIKey("your-api-key-here-1234567890").
		WithAutoRefresh(false, 0). // API Key不需要自动刷新
		Build()

	if err != nil {
		fmt.Printf("创建认证管理器失败: %v\n", err)
		return
	}
	defer manager.Close()

	// 检查认证类型
	fmt.Printf("认证类型: %s\n", manager.GetType())

	// 检查凭证是否有效
	fmt.Printf("凭证有效: %t\n", manager.IsValid())

	// 创建HTTP请求并添加认证
	req, _ := http.NewRequest("POST", "https://api.example.com/tasks", nil)
	err = manager.Authenticate(req)
	if err != nil {
		fmt.Printf("认证失败: %v\n", err)
		return
	}

	fmt.Printf("HTTP请求已认证\n")

	// 更新API Key
	err = manager.UpdateAPIKey("new-api-key-here-0987654321")
	if err != nil {
		fmt.Printf("更新API Key失败: %v\n", err)
		return
	}

	fmt.Printf("API Key已更新\n")

	// Output:
	// 认证类型: api_key
	// 凭证有效: true
	// HTTP请求已认证
	// API Key已更新
}

// ExampleJWTAuthManager 演示JWT认证管理器的使用
func ExampleJWTAuthManager() {
	// 注意：这里使用示例JWT令牌，实际使用时需要有效的JWT
	jwtConfig := &auth.JWTManagerConfig{
		AccessToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example", // 示例token
		RefreshToken: "refresh-token-example",
		Secret:       "your-secret-key",
		Issuer:       "task-center",
		Audience:     "task-center-sdk",
		RefreshURL:   "https://api.example.com/auth/refresh",
	}

	manager, err := auth.NewJWTAuthManager(jwtConfig)
	if err != nil {
		fmt.Printf("创建JWT管理器失败: %v\n", err)
		return
	}

	// 检查认证类型
	fmt.Printf("认证类型: %s\n", manager.GetType())

	// 检查是否需要刷新（示例token会失败，但演示了API用法）
	needsRefresh := manager.NeedsRefresh()
	fmt.Printf("需要刷新: %t\n", needsRefresh)

	// 获取凭证信息
	creds := manager.GetCredentials()
	if jwtCreds, ok := creds.(*auth.JWTCredentials); ok {
		fmt.Printf("访问令牌: %s...\n", jwtCreds.AccessToken[:20])
	}

	// Output:
	// 认证类型: jwt
	// 需要刷新: false
	// 访问令牌: eyJhbGciOiJIUzI1NiIs...
}