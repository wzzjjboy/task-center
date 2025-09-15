package auth

import (
	"net/http"
	"testing"
	"time"
)

func TestNewAPIKeyAuth(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{
			name:   "valid API key",
			apiKey: "test-api-key-12345678",
			want:   "test-api-key-12345678",
		},
		{
			name:   "API key with spaces",
			apiKey: "  test-api-key-12345678  ",
			want:   "test-api-key-12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAPIKeyAuth(tt.apiKey)
			if auth.GetAPIKey() != tt.want {
				t.Errorf("NewAPIKeyAuth() API key = %v, want %v", auth.GetAPIKey(), tt.want)
			}
		})
	}
}

func TestAPIKeyAuth_Authenticate(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test-api-key-12345678",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAPIKeyAuth(tt.apiKey)
			req, _ := http.NewRequest("GET", "http://example.com", nil)

			err := auth.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("APIKeyAuth.Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expectedAuth := "Bearer " + tt.apiKey
				if req.Header.Get("Authorization") != expectedAuth {
					t.Errorf("APIKeyAuth.Authenticate() Authorization header = %v, want %v",
						req.Header.Get("Authorization"), expectedAuth)
				}
			}
		})
	}
}

func TestAPIKeyAuth_ValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test-api-key-12345678",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
		},
		{
			name:    "too short API key",
			apiKey:  "short",
			wantErr: true,
		},
		{
			name:    "API key with invalid characters",
			apiKey:  "test-api-key-!@#$%^&*()",
			wantErr: true,
		},
		{
			name:    "valid API key with underscores",
			apiKey:  "test_api_key_12345678",
			wantErr: false,
		},
		{
			name:    "valid API key with mixed case",
			apiKey:  "Test-API-Key-12345678",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAPIKeyAuth(tt.apiKey)
			err := auth.ValidateAPIKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("APIKeyAuth.ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIKeyAuth_UpdateAPIKey(t *testing.T) {
	auth := NewAPIKeyAuth("old-api-key-12345678")

	tests := []struct {
		name      string
		newAPIKey string
		wantErr   bool
	}{
		{
			name:      "valid new API key",
			newAPIKey: "new-api-key-87654321",
			wantErr:   false,
		},
		{
			name:      "invalid new API key",
			newAPIKey: "short",
			wantErr:   true,
		},
		{
			name:      "empty new API key",
			newAPIKey: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.UpdateAPIKey(tt.newAPIKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("APIKeyAuth.UpdateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if auth.GetAPIKey() != tt.newAPIKey {
					t.Errorf("APIKeyAuth.UpdateAPIKey() API key = %v, want %v",
						auth.GetAPIKey(), tt.newAPIKey)
				}
			}
		})
	}
}

func TestAPIKeyAuth_Clone(t *testing.T) {
	original := NewAPIKeyAuth("test-api-key-12345678")
	cloned := original.Clone()

	if cloned.GetAPIKey() != original.GetAPIKey() {
		t.Errorf("APIKeyAuth.Clone() API key = %v, want %v",
			cloned.GetAPIKey(), original.GetAPIKey())
	}

	// 修改原始对象，确保克隆对象不受影响
	original.UpdateAPIKey("modified-api-key-87654321")
	if cloned.GetAPIKey() == original.GetAPIKey() {
		t.Error("APIKeyAuth.Clone() should create independent copy")
	}
}

func TestNewAPIKeyManager(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test-api-key-12345678",
			wantErr: false,
		},
		{
			name:    "invalid API key",
			apiKey:  "short",
			wantErr: true,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAPIKeyManager(tt.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAPIKeyManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if manager.GetAPIKey() != tt.apiKey {
					t.Errorf("NewAPIKeyManager() API key = %v, want %v",
						manager.GetAPIKey(), tt.apiKey)
				}

				if !manager.IsValid() {
					t.Error("NewAPIKeyManager() should create valid manager")
				}

				if time.Since(manager.GetCreatedAt()) > time.Second {
					t.Error("NewAPIKeyManager() created time should be recent")
				}
			}
		})
	}
}

