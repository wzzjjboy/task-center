package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// JWTHeader JWT头部结构
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// JWTPayload JWT载荷结构
type JWTPayload struct {
	Iss       string `json:"iss"`       // 签发者
	Sub       string `json:"sub"`       // 主题（通常是用户ID）
	Aud       string `json:"aud"`       // 受众
	Exp       int64  `json:"exp"`       // 过期时间
	Iat       int64  `json:"iat"`       // 签发时间
	Nbf       int64  `json:"nbf"`       // 生效时间
	Jti       string `json:"jti"`       // JWT唯一标识
	BusinessID int64 `json:"business_id"` // 业务系统ID
}

// JWTAuth JWT认证器
type JWTAuth struct {
	accessToken  string
	refreshToken string
	secret       string
	issuer       string
	audience     string
}

// NewJWTAuth 创建JWT认证器
func NewJWTAuth(accessToken, refreshToken, secret string) *JWTAuth {
	return &JWTAuth{
		accessToken:  strings.TrimSpace(accessToken),
		refreshToken: strings.TrimSpace(refreshToken),
		secret:       secret,
		issuer:       "task-center",
		audience:     "task-center-sdk",
	}
}

// NewJWTAuthWithConfig 创建带配置的JWT认证器
func NewJWTAuthWithConfig(accessToken, refreshToken, secret, issuer, audience string) *JWTAuth {
	return &JWTAuth{
		accessToken:  strings.TrimSpace(accessToken),
		refreshToken: strings.TrimSpace(refreshToken),
		secret:       secret,
		issuer:       issuer,
		audience:     audience,
	}
}

// Authenticate 为HTTP请求添加JWT认证头
func (j *JWTAuth) Authenticate(req *http.Request) error {
	if j.accessToken == "" {
		return fmt.Errorf("access token is empty")
	}

	req.Header.Set("Authorization", "Bearer "+j.accessToken)
	return nil
}

// ValidateToken 验证JWT令牌
func (j *JWTAuth) ValidateToken(token string) (*JWTPayload, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	// 分割JWT令牌
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// 解码头部
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header JWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// 检查算法
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// 解码载荷
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// 验证签名
	expectedSignature := j.generateSignature(parts[0] + "." + parts[1])
	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	if !hmac.Equal(expectedSignature, actualSignature) {
		return nil, fmt.Errorf("invalid signature")
	}

	// 验证时间
	now := time.Now().Unix()
	if payload.Exp > 0 && now >= payload.Exp {
		return nil, fmt.Errorf("token has expired")
	}

	if payload.Nbf > 0 && now < payload.Nbf {
		return nil, fmt.Errorf("token not yet valid")
	}

	// 验证签发者和受众
	if j.issuer != "" && payload.Iss != j.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	if j.audience != "" && payload.Aud != j.audience {
		return nil, fmt.Errorf("invalid audience")
	}

	return &payload, nil
}

// IsTokenExpired 检查令牌是否过期
func (j *JWTAuth) IsTokenExpired() (bool, error) {
	if j.accessToken == "" {
		return true, fmt.Errorf("access token is empty")
	}

	payload, err := j.ValidateToken(j.accessToken)
	if err != nil {
		return true, err
	}

	now := time.Now().Unix()
	return payload.Exp > 0 && now >= payload.Exp, nil
}

// GetAccessToken 获取访问令牌
func (j *JWTAuth) GetAccessToken() string {
	return j.accessToken
}

// GetRefreshToken 获取刷新令牌
func (j *JWTAuth) GetRefreshToken() string {
	return j.refreshToken
}

// UpdateTokens 更新令牌
func (j *JWTAuth) UpdateTokens(accessToken, refreshToken string) {
	j.accessToken = strings.TrimSpace(accessToken)
	j.refreshToken = strings.TrimSpace(refreshToken)
}

// generateSignature 生成HMAC-SHA256签名
func (j *JWTAuth) generateSignature(data string) []byte {
	h := hmac.New(sha256.New, []byte(j.secret))
	h.Write([]byte(data))
	return h.Sum(nil)
}

// Clone 克隆认证器
func (j *JWTAuth) Clone() *JWTAuth {
	return &JWTAuth{
		accessToken:  j.accessToken,
		refreshToken: j.refreshToken,
		secret:       j.secret,
		issuer:       j.issuer,
		audience:     j.audience,
	}
}

// JWTManager JWT令牌管理器
type JWTManager struct {
	accessToken       string
	refreshToken      string
	secret            string
	issuer            string
	audience          string
	createdAt         time.Time
	updatedAt         time.Time
	accessTokenExpiry *time.Time
	refreshTokenExpiry *time.Time
	httpClient        *http.Client
	refreshURL        string
}

