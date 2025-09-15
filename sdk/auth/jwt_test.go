package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// generateTestJWT 生成测试用的JWT令牌
func generateTestJWT(payload *JWTPayload, secret string) (string, error) {
	// 头部
	header := JWTHeader{
		Alg: "HS256",
		Typ: "JWT",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)

	// 载荷
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// 签名
	message := headerEncoded + "." + payloadEncoded
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return headerEncoded + "." + payloadEncoded + "." + signature, nil
}

func TestNewJWTAuth(t *testing.T) {
	accessToken := "access.token.here"
	refreshToken := "refresh.token.here"
	secret := "test-secret"

	auth := NewJWTAuth(accessToken, refreshToken, secret)

	if auth.GetAccessToken() != accessToken {
		t.Errorf("NewJWTAuth() access token = %v, want %v", auth.GetAccessToken(), accessToken)
	}

	if auth.GetRefreshToken() != refreshToken {
		t.Errorf("NewJWTAuth() refresh token = %v, want %v", auth.GetRefreshToken(), refreshToken)
	}
}

func TestNewJWTAuthWithConfig(t *testing.T) {
	accessToken := "access.token.here"
	refreshToken := "refresh.token.here"
	secret := "test-secret"
	issuer := "test-issuer"
	audience := "test-audience"

	auth := NewJWTAuthWithConfig(accessToken, refreshToken, secret, issuer, audience)

	if auth.issuer != issuer {
		t.Errorf("NewJWTAuthWithConfig() issuer = %v, want %v", auth.issuer, issuer)
	}

	if auth.audience != audience {
		t.Errorf("NewJWTAuthWithConfig() audience = %v, want %v", auth.audience, audience)
	}
}

func TestJWTAuth_Authenticate(t *testing.T) {
	tests := []struct {
		name        string
		accessToken string
		wantErr     bool
	}{
		{
			name:        "valid access token",
			accessToken: "access.token.here",
			wantErr:     false,
		},
		{
			name:        "empty access token",
			accessToken: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewJWTAuth(tt.accessToken, "refresh.token", "secret")
			req, _ := http.NewRequest("GET", "http://example.com", nil)

			err := auth.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("JWTAuth.Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expectedAuth := "Bearer " + tt.accessToken
				if req.Header.Get("Authorization") != expectedAuth {
					t.Errorf("JWTAuth.Authenticate() Authorization header = %v, want %v",
						req.Header.Get("Authorization"), expectedAuth)
				}
			}
		})
	}
}

func TestJWTAuth_ValidateToken(t *testing.T) {
	secret := "test-secret"
	auth := NewJWTAuth("", "", secret)

	now := time.Now()
	validPayload := &JWTPayload{
		Iss:        "task-center",
		Sub:        "user123",
		Aud:        "task-center-sdk",
		Exp:        now.Add(1 * time.Hour).Unix(),
		Iat:        now.Unix(),
		Nbf:        now.Unix(),
		BusinessID: 123,
	}

	expiredPayload := &JWTPayload{
		Iss:        "task-center",
		Sub:        "user123",
		Aud:        "task-center-sdk",
		Exp:        now.Add(-1 * time.Hour).Unix(), // 过期
		Iat:        now.Add(-2 * time.Hour).Unix(),
		Nbf:        now.Add(-2 * time.Hour).Unix(),
		BusinessID: 123,
	}

	futurePayload := &JWTPayload{
		Iss:        "task-center",
		Sub:        "user123",
		Aud:        "task-center-sdk",
		Exp:        now.Add(2 * time.Hour).Unix(),
		Iat:        now.Unix(),
		Nbf:        now.Add(1 * time.Hour).Unix(), // 还未生效
		BusinessID: 123,
	}

	validToken, _ := generateTestJWT(validPayload, secret)
	expiredToken, _ := generateTestJWT(expiredPayload, secret)
	futureToken, _ := generateTestJWT(futurePayload, secret)
	invalidToken := "invalid.token.format"

	tests := []struct {
		name    string
		token   string
		wantErr bool
		wantSub string
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
			wantSub: "user123",
		},
		{
			name:    "expired token",
			token:   expiredToken,
			wantErr: true,
		},
		{
			name:    "future token",
			token:   futureToken,
			wantErr: true,
		},
		{
			name:    "invalid token format",
			token:   invalidToken,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := auth.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("JWTAuth.ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if payload == nil {
					t.Error("JWTAuth.ValidateToken() should return payload for valid token")
					return
				}

				if payload.Sub != tt.wantSub {
					t.Errorf("JWTAuth.ValidateToken() subject = %v, want %v", payload.Sub, tt.wantSub)
				}
			}
		})
	}
}

