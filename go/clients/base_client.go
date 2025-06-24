package clients

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type BaseClient struct {
	baseURL string
	client  *http.Client
	headers map[string]string
}

func NewBaseClient(baseURL string) *BaseClient {
	return &BaseClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: make(map[string]string),
	}
}

func (c *BaseClient) SetHeader(key, value string) {
	c.headers[key] = value
}

func (c *BaseClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

func (c *BaseClient) MakeRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status code: %d, response: %s", resp.StatusCode, string(responseBody))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return responseBody, nil
}

func (c *BaseClient) Get(endpoint string) ([]byte, error) {
	return c.MakeRequest("GET", endpoint, nil)
}

func (c *BaseClient) Post(endpoint string, body io.Reader) ([]byte, error) {
	return c.MakeRequest("POST", endpoint, body)
}

func (c *BaseClient) Put(endpoint string, body io.Reader) ([]byte, error) {
	return c.MakeRequest("PUT", endpoint, body)
}

func (c *BaseClient) Delete(endpoint string) ([]byte, error) {
	return c.MakeRequest("DELETE", endpoint, nil)
}