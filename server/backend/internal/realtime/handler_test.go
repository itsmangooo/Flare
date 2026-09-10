package realtime

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/itsmangooo/flare/server/backend/internal/overview"
)

type fakeSource struct {
	snapshot overview.Snapshot
	err      error
}

func (source fakeSource) Snapshot(context.Context) (overview.Snapshot, error) {
	return source.snapshot, source.err
}

type streamWriter struct {
	header http.Header
	body   bytes.Buffer
	cancel context.CancelFunc
	once   sync.Once
}

func (writer *streamWriter) Header() http.Header { return writer.header }
func (writer *streamWriter) WriteHeader(int)     {}
func (writer *streamWriter) Flush()              {}
func (writer *streamWriter) Write(value []byte) (int, error) {
	written, err := writer.body.Write(value)
	if bytes.Contains(writer.body.Bytes(), []byte("\n\n")) {
		writer.once.Do(writer.cancel)
	}
	return written, err
}

func TestStreamPublishesFlutterCompatibleSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	writer := &streamWriter{header: make(http.Header), cancel: cancel}
	snapshot := overview.Snapshot{GeneratedAt: time.Date(2026, time.September, 6, 8, 0, 0, 0, time.UTC), Freshness: "Live"}
	newHandler(fakeSource{snapshot: snapshot}, nil, time.Hour).ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))

	body := writer.body.String()
	if writer.header.Get("Content-Type") != "text/event-stream; charset=utf-8" ||
		!strings.Contains(body, "event: snapshot\n") || !strings.Contains(body, `"freshness":"Live"`) {
		t.Fatalf("stream headers=%v body=%q", writer.header, body)
	}
}

func TestStreamSanitizesInitialFailure(t *testing.T) {
	response := httptest.NewRecorder()
	newHandler(fakeSource{err: errors.New("password=private host=db.internal")}, nil, time.Hour).
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "private") || strings.Contains(response.Body.String(), "db.internal") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestStreamRejectsMutationMethods(t *testing.T) {
	response := httptest.NewRecorder()
	newHandler(fakeSource{}, nil, time.Hour).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("response = %d allow=%q", response.Code, response.Header().Get("Allow"))
	}
}
