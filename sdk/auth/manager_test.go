package auth

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuthManagerConfig
		wantErr bool
	}{
		{
			name: "valid API key config",
			config: &AuthManagerConfig{
				AuthType: AuthTypeAPIKey,
				APIKey:   "test-api-key-12345678",
			},
			wantErr: false,
		},
		{
			name: "valid JWT config",
			config: &AuthManagerConfig{
				AuthType: AuthTypeJWT,
				JWTConfig: &JWTManagerConfig{
					AccessToken: "test.access.token",
					Secret:      "test-secret",
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "API key config with empty key",
			config: &AuthManagerConfig{
				AuthType: AuthTypeAPIKey,
				APIKey:   "",
			},
			wantErr: true,
		},
		{
			name: "JWT config with nil JWT config",
			config: &AuthManagerConfig{
				AuthType:  AuthTypeJWT,
				JWTConfig: nil,
			},
			wantErr: true,
		},
		{
			name: "unsupported auth type",
			config: &AuthManagerConfig{
				AuthType: "unsupported",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuthManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if manager == nil {
					t.Error("NewAuthManager() should not return nil manager")
					return
				}

				if manager.GetType() != tt.config.AuthType {
					t.Errorf("NewAuthManager() auth type = %v, want %v",
						manager.GetType(), tt.config.AuthType)
				}

				// 验证刷新间隔默认值
				if manager.refreshInterval != 30*time.Minute {
					t.Errorf("NewAuthManager() refresh interval = %v, want %v",
						manager.refreshInterval, 30*time.Minute)
				}
			}
		})
	}
}

func TestNewAPIKeyAuthManager(t *testing.T) {
	apiKey := "test-api-key-12345678"
	manager, err := NewAPIKeyAuthManager(apiKey)
	if err != nil {
		t.Errorf("NewAPIKeyAuthManager() error = %v", err)
		return
	}

	if manager.GetType() != AuthTypeAPIKey {
		t.Errorf("NewAPIKeyAuthManager() auth type = %v, want %v",
			manager.GetType(), AuthTypeAPIKey)
	}

	if !manager.IsValid() {
		t.Error("NewAPIKeyAuthManager() should create valid manager")
	}
}

func TestNewJWTAuthManager(t *testing.T) {
	jwtConfig := &JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	}

	manager, err := NewJWTAuthManager(jwtConfig)
	if err != nil {
		t.Errorf("NewJWTAuthManager() error = %v", err)
		return
	}

	if manager.GetType() != AuthTypeJWT {
		t.Errorf("NewJWTAuthManager() auth type = %v, want %v",
			manager.GetType(), AuthTypeJWT)
	}
}

func TestAuthManager_GetAuthenticator(t *testing.T) {
	// 测试API Key认证器
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")
	apiKeyAuth := apiKeyManager.GetAuthenticator()
	if apiKeyAuth == nil {
		t.Fatal("GetAuthenticator() should not return nil for API key manager")
	}

	if apiKeyAuth.GetType() != AuthTypeAPIKey {
		t.Errorf("GetAuthenticator() auth type = %v, want %v",
			apiKeyAuth.GetType(), AuthTypeAPIKey)
	}

	// 测试JWT认证器
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	jwtAuth := jwtManager.GetAuthenticator()
	if jwtAuth == nil {
		t.Fatal("GetAuthenticator() should not return nil for JWT manager")
	}

	if jwtAuth.GetType() != AuthTypeJWT {
		t.Errorf("GetAuthenticator() auth type = %v, want %v",
			jwtAuth.GetType(), AuthTypeJWT)
	}
}

func TestAuthManager_Authenticate(t *testing.T) {
	tests := []struct {
		name    string
		manager *AuthManager
		wantErr bool
	}{
		{
			name: "API key authentication",
			manager: func() *AuthManager {
				m, _ := NewAPIKeyAuthManager("test-api-key-12345678")
				return m
			}(),
			wantErr: false,
		},
		{
			name: "JWT authentication",
			manager: func() *AuthManager {
				m, _ := NewJWTAuthManager(&JWTManagerConfig{
					AccessToken: "test.access.token",
					Secret:      "test-secret",
				})
				return m
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)

			err := tt.manager.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthManager.Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				authHeader := req.Header.Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Errorf("AuthManager.Authenticate() Authorization header = %v, should start with 'Bearer '",
						authHeader)
				}
			}
		})
	}
}

