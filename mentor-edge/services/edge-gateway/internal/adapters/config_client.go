package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"edge-gateway/internal/ports"
)

type HTTPConfigClient struct {
	baseURL    string
	lineaID    string
	httpClient *http.Client
}

func NewHTTPConfigClient(baseURL string, lineaID string) ports.ConfigClient {
	return &HTTPConfigClient{
		baseURL:    baseURL,
		lineaID:    lineaID,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPConfigClient) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config?linea_id="+c.lineaID, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("config-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("config-service returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode config response: %w", err)
	}
	return result, nil
}

func (c *HTTPConfigClient) UpdateConfig(ctx context.Context, patch map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/config?linea_id="+c.lineaID, bytesReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("config-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("config-service returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode config response: %w", err)
	}
	return result, nil
}

func (c *HTTPConfigClient) GetConfigVersion(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config/version?linea_id="+c.lineaID, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("config-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("config-service returned %d", resp.StatusCode)
	}

	var result struct {
		Version int `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Version, nil
}

func (c *HTTPConfigClient) StartCalibration(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/calibration/start", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("config-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("config-service returned %d", resp.StatusCode)
	}
	return nil
}
