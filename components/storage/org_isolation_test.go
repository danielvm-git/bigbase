package storage_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageOrgIsolation(t *testing.T) {
	t.Run("files_scoped_by_org_id", func(t *testing.T) {
		_, handler := setupStorage(t)

		uploadForOrg := func(orgID int64, filename, content string) string {
			t.Helper()
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			fw, err := w.CreateFormFile("file", filename)
			if err != nil {
				t.Fatalf("create form file: %v", err)
			}
			_, _ = fw.Write([]byte(content))
			_ = w.Close()

			req := withOrg(orgID, httptest.NewRequest("POST", "/api/storage/upload", &buf))
			req.Header.Set("Content-Type", w.FormDataContentType())
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusCreated {
				t.Fatalf("org%d upload: expected 201, got %d: %s", orgID, resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			id, _ := body["id"].(string)
			return id
		}

		org1ID := uploadForOrg(1, "org1.txt", "org1 secret")
		org2ID := uploadForOrg(2, "org2.txt", "org2 secret")

		listForOrg := func(orgID int64) []any {
			t.Helper()
			req := withOrg(orgID, httptest.NewRequest("GET", "/api/storage/files", nil))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("org%d list: expected 200, got %d: %s", orgID, w.Code, w.Body.String())
			}
			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			data, _ := resp["data"].([]any)
			return data
		}

		org1Files := listForOrg(1)
		if len(org1Files) != 1 {
			t.Fatalf("org1 should see 1 file, got %d", len(org1Files))
		}
		org1File := org1Files[0].(map[string]any)
		if org1File["name"] != "org1.txt" {
			t.Fatalf("org1 should see org1.txt, got %v", org1File["name"])
		}

		org2Files := listForOrg(2)
		if len(org2Files) != 1 {
			t.Fatalf("org2 should see 1 file, got %d", len(org2Files))
		}
		org2File := org2Files[0].(map[string]any)
		if org2File["name"] != "org2.txt" {
			t.Fatalf("org2 should see org2.txt, got %v", org2File["name"])
		}

		// Org 1 cannot download org 2's file
		dlReq := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+org2ID, nil))
		dlW := httptest.NewRecorder()
		handler.ServeHTTP(dlW, dlReq)
		if dlW.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant download: expected 404, got %d", dlW.Code)
		}

		// Org 1 cannot delete org 2's file
		delReq := withOrg(1, httptest.NewRequest("DELETE", "/api/storage/files/"+org2ID, nil))
		delW := httptest.NewRecorder()
		handler.ServeHTTP(delW, delReq)
		if delW.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant delete: expected 404, got %d", delW.Code)
		}

		// Org 2 can still download its own file
		okReq := withOrg(2, httptest.NewRequest("GET", "/api/storage/files/"+org2ID, nil))
		okW := httptest.NewRecorder()
		handler.ServeHTTP(okW, okReq)
		if okW.Code != http.StatusOK {
			t.Fatalf("org2 own download: expected 200, got %d", okW.Code)
		}
		if okW.Body.String() != "org2 secret" {
			t.Fatalf("expected org2 secret, got %q", okW.Body.String())
		}

		// Org 1 can download its own file
		ownReq := withOrg(1, httptest.NewRequest("GET", "/api/storage/files/"+org1ID, nil))
		ownW := httptest.NewRecorder()
		handler.ServeHTTP(ownW, ownReq)
		if ownW.Code != http.StatusOK {
			t.Fatalf("org1 own download: expected 200, got %d", ownW.Code)
		}
	})

	t.Run("storage_requires_org_id", func(t *testing.T) {
		_, handler := setupStorage(t)

		req := httptest.NewRequest("GET", "/api/storage/files", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without org_id, got %d: %s", w.Code, w.Body.String())
		}
	})
}