func TestAuthManager_IsValid(t *testing.T) {
	// 测试有效的API Key管理器
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")
	if !apiKeyManager.IsValid() {
		t.Error("API key manager should be valid")
	}

	// 测试JWT管理器（注意：这里的token不是真正的JWT格式，但足够测试逻辑）
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	// JWT管理器的IsValid依赖于token的实际验证，这里的测试token会失败
	if jwtManager.IsValid() {
		t.Error("JWT manager with invalid token should not be valid")
	}
}

func TestAuthManager_NeedsRefresh(t *testing.T) {
	// API Key不需要刷新
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")
	if apiKeyManager.NeedsRefresh() {
		t.Error("API key manager should not need refresh")
	}

	// JWT管理器的NeedsRefresh依赖于token的实际过期时间
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	// 由于token格式无效，NeedsRefresh应该返回false
	if jwtManager.NeedsRefresh() {
		t.Error("JWT manager with invalid token should not need refresh")
	}
}

func TestAuthManager_Refresh(t *testing.T) {
	ctx := context.Background()

	// 测试API Key刷新
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")
	err := apiKeyManager.Refresh(ctx)
	if err != nil {
		t.Errorf("API key manager Refresh() error = %v", err)
	}

	// 测试JWT刷新（会失败，因为没有配置刷新URL）
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	err = jwtManager.Refresh(ctx)
	if err == nil {
		t.Error("JWT manager Refresh() should fail without refresh URL")
	}
}

func TestAuthManager_UpdateAPIKey(t *testing.T) {
	// 测试API Key管理器更新
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")
	newAPIKey := "new-api-key-87654321"

	err := apiKeyManager.UpdateAPIKey(newAPIKey)
	if err != nil {
		t.Errorf("UpdateAPIKey() error = %v", err)
	}

	// 验证更新是否成功
	if !apiKeyManager.IsValid() {
		t.Error("Manager should still be valid after updating API key")
	}

	// 测试JWT管理器更新API Key（应该失败）
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	err = jwtManager.UpdateAPIKey(newAPIKey)
	if err == nil {
		t.Error("JWT manager UpdateAPIKey() should fail")
	}

	// 测试无效的API Key
	err = apiKeyManager.UpdateAPIKey("short")
	if err == nil {
		t.Error("UpdateAPIKey() should fail for invalid API key")
	}
}

func TestAuthManager_UpdateJWTTokens(t *testing.T) {
	// 测试JWT管理器更新
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	newAccessToken := "new.access.token"
	newRefreshToken := "new.refresh.token"

	err := jwtManager.UpdateJWTTokens(newAccessToken, newRefreshToken)
	if err != nil {
		t.Errorf("UpdateJWTTokens() error = %v", err)
	}

	// 测试API Key管理器更新JWT令牌（应该失败）
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")

	err = apiKeyManager.UpdateJWTTokens(newAccessToken, newRefreshToken)
	if err == nil {
		t.Error("API key manager UpdateJWTTokens() should fail")
	}

	// 测试空的access token
	err = jwtManager.UpdateJWTTokens("", newRefreshToken)
	if err == nil {
		t.Error("UpdateJWTTokens() should fail for empty access token")
	}
}

func TestAuthManager_GetCredentials(t *testing.T) {
	// 测试API Key凭证
	apiKeyManager, _ := NewAPIKeyAuthManager("test-api-key-12345678")
	apiCreds := apiKeyManager.GetCredentials()
	if apiCreds == nil {
		t.Error("GetCredentials() should not return nil for API key manager")
	}

	if _, ok := apiCreds.(*APIKeyCredentials); !ok {
		t.Error("GetCredentials() should return APIKeyCredentials for API key manager")
	}

	// 测试JWT凭证
	jwtManager, _ := NewJWTAuthManager(&JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	})

	jwtCreds := jwtManager.GetCredentials()
	if jwtCreds == nil {
		t.Error("GetCredentials() should not return nil for JWT manager")
	}

	if _, ok := jwtCreds.(*JWTCredentials); !ok {
		t.Error("GetCredentials() should return JWTCredentials for JWT manager")
	}
}

func TestAuthManager_GetTimestamps(t *testing.T) {
	manager, _ := NewAPIKeyAuthManager("test-api-key-12345678")

	createdAt := manager.GetCreatedAt()
	if createdAt.IsZero() {
		t.Error("GetCreatedAt() should not return zero time")
	}

	updatedAt := manager.GetUpdatedAt()
	if updatedAt.IsZero() {
		t.Error("GetUpdatedAt() should not return zero time")
	}

	if time.Since(createdAt) > time.Second {
		t.Error("GetCreatedAt() should be recent")
	}

	if time.Since(updatedAt) > time.Second {
		t.Error("GetUpdatedAt() should be recent")
	}
}

