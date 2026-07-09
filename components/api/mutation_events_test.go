package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/danielvm/bigbase/components/api"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func TestMutationEventSiteID(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	var mu sync.Mutex
	var siteID string
	a := api.New(api.Options{DB: database, Logger: logger})
	k := kernel.New(logger)
	bus := k.EventBus()
	bus.Subscribe(kernel.HookDef{
		Name: "mutation",
		Handler: func(_ *kernel.Context, ev kernel.Event) error {
			mu.Lock()
			defer mu.Unlock()
			if v, ok := ev.Data["site_id"].(string); ok {
				siteID = v
			}
			return nil
		},
	})
	k.Register(database)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	body, _ := json.Marshal(map[string]any{"title": "hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/collections/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bigbase-Site-ID", "site-abc")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	mu.Lock()
	got := siteID
	mu.Unlock()
	if got != "site-abc" {
		t.Fatalf("expected site_id site-abc, got %q", got)
	}
}
