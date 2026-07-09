package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/internal/llm"
)

func TestComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer test-key") {
			t.Fatalf("missing auth header")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "deepseek-chat" {
			t.Fatalf("model: %v", body["model"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "Check your build script."}},
			},
		})
	}))
	defer srv.Close()

	client := llm.New(llm.Config{APIKey: "test-key", BaseURL: srv.URL, Model: "deepseek-chat"})
	text, err := client.Complete(context.Background(), "API_KEY=secret\nBuild failed")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if text != "Check your build script." {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestStripSecretLines(t *testing.T) {
	in := "ok line\nAPI_KEY=abc\nSECRET: xyz\nfine"
	out := llm.StripSecretLines(in)
	if strings.Contains(out, "abc") || strings.Contains(out, "xyz") {
		t.Fatalf("secrets not stripped: %q", out)
	}
	if !strings.Contains(out, "ok line") || !strings.Contains(out, "fine") {
		t.Fatalf("expected safe lines kept: %q", out)
	}
}

func TestCompleteNotConfigured(t *testing.T) {
	client := llm.New(llm.Config{})
	_, err := client.Complete(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected error when API key missing")
	}
}