func TestAuthManager_AutoRefresh(t *testing.T) {
	// 测试带自动刷新的配置
	config := &AuthManagerConfig{
		AuthType:        AuthTypeAPIKey,
		APIKey:          "test-api-key-12345678",
		AutoRefresh:     true,
		RefreshInterval: 100 * time.Millisecond,
	}

	manager, err := NewAuthManager(config)
	if err != nil {
		t.Errorf("NewAuthManager() error = %v", err)
		return
	}

	// 验证自动刷新已启动
	if manager.refreshCancel == nil {
		t.Error("Auto refresh should be started")
	}

	// 停止自动刷新
	manager.StopAutoRefresh()

	// 验证自动刷新已停止
	select {
	case <-manager.refreshCtx.Done():
		// 正确停止
	case <-time.After(100 * time.Millisecond):
		t.Error("Auto refresh should be stopped")
	}
}

func TestAuthManager_Close(t *testing.T) {
	config := &AuthManagerConfig{
		AuthType:        AuthTypeAPIKey,
		APIKey:          "test-api-key-12345678",
		AutoRefresh:     true,
		RefreshInterval: 100 * time.Millisecond,
	}

	manager, err := NewAuthManager(config)
	if err != nil {
		t.Errorf("NewAuthManager() error = %v", err)
		return
	}

	err = manager.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 验证自动刷新已停止
	select {
	case <-manager.refreshCtx.Done():
		// 正确停止
	case <-time.After(100 * time.Millisecond):
		t.Error("Auto refresh should be stopped after Close()")
	}
}

func TestAuthManagerBuilder(t *testing.T) {
	// 测试API Key构建器
	builder := NewAuthManagerBuilder()
	manager, err := builder.
		WithAPIKey("test-api-key-12345678").
		WithAutoRefresh(true, 5*time.Minute).
		Build()

	if err != nil {
		t.Errorf("AuthManagerBuilder.Build() error = %v", err)
		return
	}

	if manager.GetType() != AuthTypeAPIKey {
		t.Errorf("Builder auth type = %v, want %v", manager.GetType(), AuthTypeAPIKey)
	}

	if !manager.autoRefresh {
		t.Error("Builder should enable auto refresh")
	}

	if manager.refreshInterval != 5*time.Minute {
		t.Errorf("Builder refresh interval = %v, want %v",
			manager.refreshInterval, 5*time.Minute)
	}

	manager.Close()

	// 测试JWT构建器
	jwtConfig := &JWTManagerConfig{
		AccessToken: "test.access.token",
		Secret:      "test-secret",
	}

	manager2, err := NewAuthManagerBuilder().
		WithJWT(jwtConfig).
		WithRefreshCallbacks(
			func() { /* success callback */ },
			func(error) { /* error callback */ },
		).
		Build()

	if err != nil {
		t.Errorf("AuthManagerBuilder.Build() error = %v", err)
		return
	}

	if manager2.GetType() != AuthTypeJWT {
		t.Errorf("Builder auth type = %v, want %v", manager2.GetType(), AuthTypeJWT)
	}

	if manager2.onRefreshSuccess == nil {
		t.Error("Builder should set refresh success callback")
	}

	if manager2.onRefreshError == nil {
		t.Error("Builder should set refresh error callback")
	}
}

func TestAuthenticatorWrappers(t *testing.T) {
	// 测试API Key认证器包装器
	apiAuth := NewAPIKeyAuth("test-api-key-12345678")
	wrapper := &apiKeyAuthenticatorWrapper{auth: apiAuth}

	if wrapper.GetType() != AuthTypeAPIKey {
		t.Errorf("apiKeyAuthenticatorWrapper.GetType() = %v, want %v",
			wrapper.GetType(), AuthTypeAPIKey)
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := wrapper.Authenticate(req)
	if err != nil {
		t.Errorf("apiKeyAuthenticatorWrapper.Authenticate() error = %v", err)
	}

	// 测试JWT认证器包装器
	jwtAuth := NewJWTAuth("access.token", "refresh.token", "secret")
	jwtWrapper := &jwtAuthenticatorWrapper{auth: jwtAuth}

	if jwtWrapper.GetType() != AuthTypeJWT {
		t.Errorf("jwtAuthenticatorWrapper.GetType() = %v, want %v",
			jwtWrapper.GetType(), AuthTypeJWT)
	}

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	err = jwtWrapper.Authenticate(req2)
	if err != nil {
		t.Errorf("jwtAuthenticatorWrapper.Authenticate() error = %v", err)
	}
}