func TestAPIKeyManager_SetAPIKey(t *testing.T) {
	manager, _ := NewAPIKeyManager("initial-api-key-12345678")
	initialUpdatedAt := manager.GetUpdatedAt()

	// 等待一小段时间确保时间戳有差异
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid new API key",
			apiKey:  "new-api-key-87654321",
			wantErr: false,
		},
		{
			name:    "invalid new API key",
			apiKey:  "short",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetAPIKey(tt.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("APIKeyManager.SetAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if manager.GetAPIKey() != tt.apiKey {
					t.Errorf("APIKeyManager.SetAPIKey() API key = %v, want %v",
						manager.GetAPIKey(), tt.apiKey)
				}

				if !manager.GetUpdatedAt().After(initialUpdatedAt) {
					t.Error("APIKeyManager.SetAPIKey() should update timestamp")
				}
			}
		})
	}
}

func TestAPIKeyManager_GetAuth(t *testing.T) {
	apiKey := "test-api-key-12345678"
	manager, _ := NewAPIKeyManager(apiKey)

	auth := manager.GetAuth()
	if auth == nil {
		t.Fatal("APIKeyManager.GetAuth() should not return nil")
	}

	if auth.GetAPIKey() != apiKey {
		t.Errorf("APIKeyManager.GetAuth() API key = %v, want %v",
			auth.GetAPIKey(), apiKey)
	}

	// 测试认证功能
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if err := auth.Authenticate(req); err != nil {
		t.Errorf("APIKeyManager.GetAuth().Authenticate() error = %v", err)
	}

	expectedAuth := "Bearer " + apiKey
	if req.Header.Get("Authorization") != expectedAuth {
		t.Errorf("APIKeyManager.GetAuth().Authenticate() Authorization header = %v, want %v",
			req.Header.Get("Authorization"), expectedAuth)
	}
}

func TestAPIKeyManager_Refresh(t *testing.T) {
	manager, _ := NewAPIKeyManager("test-api-key-12345678")
	initialUpdatedAt := manager.GetUpdatedAt()

	// 等待一小段时间确保时间戳有差异
	time.Sleep(10 * time.Millisecond)

	err := manager.Refresh(nil)
	if err != nil {
		t.Errorf("APIKeyManager.Refresh() error = %v", err)
	}

	if !manager.GetUpdatedAt().After(initialUpdatedAt) {
		t.Error("APIKeyManager.Refresh() should update timestamp")
	}
}

func TestAPIKeyManager_ToCredentials(t *testing.T) {
	apiKey := "test-api-key-12345678"
	manager, _ := NewAPIKeyManager(apiKey)

	creds := manager.ToCredentials()
	if creds == nil {
		t.Fatal("APIKeyManager.ToCredentials() should not return nil")
	}

	if creds.APIKey != apiKey {
		t.Errorf("APIKeyManager.ToCredentials() API key = %v, want %v",
			creds.APIKey, apiKey)
	}

	if creds.CreatedAt.IsZero() {
		t.Error("APIKeyManager.ToCredentials() created time should not be zero")
	}

	if creds.UpdatedAt.IsZero() {
		t.Error("APIKeyManager.ToCredentials() updated time should not be zero")
	}

	if creds.ExpiresAt != nil {
		t.Error("APIKeyManager.ToCredentials() expires time should be nil for API keys")
	}
}

func TestAPIKeyManager_FromCredentials(t *testing.T) {
	manager, _ := NewAPIKeyManager("initial-api-key-12345678")

	creds := &APIKeyCredentials{
		APIKey:    "restored-api-key-87654321",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-30 * time.Minute),
	}

	err := manager.FromCredentials(creds)
	if err != nil {
		t.Errorf("APIKeyManager.FromCredentials() error = %v", err)
		return
	}

	if manager.GetAPIKey() != creds.APIKey {
		t.Errorf("APIKeyManager.FromCredentials() API key = %v, want %v",
			manager.GetAPIKey(), creds.APIKey)
	}

	if !manager.GetCreatedAt().Equal(creds.CreatedAt) {
		t.Errorf("APIKeyManager.FromCredentials() created time = %v, want %v",
			manager.GetCreatedAt(), creds.CreatedAt)
	}

	if !manager.GetUpdatedAt().Equal(creds.UpdatedAt) {
		t.Errorf("APIKeyManager.FromCredentials() updated time = %v, want %v",
			manager.GetUpdatedAt(), creds.UpdatedAt)
	}

	// 测试无效凭证
	err = manager.FromCredentials(nil)
	if err == nil {
		t.Error("APIKeyManager.FromCredentials() should return error for nil credentials")
	}

	invalidCreds := &APIKeyCredentials{
		APIKey:    "short", // 无效的API key
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = manager.FromCredentials(invalidCreds)
	if err == nil {
		t.Error("APIKeyManager.FromCredentials() should return error for invalid API key")
	}
}