func TestJWTAuth_IsTokenExpired(t *testing.T) {
	secret := "test-secret"
	now := time.Now()

	validPayload := &JWTPayload{
		Iss: "task-center",
		Aud: "task-center-sdk",
		Exp: now.Add(1 * time.Hour).Unix(),
		Iat: now.Unix(),
	}

	expiredPayload := &JWTPayload{
		Iss: "task-center",
		Aud: "task-center-sdk",
		Exp: now.Add(-1 * time.Hour).Unix(),
		Iat: now.Add(-2 * time.Hour).Unix(),
	}

	validToken, _ := generateTestJWT(validPayload, secret)
	expiredToken, _ := generateTestJWT(expiredPayload, secret)

	tests := []struct {
		name        string
		accessToken string
		wantExpired bool
		wantErr     bool
	}{
		{
			name:        "valid token",
			accessToken: validToken,
			wantExpired: false,
			wantErr:     false,
		},
		{
			name:        "expired token",
			accessToken: expiredToken,
			wantExpired: true,
			wantErr:     true,
		},
		{
			name:        "empty token",
			accessToken: "",
			wantExpired: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewJWTAuth(tt.accessToken, "", secret)
			expired, err := auth.IsTokenExpired()
			if (err != nil) != tt.wantErr {
				t.Errorf("JWTAuth.IsTokenExpired() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if expired != tt.wantExpired {
				t.Errorf("JWTAuth.IsTokenExpired() expired = %v, want %v", expired, tt.wantExpired)
			}
		})
	}
}

func TestJWTAuth_UpdateTokens(t *testing.T) {
	auth := NewJWTAuth("old-access", "old-refresh", "secret")

	newAccess := "new-access-token"
	newRefresh := "new-refresh-token"

	auth.UpdateTokens(newAccess, newRefresh)

	if auth.GetAccessToken() != newAccess {
		t.Errorf("JWTAuth.UpdateTokens() access token = %v, want %v",
			auth.GetAccessToken(), newAccess)
	}

	if auth.GetRefreshToken() != newRefresh {
		t.Errorf("JWTAuth.UpdateTokens() refresh token = %v, want %v",
			auth.GetRefreshToken(), newRefresh)
	}
}

func TestJWTAuth_Clone(t *testing.T) {
	original := NewJWTAuth("access-token", "refresh-token", "secret")
	cloned := original.Clone()

	if cloned.GetAccessToken() != original.GetAccessToken() {
		t.Errorf("JWTAuth.Clone() access token = %v, want %v",
			cloned.GetAccessToken(), original.GetAccessToken())
	}

	if cloned.GetRefreshToken() != original.GetRefreshToken() {
		t.Errorf("JWTAuth.Clone() refresh token = %v, want %v",
			cloned.GetRefreshToken(), original.GetRefreshToken())
	}

	// 修改原始对象，确保克隆对象不受影响
	original.UpdateTokens("modified-access", "modified-refresh")
	if cloned.GetAccessToken() == original.GetAccessToken() {
		t.Error("JWTAuth.Clone() should create independent copy")
	}
}

func TestNewJWTManager(t *testing.T) {
	secret := "test-secret"
	now := time.Now()
	validPayload := &JWTPayload{
		Exp: now.Add(1 * time.Hour).Unix(),
		Iat: now.Unix(),
	}
	validToken, _ := generateTestJWT(validPayload, secret)

	tests := []struct {
		name    string
		config  *JWTManagerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &JWTManagerConfig{
				AccessToken: validToken,
				Secret:      secret,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty access token",
			config: &JWTManagerConfig{
				AccessToken: "",
				Secret:      secret,
			},
			wantErr: true,
		},
		{
			name: "empty secret",
			config: &JWTManagerConfig{
				AccessToken: validToken,
				Secret:      "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJWTManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if manager == nil {
					t.Error("NewJWTManager() should not return nil manager")
					return
				}

				if time.Since(manager.GetCreatedAt()) > time.Second {
					t.Error("NewJWTManager() created time should be recent")
				}
			}
		})
	}
}

