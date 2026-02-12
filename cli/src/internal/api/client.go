package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client wraps HTTP calls to the Nebula REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL, apiKey string, timeout ...time.Duration) *Client {
	httpTimeout := 30 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		httpTimeout = timeout[0]
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// SetAPIKey updates the bearer token used for subsequent requests.
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// do executes an HTTP request and returns the raw response body.
func (c *Client) do(method, path string, body any) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp apiResponse[any]
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
			return nil, resp.StatusCode, fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// get performs a GET request.
func (c *Client) get(path string) ([]byte, error) {
	body, _, err := c.do(http.MethodGet, path, nil)
	return body, err
}

// post performs a POST request.
func (c *Client) post(path string, body any) ([]byte, error) {
	b, _, err := c.do(http.MethodPost, path, body)
	return b, err
}

// patch performs a PATCH request.
func (c *Client) patch(path string, body any) ([]byte, error) {
	b, _, err := c.do(http.MethodPatch, path, body)
	return b, err
}

// del performs a DELETE request.
func (c *Client) del(path string) ([]byte, error) {
	b, _, err := c.do(http.MethodDelete, path, nil)
	return b, err
}

// decodeOne decodes a single-item API response.
func decodeOne[T any](data []byte) (*T, error) {
	var resp apiResponse[T]
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &resp.Data, nil
}

// decodeList decodes a list API response.
func decodeList[T any](data []byte) ([]T, error) {
	var resp apiResponse[[]T]
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return resp.Data, nil
}

// buildQuery appends query params to a path.
func buildQuery(path string, params QueryParams) string {
	if len(params) == 0 {
		return path
	}
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	return path + "?" + q.Encode()
}
