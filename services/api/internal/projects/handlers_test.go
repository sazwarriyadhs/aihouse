package projects

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const validCreateRequest = `{
    "name": "Test House",
    "description": "Unit test",
    "location": "Bogor",
    "land_width_m": 10,
    "land_depth_m": 15,
    "orientation": "N",
    "floors": 2
}`

func performRequest(handler *Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, wantStatus, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if payload["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestCreateRejectsInvalidRequests(t *testing.T) {
	longName := strings.Repeat("a", 201)
	longDescription := strings.Repeat("a", 5001)
	longLocation := strings.Repeat("a", 301)
	oversizedBody := `{"name":"` + strings.Repeat("a", 1<<20) + `"}`

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"name":`},
		{"unknown field", `{"name":"Test","unexpected":true}`},
		{"multiple JSON values", validCreateRequest + ` {}`},
		{"empty name", `{"name":"","land_width_m":10,"land_depth_m":15,"floors":1,"orientation":"N"}`},
		{"name too long", `{"name":"` + longName + `","land_width_m":10,"land_depth_m":15,"floors":1,"orientation":"N"}`},
		{"description too long", `{"name":"Test","description":"` + longDescription + `","land_width_m":10,"land_depth_m":15,"floors":1,"orientation":"N"}`},
		{"location too long", `{"name":"Test","location":"` + longLocation + `","land_width_m":10,"land_depth_m":15,"floors":1,"orientation":"N"}`},
		{"width below minimum", `{"name":"Test","land_width_m":0.9,"land_depth_m":15,"floors":1,"orientation":"N"}`},
		{"width above maximum", `{"name":"Test","land_width_m":1001,"land_depth_m":15,"floors":1,"orientation":"N"}`},
		{"depth below minimum", `{"name":"Test","land_width_m":10,"land_depth_m":0,"floors":1,"orientation":"N"}`},
		{"depth above maximum", `{"name":"Test","land_width_m":10,"land_depth_m":1001,"floors":1,"orientation":"N"}`},
		{"zero floors", `{"name":"Test","land_width_m":10,"land_depth_m":15,"floors":0,"orientation":"N"}`},
		{"too many floors", `{"name":"Test","land_width_m":10,"land_depth_m":15,"floors":3,"orientation":"N"}`},
		{"invalid orientation", `{"name":"Test","land_width_m":10,"land_depth_m":15,"floors":1,"orientation":"NORTH"}`},
		{"body too large", oversizedBody},
	}

	handler := NewHandler(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performRequest(handler, http.MethodPost, "/api/v1/projects", tt.body)
			assertJSONError(t, rec, http.StatusBadRequest)
		})
	}
}

func TestProjectRoutesRejectUnsupportedMethods(t *testing.T) {
	handler := NewHandler(nil)

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{"POST detail route", http.MethodPost, "/api/v1/projects/00000000-0000-4000-8000-000000000001", "GET"},
		{"PUT collection route", http.MethodPut, "/api/v1/projects", "GET, POST"},
		{"DELETE collection route", http.MethodDelete, "/api/v1/projects", "GET, POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performRequest(handler, tt.method, tt.path, "")
			assertJSONError(t, rec, http.StatusMethodNotAllowed)

			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

func TestProjectRoutesRejectInvalidPathsAndIDs(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("unknown route", func(t *testing.T) {
		rec := performRequest(handler, http.MethodGet, "/api/v1/unknown", "")
		assertJSONError(t, rec, http.StatusNotFound)
	})

	t.Run("invalid project UUID", func(t *testing.T) {
		rec := performRequest(handler, http.MethodGet, "/api/v1/projects/not-a-uuid", "")
		assertJSONError(t, rec, http.StatusBadRequest)
	})
}
