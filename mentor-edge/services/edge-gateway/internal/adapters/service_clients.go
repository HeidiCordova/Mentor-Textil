package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
)

type HTTPResilienciaClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPResilienciaClient(baseURL string) ports.BufferClient {
	return &HTTPResilienciaClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPResilienciaClient) GetSummary(ctx context.Context) (*domain.BufferSummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health/stats", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resiliencia unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("resiliencia returned %d: %s", resp.StatusCode, string(body))
	}

	var summary domain.BufferSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("decode buffer summary: %w", err)
	}
	return &summary, nil
}

func (c *HTTPResilienciaClient) GetRecentEvents(ctx context.Context, limit int, since *time.Time) ([]domain.Event, error) {
	url := fmt.Sprintf("%s/events/recent?limit=%d", c.baseURL, limit)
	if since != nil {
		url += "&since=" + since.UTC().Format(time.RFC3339)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resiliencia unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("resiliencia returned %d: %s", resp.StatusCode, string(body))
	}

	var events []domain.Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return events, nil
}

func (c *HTTPResilienciaClient) GetVisionCount(
	ctx context.Context,
	lineaID int,
	since time.Time,
	until time.Time,
) (*domain.VisionCountWindow, error) {
	query := url.Values{}
	query.Set("linea_id", strconv.Itoa(lineaID))
	query.Set("since", since.UTC().Format(time.RFC3339Nano))
	query.Set("until", until.UTC().Format(time.RFC3339Nano))

	endpoint := c.baseURL + "/vision/count?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resiliencia unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("resiliencia returned %d: %s", resp.StatusCode, string(body))
	}

	var result domain.VisionCountWindow
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode vision count: %w", err)
	}
	if result.LineaID != lineaID {
		return nil, fmt.Errorf("vision count line mismatch: requested %d, received %d", lineaID, result.LineaID)
	}
	if result.Count < 0 ||
		result.Since.IsZero() ||
		result.Until.IsZero() ||
		!result.Until.After(result.Since) {
		return nil, fmt.Errorf("invalid vision count response")
	}
	return &result, nil
}

func (c *HTTPResilienciaClient) GetPendingEvents(ctx context.Context, limit int) ([]domain.Event, error) {
	url := fmt.Sprintf("%s/events/pending?limit=%d", c.baseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resiliencia unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("resiliencia returned %d: %s", resp.StatusCode, string(body))
	}

	var events []domain.Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return events, nil
}

func (c *HTTPResilienciaClient) Health(ctx context.Context) (string, error) {
	return httpHealthCheck(ctx, c.httpClient, c.baseURL+"/health")
}

type HTTPDetectorClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPDetectorClient(baseURL string) ports.DetectorClient {
	return &HTTPDetectorClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *HTTPDetectorClient) Health(ctx context.Context) (string, error) {
	return httpHealthCheck(ctx, c.httpClient, c.baseURL+"/health")
}

func (c *HTTPDetectorClient) CalibrationStatus(ctx context.Context) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/calibrate", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("detector unreachable: %w", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

type HTTPEnviadorClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPEnviadorClient(baseURL string) ports.EnviadorClient {
	return &HTTPEnviadorClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *HTTPEnviadorClient) Health(ctx context.Context) (string, error) {
	return httpHealthCheck(ctx, c.httpClient, c.baseURL+"/health")
}

func httpHealthCheck(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "error", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "error", fmt.Errorf("service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "degraded", nil
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "ok", nil
	}
	if result.Status == "" {
		return "ok", nil
	}
	return result.Status, nil
}

func bytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}
