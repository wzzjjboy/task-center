package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIKeyAuth API Key认证器
type APIKeyAuth struct {
	apiKey string
}

// NewAPIKeyAuth 创建API Key认证器
func NewAPIKeyAuth(apiKey string) *APIKeyAuth {
	return &APIKeyAuth{
		apiKey: strings.TrimSpace(apiKey),
	}
}

// Authenticate 为HTTP请求添加API Key认证头
func (a *APIKeyAuth) Authenticate(req *http.Request) error {
	if a.apiKey == "" {
		return fmt.Errorf("API key is empty")
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// ValidateAPIKey 验证API Key格式
func (a *APIKeyAuth) ValidateAPIKey() error {
	if a.apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// API Key长度检查（一般至少16个字符）
	if len(a.apiKey) < 16 {
		return fmt.Errorf("API key is too short, minimum 16 characters required")
	}

	// 检查是否包含不合法字符（只允许字母、数字、下划线、短横线）
	for _, char := range a.apiKey {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return fmt.Errorf("API key contains invalid characters")
		}
	}

	return nil
}

// GetAPIKey 获取API Key
func (a *APIKeyAuth) GetAPIKey() string {
	return a.apiKey
}

// UpdateAPIKey 更新API Key
func (a *APIKeyAuth) UpdateAPIKey(newAPIKey string) error {
	newAPIKey = strings.TrimSpace(newAPIKey)

	// 创建临时认证器验证新的API Key
	tempAuth := NewAPIKeyAuth(newAPIKey)
	if err := tempAuth.ValidateAPIKey(); err != nil {
		return fmt.Errorf("invalid new API key: %w", err)
	}

	a.apiKey = newAPIKey
	return nil
}

// Clone 克隆认证器
func (a *APIKeyAuth) Clone() *APIKeyAuth {
	return &APIKeyAuth{
		apiKey: a.apiKey,
	}
}

// APIKeyManager API Key管理器，负责凭证存储和管理
type APIKeyManager struct {
	apiKey    string
	createdAt time.Time
	updatedAt time.Time
}

// NewAPIKeyManager 创建API Key管理器
func NewAPIKeyManager(apiKey string) (*APIKeyManager, error) {
	manager := &APIKeyManager{
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}

	if err := manager.SetAPIKey(apiKey); err != nil {
		return nil, err
	}

	return manager, nil
}

// SetAPIKey 设置API Key
func (m *APIKeyManager) SetAPIKey(apiKey string) error {
	auth := NewAPIKeyAuth(apiKey)
	if err := auth.ValidateAPIKey(); err != nil {
		return err
	}

	m.apiKey = apiKey
	m.updatedAt = time.Now()
	return nil
}

// GetAPIKey 获取API Key
func (m *APIKeyManager) GetAPIKey() string {
	return m.apiKey
}

// GetAuth 获取认证器实例
func (m *APIKeyManager) GetAuth() *APIKeyAuth {
	return NewAPIKeyAuth(m.apiKey)
}

// IsValid 检查API Key是否有效
func (m *APIKeyManager) IsValid() bool {
	auth := NewAPIKeyAuth(m.apiKey)
	return auth.ValidateAPIKey() == nil
}

// GetCreatedAt 获取创建时间
func (m *APIKeyManager) GetCreatedAt() time.Time {
	return m.createdAt
}

// GetUpdatedAt 获取更新时间
func (m *APIKeyManager) GetUpdatedAt() time.Time {
	return m.updatedAt
}

// Refresh 刷新API Key（在此实现中，API Key通常不需要刷新，这个方法主要是为了接口一致性）
func (m *APIKeyManager) Refresh(ctx context.Context) error {
	// API Key通常是静态的，不需要刷新
	// 这里只更新时间戳以表示"刷新"操作
	m.updatedAt = time.Now()
	return nil
}

// APIKeyCredentials API Key凭证信息
type APIKeyCredentials struct {
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // API Key通常不会过期，但预留字段
}

// ToCredentials 转换为凭证信息
func (m *APIKeyManager) ToCredentials() *APIKeyCredentials {
	return &APIKeyCredentials{
		APIKey:    m.apiKey,
		CreatedAt: m.createdAt,
		UpdatedAt: m.updatedAt,
		ExpiresAt: nil, // API Key通常不过期
	}
}

// FromCredentials 从凭证信息恢复管理器
func (m *APIKeyManager) FromCredentials(creds *APIKeyCredentials) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	if err := m.SetAPIKey(creds.APIKey); err != nil {
		return err
	}

	m.createdAt = creds.CreatedAt
	m.updatedAt = creds.UpdatedAt

	return nil
}