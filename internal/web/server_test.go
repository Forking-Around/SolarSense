package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Forking-Around/SolarSense/internal/config"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	s, err := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return s.Handler()
}
func TestHomeSetsCSRFAndRendersAssessment(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	testHandler(t).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("missing CSRF cookie")
	}
	if !strings.Contains(w.Body.String(), "Does solar make sense") {
		t.Fatal("missing assessment")
	}
}
func TestAssessmentRequiresCSRF(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/assessment", strings.NewReader("bill=3000"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	testHandler(t).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d", w.Code)
	}
}
func TestAssessmentReturnsPreliminaryResult(t *testing.T) {
	h := testHandler(t)
	home := httptest.NewRecorder()
	h.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	cookie := home.Result().Cookies()[0]
	form := url.Values{"csrf_token": {cookie.Value}, "location": {"Hyderabad"}, "bill": {"3000"}, "roof_area": {"500"}, "roof_type": {"rcc"}, "orientation": {"south"}, "morning": {"full"}, "midday": {"full"}, "evening": {"full"}, "grid_rate": {"8"}, "export_rate": {"2.5"}}
	r := httptest.NewRequest(http.MethodPost, "/assessment", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Preliminary") {
		t.Fatal("unverified result was not marked preliminary")
	}
}
