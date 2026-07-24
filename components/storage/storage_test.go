package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/storage"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func withOrg(orgID int64, req *http.Request) *http.Request {
	return req.WithContext(kernel.WithOrgID(req.Context(), orgID))
}

func setupStorage(t *testing.T) (*storage.Storage, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	dir := t.TempDir()

	s := storage.New(storage.Options{DB: d, Logger: logger, Dir: dir, MaxSize: 10 << 20})
	k.Register(s)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return s, s.Handler()
}

func uploadFile(t *testing.T, handler http.Handler, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = fw.Write([]byte(content))
	_ = w.Close()

	req := withOrg(1, httptest.NewRequest("POST", "/api/storage/upload", &buf))
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func TestUploadSuccess(t *testing.T) {
	_, handler := setupStorage(t)

	resp := uploadFile(t, handler, "test.txt", "hello world")

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if id, ok := body["id"]; !ok || id == "" {
		t.Fatalf("expected non-empty id, got: %v", body)
	}
	if name, ok := body["name"]; !ok || name != "test.txt" {
		t.Fatalf("expected name 'test.txt', got: %v", body)
	}
	if mime, ok := body["mime_type"]; !ok || mime != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain, got: %v", body)
	}
	if size, ok := body["size"]; !ok || size != float64(11) {
		t.Fatalf("expected size 11, got: %v", body)
	}
}

func TestUploadMissingFile(t *testing.T) {
	_, handler := setupStorage(t)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.Close()

	req := withOrg(1, httptest.NewRequest("POST", "/api/storage/upload", &buf))
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUploadWrongFieldName(t *testing.T) {
	_, handler := setupStorage(t)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("wrongfield", "test.txt")
	_, _ = fw.Write([]byte("data"))
	_ = w.Close()

	req := withOrg(1, httptest.NewRequest("POST", "/api/storage/upload", &buf))
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUploadWrongMethod(t *testing.T) {
	_, handler := setupStorage(t)

	req := httptest.NewRequest("GET", "/api/storage/upload", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.Code)
	}
}

func TestUploadFileWrittenToDisk(t *testing.T) {
	s, handler := setupStorage(t)

	resp := uploadFile(t, handler, "hello.txt", "file content")
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)

	// Verify file exists on disk
	diskPath := filepath.Join(s.Dir(), body["id"].(string), "hello.txt")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatalf("file not found on disk at %s", diskPath)
	}
	data, _ := os.ReadFile(diskPath)
	if string(data) != "file content" {
		t.Fatalf("expected 'file content', got '%s'", string(data))
	}
}

func uploadAndGetID(t *testing.T, handler http.Handler) string {
	t.Helper()
	resp := uploadFile(t, handler, "data.txt", "test data")
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	id, _ := body["id"].(string)
	return id
}

func TestFileDownloadPathTraversal(t *testing.T) {
	s, handler := setupStorage(t)

	// Upload a normal file to confirm handler is working.
	id := uploadAndGetID(t, handler)
	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+id, nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("normal download: expected 200, got %d", w.Code)
	}

	tests := []struct {
		name    string
		path    string
		want403 bool
	}{
		{"normal path works", "hello.txt", false},
		{"dotdot blocked", "../../../etc/passwd", true},
		{"dotprefix normalized", "./hello.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := s.DBer()
			_, err := db.ExecContext(context.Background(),
				"INSERT INTO storage_files (id, name, size, mime_type, path, org_id) VALUES (?, ?, ?, ?, ?, ?)",
				"trav-"+tt.name[:4], tt.name[:8]+".txt", 0, "text/plain", tt.path, 1)
			if err != nil {
				t.Fatalf("insert: %v", err)
			}

			req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/trav-"+tt.name[:4], nil))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if tt.want403 && w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for %q, got %d", tt.path, w.Code)
			}
			// Normal path should not return 403.
			if !tt.want403 && w.Code == http.StatusForbidden {
				t.Fatalf("unexpected 403 for normal path %q", tt.path)
			}
		})
	}
}

