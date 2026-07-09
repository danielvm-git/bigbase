package deploy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/kernel"
)

func TestPipelineTimelineSchema(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	gitDir := t.TempDir()
	gitComp := newGitStub(gitDir)

	dep := deploy.New(deploy.Options{
		DB:     database,
		Logger: logger,
		GitDir: gitDir,
	})
	k := kernel.New(logger)
	k.Register(database)
	k.Register(gitComp)
	k.Register(dep)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	var count int
	if err := database.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM pragma_table_info('deployments') WHERE name = 'pipeline_timeline'").Scan(&count); err != nil {
		t.Fatalf("pragma query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected pipeline_timeline column, got count=%d", count)
	}

	sampleStart := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	sampleEnd := time.Date(2026, 7, 8, 12, 0, 12, 0, time.UTC)
	sample := deploy.PipelineTimeline{
		CloneStart: &sampleStart,
		CloneEnd:   &sampleEnd,
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal keys: %v", err)
	}
	for _, key := range []string{"clone_start", "clone_end"} {
		if parsed[key] == "" {
			t.Fatalf("expected JSON key %q in marshaled timeline", key)
		}
	}

	// Idempotent migration on second Start
	if err := k.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	k2 := kernel.New(logger)
	k2.Register(database)
	k2.Register(gitComp)
	k2.Register(dep)
	if err := k2.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	_ = k2.Stop()
}

func TestPipelineTimelineInstrumentation(t *testing.T) {
	t.Run("static deployment records clone timestamps", func(t *testing.T) {
		_, handler, database, gitDir := setupDeploy(t)
		repoID := createTestRepo(t, database, "repo-timeline-static", gitDir)

		depID := triggerDeploy(t, handler, repoID)
		waitForDeploymentTerminal(t, handler, depID, 10*time.Second)

		req := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET deploy: %d", w.Code)
		}

		var got map[string]any
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got["status"] != "running" {
			t.Fatalf("expected running, got %v", got["status"])
		}

		timeline, ok := got["pipeline_timeline"].(map[string]any)
		if !ok {
			t.Fatalf("expected pipeline_timeline object, got %T", got["pipeline_timeline"])
		}
		if timeline["clone_start"] == nil || timeline["clone_end"] == nil {
			t.Fatalf("expected clone_start and clone_end, got %#v", timeline)
		}
		assertTimelineOrder(t, timeline, "clone_start", "clone_end")
	})

	t.Run("failed node build records partial timeline", func(t *testing.T) {
		_, handler, database, gitDir := setupDeploy(t)
		repoID := createTestNodeRepo(t, database, "repo-timeline-fail", gitDir)

		depID := triggerDeploy(t, handler, repoID)
		waitForDeploymentTerminal(t, handler, depID, 30*time.Second)

		req := httptest.NewRequest("GET", "/api/deploy/"+depID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET deploy: %d", w.Code)
		}

		var got map[string]any
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got["status"] != "failed" {
			t.Fatalf("expected failed, got %v", got["status"])
		}

		timeline, ok := got["pipeline_timeline"].(map[string]any)
		if !ok {
			t.Fatalf("expected pipeline_timeline object, got %T", got["pipeline_timeline"])
		}
		if timeline["clone_start"] == nil || timeline["build_start"] == nil {
			t.Fatalf("expected clone_start and build_start on failed build, got %#v", timeline)
		}
		if timeline["build_end"] != nil {
			t.Fatalf("expected build_end absent on failed build, got %v", timeline["build_end"])
		}
		if timeline["start_start"] != nil || timeline["health_start"] != nil {
			t.Fatalf("expected start/health absent on failed build, got %#v", timeline)
		}
	})

	t.Run("list response includes pipeline_timeline", func(t *testing.T) {
		_, handler, database, gitDir := setupDeploy(t)
		repoID := createTestRepo(t, database, "repo-timeline-list", gitDir)
		depID := triggerDeploy(t, handler, repoID)
		waitForDeploymentTerminal(t, handler, depID, 10*time.Second)

		req := httptest.NewRequest("GET", "/api/deploy", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list: %d", w.Code)
		}

		var body map[string]any
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		items, ok := body["data"].([]any)
		if !ok || len(items) == 0 {
			t.Fatalf("expected data array, got %#v", body["data"])
		}
		first, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map item, got %T", items[0])
		}
		if first["id"] != depID {
			t.Fatalf("expected dep %s, got %v", depID, first["id"])
		}
		if _, ok := first["pipeline_timeline"].(map[string]any); !ok {
			t.Fatalf("list item missing pipeline_timeline: %#v", first)
		}
	})
}

func assertTimelineOrder(t *testing.T, timeline map[string]any, startKey, endKey string) {
	t.Helper()
	startStr, _ := timeline[startKey].(string)
	endStr, _ := timeline[endKey].(string)
	if startStr == "" || endStr == "" {
		t.Fatalf("missing %s or %s", startKey, endKey)
	}
	start, err := time.Parse(time.RFC3339Nano, startStr)
	if err != nil {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			t.Fatalf("parse %s: %v", startKey, err)
		}
	}
	end, err := time.Parse(time.RFC3339Nano, endStr)
	if err != nil {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			t.Fatalf("parse %s: %v", endKey, err)
		}
	}
	if !end.After(start) && !end.Equal(start) {
		t.Fatalf("%s (%v) should be >= %s (%v)", endKey, end, startKey, start)
	}
}
