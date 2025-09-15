package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// AuthType 认证类型
type AuthType string

const (
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeJWT    AuthType = "jwt"
)

// Authenticator 认证器接口
type Authenticator interface {
	// Authenticate 为HTTP请求添加认证信息
	Authenticate(req *http.Request) error
	// GetType 获取认证类型
	GetType() AuthType
}

// CredentialManager 凭证管理器接口
type CredentialManager interface {
	// Refresh 刷新凭证
	Refresh(ctx context.Context) error
	// IsValid 检查凭证是否有效
	IsValid() bool
	// GetCreatedAt 获取创建时间
	GetCreatedAt() time.Time
	// GetUpdatedAt 获取更新时间
	GetUpdatedAt() time.Time
}

// AuthManager 认证管理器统一接口
type AuthManager struct {
	authType           AuthType
	apiKeyManager      *APIKeyManager
	jwtManager         *JWTManager
	autoRefresh        bool
	refreshInterval    time.Duration
	refreshCtx         context.Context
	refreshCancel      context.CancelFunc
	onRefreshSuccess   func()
	onRefreshError     func(error)
}

// AuthManagerConfig 认证管理器配置
type AuthManagerConfig struct {
	AuthType        AuthType
	APIKey          string
	JWTConfig       *JWTManagerConfig
	AutoRefresh     bool
	RefreshInterval time.Duration
	OnRefreshSuccess func()
	OnRefreshError   func(error)
}

// NewAuthManager 创建认证管理器
func NewAuthManager(config *AuthManagerConfig) (*AuthManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	manager := &AuthManager{
		authType:         config.AuthType,
		autoRefresh:      config.AutoRefresh,
		refreshInterval:  config.RefreshInterval,
		onRefreshSuccess: config.OnRefreshSuccess,
		onRefreshError:   config.OnRefreshError,
	}

	// 设置默认刷新间隔
	if manager.refreshInterval == 0 {
		manager.refreshInterval = 30 * time.Minute
	}

	// 根据认证类型初始化对应的管理器
	switch config.AuthType {
	case AuthTypeAPIKey:
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for API key authentication")
		}
		apiKeyManager, err := NewAPIKeyManager(config.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create API key manager: %w", err)
		}
		manager.apiKeyManager = apiKeyManager

	case AuthTypeJWT:
		if config.JWTConfig == nil {
			return nil, fmt.Errorf("JWT config is required for JWT authentication")
		}
		jwtManager, err := NewJWTManager(config.JWTConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create JWT manager: %w", err)
		}
		manager.jwtManager = jwtManager

	default:
		return nil, fmt.Errorf("unsupported authentication type: %s", config.AuthType)
	}

	// 启动自动刷新
	if manager.autoRefresh {
		manager.startAutoRefresh()
	}

	return manager, nil
}

// NewAPIKeyAuthManager 创建API Key认证管理器
func NewAPIKeyAuthManager(apiKey string) (*AuthManager, error) {
	return NewAuthManager(&AuthManagerConfig{
		AuthType: AuthTypeAPIKey,
		APIKey:   apiKey,
	})
}

// NewJWTAuthManager 创建JWT认证管理器
func NewJWTAuthManager(jwtConfig *JWTManagerConfig) (*AuthManager, error) {
	return NewAuthManager(&AuthManagerConfig{
		AuthType:  AuthTypeJWT,
		JWTConfig: jwtConfig,
	})
}

// GetAuthenticator 获取认证器
func (m *AuthManager) GetAuthenticator() Authenticator {
	switch m.authType {
	case AuthTypeAPIKey:
		return &apiKeyAuthenticatorWrapper{
			auth: m.apiKeyManager.GetAuth(),
		}
	case AuthTypeJWT:
		return &jwtAuthenticatorWrapper{
			auth: m.jwtManager.GetAuth(),
		}
	default:
		return nil
	}
}

// Authenticate 为HTTP请求添加认证信息
func (m *AuthManager) Authenticate(req *http.Request) error {
	authenticator := m.GetAuthenticator()
	if authenticator == nil {
		return fmt.Errorf("no authenticator available")
	}
	return authenticator.Authenticate(req)
}

// GetType 获取认证类型
func (m *AuthManager) GetType() AuthType {
	return m.authType
}

// IsValid 检查凭证是否有效
func (m *AuthManager) IsValid() bool {
	switch m.authType {
	case AuthTypeAPIKey:
		return m.apiKeyManager != nil && m.apiKeyManager.IsValid()
	case AuthTypeJWT:
		return m.jwtManager != nil && m.jwtManager.IsAccessTokenValid()
	default:
		return false
	}
}

// NeedsRefresh 检查是否需要刷新凭证
func (m *AuthManager) NeedsRefresh() bool {
	switch m.authType {
	case AuthTypeAPIKey:
		// API Key通常不需要刷新
		return false
	case AuthTypeJWT:
		return m.jwtManager != nil && m.jwtManager.NeedsRefresh()
	default:
		return false
	}
}

// Refresh 手动刷新凭证
func (m *AuthManager) Refresh(ctx context.Context) error {
	switch m.authType {
	case AuthTypeAPIKey:
		if m.apiKeyManager == nil {
			return fmt.Errorf("API key manager not initialized")
		}
		return m.apiKeyManager.Refresh(ctx)
	case AuthTypeJWT:
		if m.jwtManager == nil {
			return fmt.Errorf("JWT manager not initialized")
		}
		return m.jwtManager.Refresh(ctx)
	default:
		return fmt.Errorf("unsupported authentication type: %s", m.authType)
	}
}