func TestJWTManager_Refresh(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req["refresh_token"] != "valid-refresh-token" {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		response := map[string]string{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	secret := "test-secret"
	now := time.Now()
	validPayload := &JWTPayload{
		Exp: now.Add(1 * time.Hour).Unix(),
		Iat: now.Unix(),
	}
	validToken, _ := generateTestJWT(validPayload, secret)

	config := &JWTManagerConfig{
		AccessToken:  validToken,
		RefreshToken: "valid-refresh-token",
		Secret:       secret,
		RefreshURL:   server.URL,
	}

	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	ctx := context.Background()
	err = manager.Refresh(ctx)
	if err != nil {
		t.Errorf("JWTManager.Refresh() error = %v", err)
		return
	}

	if manager.accessToken != "new-access-token" {
		t.Errorf("JWTManager.Refresh() access token = %v, want %v",
			manager.accessToken, "new-access-token")
	}

	if manager.refreshToken != "new-refresh-token" {
		t.Errorf("JWTManager.Refresh() refresh token = %v, want %v",
			manager.refreshToken, "new-refresh-token")
	}
}

func TestJWTManager_IsAccessTokenValid(t *testing.T) {
	secret := "test-secret"
	now := time.Now()

	validPayload := &JWTPayload{
		Iss: "task-center",
		Aud: "task-center-sdk",
		Exp: now.Add(1 * time.Hour).Unix(),
		Iat: now.Unix(),
	}

	expiredPayload := &JWTPayload{
		Iss: "task-center",
		Aud: "task-center-sdk",
		Exp: now.Add(-1 * time.Hour).Unix(),
		Iat: now.Add(-2 * time.Hour).Unix(),
	}

	validToken, _ := generateTestJWT(validPayload, secret)
	expiredToken, _ := generateTestJWT(expiredPayload, secret)

	tests := []struct {
		name        string
		accessToken string
		want        bool
	}{
		{
			name:        "valid token",
			accessToken: validToken,
			want:        true,
		},
		{
			name:        "expired token",
			accessToken: expiredToken,
			want:        false,
		},
		{
			name:        "empty token",
			accessToken: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &JWTManagerConfig{
				AccessToken: tt.accessToken,
				Secret:      secret,
			}

			manager, err := NewJWTManager(config)
			if err != nil && tt.accessToken != "" {
				t.Fatalf("NewJWTManager() error = %v", err)
			}

			if tt.accessToken == "" {
				// 对于空token，创建一个空的manager进行测试
				manager = &JWTManager{
					accessToken: "",
					secret:      secret,
				}
			}

			if got := manager.IsAccessTokenValid(); got != tt.want {
				t.Errorf("JWTManager.IsAccessTokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJWTManager_ToCredentials(t *testing.T) {
	secret := "test-secret"
	now := time.Now()
	validPayload := &JWTPayload{
		Iss: "test-issuer",
		Aud: "test-audience",
		Exp: now.Add(1 * time.Hour).Unix(),
		Iat: now.Unix(),
	}
	validToken, _ := generateTestJWT(validPayload, secret)

	config := &JWTManagerConfig{
		AccessToken:  validToken,
		RefreshToken: "refresh-token",
		Secret:       secret,
		Issuer:       "test-issuer",
		Audience:     "test-audience",
	}

	manager, err := NewJWTManager(config)
	if err != nil {
		t.Fatalf("NewJWTManager() error = %v", err)
	}

	creds := manager.ToCredentials()
	if creds == nil {
		t.Fatal("JWTManager.ToCredentials() should not return nil")
	}

	if creds.AccessToken != validToken {
		t.Errorf("JWTManager.ToCredentials() access token = %v, want %v",
			creds.AccessToken, validToken)
	}

	if creds.RefreshToken != "refresh-token" {
		t.Errorf("JWTManager.ToCredentials() refresh token = %v, want %v",
			creds.RefreshToken, "refresh-token")
	}

	if creds.Secret != secret {
		t.Errorf("JWTManager.ToCredentials() secret = %v, want %v",
			creds.Secret, secret)
	}

	if creds.AccessTokenExpiry == nil {
		t.Error("JWTManager.ToCredentials() access token expiry should not be nil")
	}
}

func TestJWTManager_FromCredentials(t *testing.T) {
	secret := "test-secret"
	now := time.Now()
	validPayload := &JWTPayload{
		Iss: "restored-issuer",
		Aud: "restored-audience",
		Exp: now.Add(1 * time.Hour).Unix(),
		Iat: now.Unix(),
	}
	validToken, _ := generateTestJWT(validPayload, secret)

	manager, _ := NewJWTManager(&JWTManagerConfig{
		AccessToken: "initial-token",
		Secret:      secret,
	})

	expiry := time.Now().Add(1 * time.Hour)
	creds := &JWTCredentials{
		AccessToken:       validToken,
		RefreshToken:      "restored-refresh-token",
		Secret:            secret,
		Issuer:            "restored-issuer",
		Audience:          "restored-audience",
		CreatedAt:         time.Now().Add(-1 * time.Hour),
		UpdatedAt:         time.Now().Add(-30 * time.Minute),
		AccessTokenExpiry: &expiry,
	}

	err := manager.FromCredentials(creds)
	if err != nil {
		t.Errorf("JWTManager.FromCredentials() error = %v", err)
		return
	}

	if manager.accessToken != creds.AccessToken {
		t.Errorf("JWTManager.FromCredentials() access token = %v, want %v",
			manager.accessToken, creds.AccessToken)
	}

	if manager.refreshToken != creds.RefreshToken {
		t.Errorf("JWTManager.FromCredentials() refresh token = %v, want %v",
			manager.refreshToken, creds.RefreshToken)
	}

	// 测试无效凭证
	err = manager.FromCredentials(nil)
	if err == nil {
		t.Error("JWTManager.FromCredentials() should return error for nil credentials")
	}

	invalidCreds := &JWTCredentials{
		AccessToken: "", // 空的access token
		Secret:      secret,
	}

	err = manager.FromCredentials(invalidCreds)
	if err == nil {
		t.Error("JWTManager.FromCredentials() should return error for empty access token")
	}
}