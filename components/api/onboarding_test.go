package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOnboarding(t *testing.T) {
	_, handler := setupAPI(t)

	t.Run("returns steps array", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/onboarding", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Steps []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
				Done  bool   `json:"done"`
			} `json:"steps"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(resp.Steps) != 4 {
			t.Fatalf("expected 4 steps, got %d", len(resp.Steps))
		}

		ids := map[string]bool{}
		for _, s := range resp.Steps {
			ids[s.ID] = true
			if s.Label == "" {
				t.Errorf("step %s has empty label", s.ID)
			}
		}

		for _, expected := range []string{"create_database", "add_collection", "deploy_function", "invite_team_member"} {
			if !ids[expected] {
				t.Errorf("missing step %q in response", expected)
			}
		}
	})

	t.Run("steps initially not done", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/onboarding", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var resp struct {
			Steps []struct {
				Done bool `json:"done"`
			} `json:"steps"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for i, s := range resp.Steps {
			if s.Done {
				t.Errorf("step[%d] expected done=false on fresh instance", i)
			}
		}
	})

	t.Run("rejects non-GET", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/onboarding", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 got %d", w.Code)
		}
	})
}
