package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScaffoldDB(t *testing.T) {
	_, handler := setupAPI(t)

	t.Run("creates collection", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"products"}`)
		req := httptest.NewRequest("POST", "/api/scaffold/db", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["name"] != "products" {
			t.Errorf("expected name=products got %s", resp["name"])
		}
		if resp["status"] != "created" {
			t.Errorf("expected status=created got %s", resp["status"])
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":""}`)
		req := httptest.NewRequest("POST", "/api/scaffold/db", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 got %d", w.Code)
		}
	})

	t.Run("rejects invalid name", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"bad name!"}`)
		req := httptest.NewRequest("POST", "/api/scaffold/db", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 got %d", w.Code)
		}
	})

	t.Run("rejects GET method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/scaffold/db", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 got %d", w.Code)
		}
	})
}

func TestScaffoldRepo(t *testing.T) {
	_, handler := setupAPI(t)

	t.Run("emits scaffold event", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"my-app"}`)
		req := httptest.NewRequest("POST", "/api/scaffold/repo", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["name"] != "my-app" {
			t.Errorf("expected name=my-app got %s", resp["name"])
		}
		if resp["status"] != "initialised" {
			t.Errorf("expected status=initialised got %s", resp["status"])
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":""}`)
		req := httptest.NewRequest("POST", "/api/scaffold/repo", body)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 got %d", w.Code)
		}
	})
}

func TestScaffoldFunction(t *testing.T) {
	_, handler := setupAPI(t)

	t.Run("creates stub function", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"hello-world"}`)
		req := httptest.NewRequest("POST", "/api/scaffold/function", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["name"] != "hello-world" {
			t.Errorf("expected name=hello-world got %v", resp["name"])
		}
		if resp["status"] != "created" {
			t.Errorf("expected status=created got %v", resp["status"])
		}
		if resp["id"] == nil || resp["id"] == "" {
			t.Error("expected non-empty id")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":""}`)
		req := httptest.NewRequest("POST", "/api/scaffold/function", body)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 got %d", w.Code)
		}
	})
}
