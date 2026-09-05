package systeminfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInfoPreservesFlutterContract(t *testing.T) {
	now := time.Date(2026, 9, 5, 20, 0, 0, 0, time.FixedZone("test", 2*60*60))
	responseRecorder := httptest.NewRecorder()
	NewHandler("1.3.0", func() time.Time { return now }).ServeHTTP(
		responseRecorder, httptest.NewRequest(http.MethodGet, "/info", nil))
	if responseRecorder.Code != http.StatusOK || responseRecorder.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("response = %d %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	var result response
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Name != "Flare" || result.APIVersion != "v1" || result.ServerVersion != "1.3.0" || !result.ServerTime.Equal(now.UTC()) {
		t.Fatalf("result = %#v", result)
	}
}

func TestInfoRejectsOtherPathsAndMethods(t *testing.T) {
	handler := NewHandler("test", nil)
	for _, test := range []struct {
		request *http.Request
		status  int
	}{
		{httptest.NewRequest(http.MethodPost, "/info", nil), http.StatusMethodNotAllowed},
		{httptest.NewRequest(http.MethodGet, "/other", nil), http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, test.request)
		if response.Code != test.status {
			t.Fatalf("response = %d", response.Code)
		}
	}
}
