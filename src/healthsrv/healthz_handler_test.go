package healthsrv

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drycc/storage/src/storage"
)

func TestHealthzHandler(t *testing.T) {

	healthChecker := storage.NewFakeHealthChecker()
	handler := healthZHandler(healthChecker)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/healthz", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("unexpected error creating request (%s)", err)
	}
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected response code %d", w.Code)
	}
}