// JWTManagerConfig JWT管理器配置
type JWTManagerConfig struct {
	AccessToken        string
	RefreshToken       string
	Secret             string
	Issuer             string
	Audience           string
	RefreshURL         string
	HTTPClient         *http.Client
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(config *JWTManagerConfig) (*JWTManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	if config.Secret == "" {
		return nil, fmt.Errorf("secret is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	manager := &JWTManager{
		accessToken:  strings.TrimSpace(config.AccessToken),
		refreshToken: strings.TrimSpace(config.RefreshToken),
		secret:       config.Secret,
		issuer:       config.Issuer,
		audience:     config.Audience,
		refreshURL:   config.RefreshURL,
		httpClient:   httpClient,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	// 解析令牌过期时间
	if err := manager.parseTokenExpiry(); err != nil {
		return nil, fmt.Errorf("failed to parse token expiry: %w", err)
	}

	return manager, nil
}

// parseTokenExpiry 解析令牌过期时间
func (m *JWTManager) parseTokenExpiry() error {
	auth := m.GetAuth()

	// 解析访问令牌过期时间
	if m.accessToken != "" {
		if payload, err := auth.ValidateToken(m.accessToken); err == nil && payload.Exp > 0 {
			expiry := time.Unix(payload.Exp, 0)
			m.accessTokenExpiry = &expiry
		}
	}

	// 解析刷新令牌过期时间
	if m.refreshToken != "" {
		if payload, err := auth.ValidateToken(m.refreshToken); err == nil && payload.Exp > 0 {
			expiry := time.Unix(payload.Exp, 0)
			m.refreshTokenExpiry = &expiry
		}
	}

	return nil
}

// GetAuth 获取认证器实例
func (m *JWTManager) GetAuth() *JWTAuth {
	return NewJWTAuthWithConfig(m.accessToken, m.refreshToken, m.secret, m.issuer, m.audience)
}

// IsAccessTokenValid 检查访问令牌是否有效
func (m *JWTManager) IsAccessTokenValid() bool {
	if m.accessToken == "" {
		return false
	}

	auth := m.GetAuth()
	_, err := auth.ValidateToken(m.accessToken)
	return err == nil
}

// IsAccessTokenExpired 检查访问令牌是否过期
func (m *JWTManager) IsAccessTokenExpired() bool {
	if m.accessTokenExpiry == nil {
		return false
	}
	return time.Now().After(*m.accessTokenExpiry)
}

// NeedsRefresh 检查是否需要刷新令牌（在过期前5分钟刷新）
func (m *JWTManager) NeedsRefresh() bool {
	if m.accessTokenExpiry == nil {
		return false
	}

	refreshTime := m.accessTokenExpiry.Add(-5 * time.Minute)
	return time.Now().After(refreshTime)
}

// Refresh 刷新JWT令牌
func (m *JWTManager) Refresh(ctx context.Context) error {
	if m.refreshToken == "" {
		return fmt.Errorf("refresh token is empty")
	}

	if m.refreshURL == "" {
		return fmt.Errorf("refresh URL is not configured")
	}

	// 准备刷新请求
	refreshRequest := map[string]string{
		"refresh_token": m.refreshToken,
	}

	reqBody, err := json.Marshal(refreshRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.refreshURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 执行刷新请求
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh request failed with status: %d", resp.StatusCode)
	}

	// 解析响应
	var refreshResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&refreshResponse); err != nil {
		return fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// 更新令牌
	m.accessToken = refreshResponse.AccessToken
	if refreshResponse.RefreshToken != "" {
		m.refreshToken = refreshResponse.RefreshToken
	}
	m.updatedAt = time.Now()

	// 重新解析过期时间
	return m.parseTokenExpiry()
}

// UpdateTokens 手动更新令牌
func (m *JWTManager) UpdateTokens(accessToken, refreshToken string) error {
	if accessToken == "" {
		return fmt.Errorf("access token cannot be empty")
	}

	m.accessToken = strings.TrimSpace(accessToken)
	if refreshToken != "" {
		m.refreshToken = strings.TrimSpace(refreshToken)
	}
	m.updatedAt = time.Now()

	return m.parseTokenExpiry()
}

// GetCreatedAt 获取创建时间
func (m *JWTManager) GetCreatedAt() time.Time {
	return m.createdAt
}

// GetUpdatedAt 获取更新时间
func (m *JWTManager) GetUpdatedAt() time.Time {
	return m.updatedAt
}

// GetAccessTokenExpiry 获取访问令牌过期时间
func (m *JWTManager) GetAccessTokenExpiry() *time.Time {
	return m.accessTokenExpiry
}

// GetRefreshTokenExpiry 获取刷新令牌过期时间
func (m *JWTManager) GetRefreshTokenExpiry() *time.Time {
	return m.refreshTokenExpiry
}

// JWTCredentials JWT凭证信息
type JWTCredentials struct {
	AccessToken        string     `json:"access_token"`
	RefreshToken       string     `json:"refresh_token"`
	Secret             string     `json:"secret"`
	Issuer             string     `json:"issuer"`
	Audience           string     `json:"audience"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	AccessTokenExpiry  *time.Time `json:"access_token_expiry,omitempty"`
	RefreshTokenExpiry *time.Time `json:"refresh_token_expiry,omitempty"`
}

// ToCredentials 转换为凭证信息
func (m *JWTManager) ToCredentials() *JWTCredentials {
	return &JWTCredentials{
		AccessToken:        m.accessToken,
		RefreshToken:       m.refreshToken,
		Secret:             m.secret,
		Issuer:             m.issuer,
		Audience:           m.audience,
		CreatedAt:          m.createdAt,
		UpdatedAt:          m.updatedAt,
		AccessTokenExpiry:  m.accessTokenExpiry,
		RefreshTokenExpiry: m.refreshTokenExpiry,
	}
}

// FromCredentials 从凭证信息恢复管理器
func (m *JWTManager) FromCredentials(creds *JWTCredentials) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	if creds.AccessToken == "" {
		return fmt.Errorf("access token cannot be empty")
	}

	m.accessToken = creds.AccessToken
	m.refreshToken = creds.RefreshToken
	m.secret = creds.Secret
	m.issuer = creds.Issuer
	m.audience = creds.Audience
	m.createdAt = creds.CreatedAt
	m.updatedAt = creds.UpdatedAt
	m.accessTokenExpiry = creds.AccessTokenExpiry
	m.refreshTokenExpiry = creds.RefreshTokenExpiry

	return nil
}