package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"resiliencia/internal/application"
	"resiliencia/internal/domain"
)

type counterTestStorage struct {
	counter    *domain.VisionCounter
	counterErr error
	calls      int
	until      time.Time
}

func (s *counterTestStorage) Store(context.Context, *domain.EventBuffer) error {
	return nil
}

func (s *counterTestStorage) GetUnsyncedEvents(context.Context, int) ([]*domain.EventBuffer, error) {
	return nil, nil
}

func (s *counterTestStorage) GetRecentEvents(context.Context, int, *time.Time) ([]*domain.EventBuffer, error) {
	return nil, nil
}

func (s *counterTestStorage) GetVisionCount(context.Context, time.Time, time.Time) (*domain.VisionCount, error) {
	return nil, nil
}

func (s *counterTestStorage) GetVisionCounter(
	_ context.Context,
	until time.Time,
) (*domain.VisionCounter, error) {
	s.calls++
	s.until = until
	return s.counter, s.counterErr
}

func (s *counterTestStorage) GetPendingEvents(context.Context, int) ([]*domain.EventBuffer, error) {
	return nil, nil
}

func (s *counterTestStorage) MarkSynced(context.Context, []string) error {
	return nil
}

func (s *counterTestStorage) EventExists(context.Context, string) (bool, error) {
	return false, nil
}

func (s *counterTestStorage) GetPendingCount(context.Context) (int, error) {
	return 0, nil
}

func (s *counterTestStorage) PurgeExpired(context.Context) (int, error) {
	return 0, nil
}

func (s *counterTestStorage) MarkStaleEventsDead(context.Context, int) (int, error) {
	return 0, nil
}

func (s *counterTestStorage) GetBufferStats(context.Context) (*domain.BufferStats, error) {
	return &domain.BufferStats{}, nil
}

func (s *counterTestStorage) EmergencyPurge(context.Context, int) (int, error) {
	return 0, nil
}

func newCounterTestServer(storage *counterTestStorage) *HTTPServer {
	service := application.NewBufferService(storage, nil, nil)
	return NewHTTPServer(map[int]*application.BufferService{1: service}, 1, "0")
}

func counterRequest(
	t *testing.T,
	server *HTTPServer,
	method string,
	lineID string,
	until time.Time,
) *httptest.ResponseRecorder {
	t.Helper()

	values := url.Values{"until": []string{until.Format(time.RFC3339Nano)}}
	if lineID != "" {
		values.Set("linea_id", lineID)
	}
	request := httptest.NewRequest(method, "/vision/counter?"+values.Encode(), nil)
	response := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(response, request)
	return response
}

func TestVisionCounterReturnsFrozenCumulativeResult(t *testing.T) {
	now := time.Now().UTC()
	boundary := time.Unix(now.Unix()/300*300-300, 0).UTC()
	storage := &counterTestStorage{
		counter: &domain.VisionCounter{
			Count:          17,
			CounterEpoch:   boundary.Add(-time.Hour),
			Until:          boundary,
			AsOf:           boundary.Add(12 * time.Second),
			StateUpdatedAt: boundary.Add(8 * time.Second),
			EventType:      "CORTE",
		},
	}

	response := counterRequest(
		t,
		newCounterTestServer(storage),
		http.MethodGet,
		"1",
		boundary,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if storage.calls != 1 || !storage.until.Equal(boundary) {
		t.Fatalf("storage calls=%d until=%s", storage.calls, storage.until)
	}

	var result domain.VisionCounter
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.LineaID != 1 || result.Count != 17 || result.EventType != "CORTE" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVisionCounterRejectsMisalignedBoundary(t *testing.T) {
	storage := &counterTestStorage{}
	boundary := time.Unix(time.Now().UTC().Unix()/300*300-299, 0).UTC()

	response := counterRequest(
		t,
		newCounterTestServer(storage),
		http.MethodGet,
		"1",
		boundary,
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if storage.calls != 0 {
		t.Fatalf("storage was called %d times", storage.calls)
	}
}

func TestVisionCounterWaitsForSettleDelay(t *testing.T) {
	storage := &counterTestStorage{}
	nextBoundary := time.Unix(time.Now().UTC().Unix()/300*300+300, 0).UTC()

	response := counterRequest(
		t,
		newCounterTestServer(storage),
		http.MethodGet,
		"1",
		nextBoundary,
	)

	if response.Code != http.StatusTooEarly {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if storage.calls != 0 {
		t.Fatalf("storage was called %d times", storage.calls)
	}
}

func TestVisionCounterRejectsUnknownLine(t *testing.T) {
	storage := &counterTestStorage{}
	boundary := time.Unix(time.Now().UTC().Unix()/300*300-300, 0).UTC()

	response := counterRequest(
		t,
		newCounterTestServer(storage),
		http.MethodGet,
		"999",
		boundary,
	)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if storage.calls != 0 {
		t.Fatalf("storage was called %d times", storage.calls)
	}
}

func TestVisionCounterReportsInvalidEpoch(t *testing.T) {
	storage := &counterTestStorage{counterErr: domain.ErrVisionCounterBoundary}
	boundary := time.Unix(time.Now().UTC().Unix()/300*300-300, 0).UTC()

	response := counterRequest(
		t,
		newCounterTestServer(storage),
		http.MethodGet,
		"1",
		boundary,
	)

	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestVisionCounterIsReadOnlyHTTPMethod(t *testing.T) {
	storage := &counterTestStorage{}
	boundary := time.Unix(time.Now().UTC().Unix()/300*300-300, 0).UTC()

	response := counterRequest(
		t,
		newCounterTestServer(storage),
		http.MethodPost,
		"1",
		boundary,
	)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if storage.calls != 0 {
		t.Fatalf("storage was called %d times", storage.calls)
	}
}
