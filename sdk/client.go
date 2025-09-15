package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client 是 TaskCenter 的 Go SDK 客户端
type Client struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	businessID  int64
	config      *Config
	retryPolicy *RetryPolicy
}

// Config 客户端配置选项
type Config struct {
	BaseURL     string        // TaskCenter 服务基础URL
	APIKey      string        // API 密钥
	BusinessID  int64         // 业务系统ID
	Timeout     time.Duration // 请求超时时间
	RetryPolicy *RetryPolicy  // 重试策略
	UserAgent   string        // 用户代理字符串
}

// RetryPolicy 重试策略配置
type RetryPolicy struct {
	MaxRetries      int           // 最大重试次数
	InitialInterval time.Duration // 初始重试间隔
	MaxInterval     time.Duration // 最大重试间隔
	Multiplier      float64       // 重试间隔倍数
	RetryableErrors []int         // 可重试的HTTP状态码
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:   30 * time.Second,
		UserAgent: "TaskCenter-Go-SDK/1.0.0",
		RetryPolicy: &RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			RetryableErrors: []int{429, 500, 502, 503, 504},
		},
	}
}

// NewClient 创建新的客户端实例
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}

	if config.BusinessID <= 0 {
		return nil, fmt.Errorf("BusinessID must be greater than 0")
	}

	// 验证 BaseURL 格式
	if _, err := url.Parse(config.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid BaseURL: %w", err)
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL:     config.BaseURL,
		apiKey:      config.APIKey,
		businessID:  config.BusinessID,
		config:      config,
		retryPolicy: config.RetryPolicy,
	}

	return client, nil
}

// NewClientWithDefaults 使用默认配置创建客户端
func NewClientWithDefaults(baseURL, apiKey string, businessID int64) (*Client, error) {
	config := DefaultConfig()
	config.BaseURL = baseURL
	config.APIKey = apiKey
	config.BusinessID = businessID

	return NewClient(config)
}

// doRequest 执行HTTP请求，包含重试逻辑
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("X-Business-ID", fmt.Sprintf("%d", c.businessID))

	// 执行请求，包含重试逻辑
	return c.executeWithRetry(req)
}

// executeWithRetry 执行带重试的HTTP请求
func (c *Client) executeWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		// 如果需要重试，等待一段时间
		if attempt > 0 {
			interval := c.calculateRetryInterval(attempt)
			time.Sleep(interval)
		}

		// 克隆请求体（因为可能需要重试）
		var bodyReader io.Reader
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			bodyReader = bytes.NewReader(bodyBytes)
		}

		// 克隆请求
		clonedReq := req.Clone(req.Context())
		if bodyReader != nil {
			clonedReq.Body = io.NopCloser(bodyReader)
		}

		resp, err := c.httpClient.Do(clonedReq)
		if err != nil {
			lastErr = err
			continue
		}

		// 检查是否需要重试
		if c.shouldRetry(resp.StatusCode) && attempt < c.retryPolicy.MaxRetries {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: request failed", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.retryPolicy.MaxRetries, lastErr)
}

// shouldRetry 检查是否应该重试请求
func (c *Client) shouldRetry(statusCode int) bool {
	for _, retryableCode := range c.retryPolicy.RetryableErrors {
		if statusCode == retryableCode {
			return true
		}
	}
	return false
}

// calculateRetryInterval 计算重试间隔
func (c *Client) calculateRetryInterval(attempt int) time.Duration {
	interval := time.Duration(float64(c.retryPolicy.InitialInterval) *
		(c.retryPolicy.Multiplier * float64(attempt-1)))

	if interval > c.retryPolicy.MaxInterval {
		interval = c.retryPolicy.MaxInterval
	}

	return interval
}

// Close 关闭客户端，清理资源
func (c *Client) Close() error {
	// 这里可以添加清理逻辑，比如关闭连接池等
	return nil
}