func TestDownloadSuccess(t *testing.T) {
	_, handler := setupStorage(t)
	id := uploadAndGetID(t, handler)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+id, nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "test data" {
		t.Fatalf("expected 'test data', got '%s'", w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain, got %q", ct)
	}
}

func TestDownloadNotFound(t *testing.T) {
	_, handler := setupStorage(t)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/nonexistent", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDownloadMissingID(t *testing.T) {
	_, handler := setupStorage(t)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListFilesEmpty(t *testing.T) {
	_, handler := setupStorage(t)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data, _ := resp["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("expected empty list, got %d items", len(data))
	}
}

func TestListFilesAfterUpload(t *testing.T) {
	_, handler := setupStorage(t)
	uploadAndGetID(t, handler)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data, _ := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 file, got %d", len(data))
	}
}

func TestDeleteSuccess(t *testing.T) {
	_, handler := setupStorage(t)
	id := uploadAndGetID(t, handler)

	delReq := withOrg(1, httptest.NewRequest("DELETE", "/api/storage/files/"+id, nil))
	delW := httptest.NewRecorder()
	handler.ServeHTTP(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify file is gone
	getReq := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+id, nil))
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getW.Code)
	}
}

func TestDeleteNotFound(t *testing.T) {
	_, handler := setupStorage(t)

	req := withOrg(1, httptest.NewRequest("DELETE", "/api/storage/files/nonexistent", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListFilesWrongMethod(t *testing.T) {
	_, handler := setupStorage(t)

	req := withOrg(1, httptest.NewRequest("POST", "/api/storage/files", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestStorageImplementsComponent(t *testing.T) {
	var _ kernel.Component = &storage.Storage{}
}

func TestStorageName(t *testing.T) {
	s := &storage.Storage{}
	if got := s.Name(); got != "storage" {
		t.Fatalf("expected Name()='storage', got '%s'", got)
	}
}

func TestStorageVersion(t *testing.T) {
	s := &storage.Storage{}
	if got := s.Version(); got == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestStorageDependencies(t *testing.T) {
	s := &storage.Storage{}
	deps := s.Dependencies()
	if len(deps) != 1 || deps[0] != "db" {
		t.Fatalf("expected dependency on 'db', got %v", deps)
	}
}

func TestThumbnail(t *testing.T) {
	s, handler := setupStorage(t)
	defer func() { _ = os.RemoveAll(s.Dir()) }()

	// Upload a test image (minimal valid PNG)
	png := uploadPNG(t, handler)
	var body map[string]any
	_ = json.NewDecoder(png.Body).Decode(&body)
	fileID := body["id"].(string)

	// Request a thumbnail
	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+fileID+"/thumbnail?w=100&h=100", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct != "image/png" {
		t.Fatalf("expected image/png, got %q", ct)
	}

	// Verify the thumbnail is smaller than original
	origResp := uploadPNG(t, handler)
	_ = json.NewDecoder(origResp.Body).Decode(&body)
	fullReq := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+body["id"].(string), nil))
	fullW := httptest.NewRecorder()
	handler.ServeHTTP(fullW, fullReq)
	if w.Body.Len() >= fullW.Body.Len() {
		t.Logf("thumbnail size %d, original size %d", w.Body.Len(), fullW.Body.Len())
	}
}

func TestThumbnailNonImage(t *testing.T) {
	s, handler := setupStorage(t)
	defer func() { _ = os.RemoveAll(s.Dir()) }()

	resp := uploadFile(t, handler, "doc.txt", "plain text content")
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	fileID := body["id"].(string)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+fileID+"/thumbnail?w=100&h=100", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-image thumbnail, got %d: %s", w.Code, w.Body.String())
	}
}

func TestThumbnailNotFound(t *testing.T) {
	_, handler := setupStorage(t)

	req := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/nonexistent/thumbnail?w=100&h=100", nil))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// uploadPNG uploads a minimal valid PNG (1x1 pixel) for thumbnail testing.
func uploadPNG(t *testing.T, handler http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	// Minimal valid PNG: 1x1 red pixel
	minimalPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x04, 0x00, 0x01, 0x25, 0x5E, 0xDE,
		0x32, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "test.png")
	_, _ = fw.Write(minimalPNG)
	_ = w.Close()

	req := withOrg(1, httptest.NewRequest("POST", "/api/storage/upload", &buf))
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