// UpdateAPIKey 更新API Key
func (m *AuthManager) UpdateAPIKey(apiKey string) error {
	if m.authType != AuthTypeAPIKey {
		return fmt.Errorf("not an API key authentication manager")
	}
	if m.apiKeyManager == nil {
		return fmt.Errorf("API key manager not initialized")
	}
	return m.apiKeyManager.SetAPIKey(apiKey)
}

// UpdateJWTTokens 更新JWT令牌
func (m *AuthManager) UpdateJWTTokens(accessToken, refreshToken string) error {
	if m.authType != AuthTypeJWT {
		return fmt.Errorf("not a JWT authentication manager")
	}
	if m.jwtManager == nil {
		return fmt.Errorf("JWT manager not initialized")
	}
	return m.jwtManager.UpdateTokens(accessToken, refreshToken)
}

// GetCredentials 获取凭证信息
func (m *AuthManager) GetCredentials() interface{} {
	switch m.authType {
	case AuthTypeAPIKey:
		if m.apiKeyManager == nil {
			return nil
		}
		return m.apiKeyManager.ToCredentials()
	case AuthTypeJWT:
		if m.jwtManager == nil {
			return nil
		}
		return m.jwtManager.ToCredentials()
	default:
		return nil
	}
}

// GetCreatedAt 获取创建时间
func (m *AuthManager) GetCreatedAt() time.Time {
	switch m.authType {
	case AuthTypeAPIKey:
		if m.apiKeyManager == nil {
			return time.Time{}
		}
		return m.apiKeyManager.GetCreatedAt()
	case AuthTypeJWT:
		if m.jwtManager == nil {
			return time.Time{}
		}
		return m.jwtManager.GetCreatedAt()
	default:
		return time.Time{}
	}
}

// GetUpdatedAt 获取更新时间
func (m *AuthManager) GetUpdatedAt() time.Time {
	switch m.authType {
	case AuthTypeAPIKey:
		if m.apiKeyManager == nil {
			return time.Time{}
		}
		return m.apiKeyManager.GetUpdatedAt()
	case AuthTypeJWT:
		if m.jwtManager == nil {
			return time.Time{}
		}
		return m.jwtManager.GetUpdatedAt()
	default:
		return time.Time{}
	}
}

// startAutoRefresh 启动自动刷新
func (m *AuthManager) startAutoRefresh() {
	m.refreshCtx, m.refreshCancel = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(m.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.refreshCtx.Done():
				return
			case <-ticker.C:
				if m.NeedsRefresh() {
					if err := m.Refresh(m.refreshCtx); err != nil {
						if m.onRefreshError != nil {
							m.onRefreshError(err)
						}
					} else {
						if m.onRefreshSuccess != nil {
							m.onRefreshSuccess()
						}
					}
				}
			}
		}
	}()
}

// StopAutoRefresh 停止自动刷新
func (m *AuthManager) StopAutoRefresh() {
	if m.refreshCancel != nil {
		m.refreshCancel()
	}
}

// Close 关闭认证管理器
func (m *AuthManager) Close() error {
	m.StopAutoRefresh()
	return nil
}

// apiKeyAuthenticatorWrapper API Key认证器包装器
type apiKeyAuthenticatorWrapper struct {
	auth *APIKeyAuth
}

func (w *apiKeyAuthenticatorWrapper) Authenticate(req *http.Request) error {
	return w.auth.Authenticate(req)
}

func (w *apiKeyAuthenticatorWrapper) GetType() AuthType {
	return AuthTypeAPIKey
}

// jwtAuthenticatorWrapper JWT认证器包装器
type jwtAuthenticatorWrapper struct {
	auth *JWTAuth
}

func (w *jwtAuthenticatorWrapper) Authenticate(req *http.Request) error {
	return w.auth.Authenticate(req)
}

func (w *jwtAuthenticatorWrapper) GetType() AuthType {
	return AuthTypeJWT
}

// AuthManagerBuilder 认证管理器构建器
type AuthManagerBuilder struct {
	config *AuthManagerConfig
}

// NewAuthManagerBuilder 创建认证管理器构建器
func NewAuthManagerBuilder() *AuthManagerBuilder {
	return &AuthManagerBuilder{
		config: &AuthManagerConfig{},
	}
}

// WithAPIKey 设置API Key认证
func (b *AuthManagerBuilder) WithAPIKey(apiKey string) *AuthManagerBuilder {
	b.config.AuthType = AuthTypeAPIKey
	b.config.APIKey = apiKey
	return b
}

// WithJWT 设置JWT认证
func (b *AuthManagerBuilder) WithJWT(jwtConfig *JWTManagerConfig) *AuthManagerBuilder {
	b.config.AuthType = AuthTypeJWT
	b.config.JWTConfig = jwtConfig
	return b
}

// WithAutoRefresh 启用自动刷新
func (b *AuthManagerBuilder) WithAutoRefresh(enable bool, interval time.Duration) *AuthManagerBuilder {
	b.config.AutoRefresh = enable
	b.config.RefreshInterval = interval
	return b
}

// WithRefreshCallbacks 设置刷新回调
func (b *AuthManagerBuilder) WithRefreshCallbacks(onSuccess func(), onError func(error)) *AuthManagerBuilder {
	b.config.OnRefreshSuccess = onSuccess
	b.config.OnRefreshError = onError
	return b
}

// Build 构建认证管理器
func (b *AuthManagerBuilder) Build() (*AuthManager, error) {
	return NewAuthManager(b.config)
}