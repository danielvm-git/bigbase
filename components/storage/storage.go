package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

var allowedMimePrefixes = []string{
	"text/",
	"image/",
	"application/pdf",
	"application/json",
	"application/xml",
	"application/zip",
	"application/gzip",
	"application/octet-stream",
}

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Storage struct {
	db      DBer
	logger  Logger
	dir     string
	maxSize int64
}

type Options struct {
	DB      DBer
	Logger  Logger
	Dir     string
	MaxSize int64
}

func New(opts Options) *Storage {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	dir := opts.Dir
	if dir == "" {
		dir = "data/storage"
	}
	maxSize := opts.MaxSize
	if maxSize == 0 {
		maxSize = 10 << 20
	}
	return &Storage{db: opts.DB, logger: logger, dir: dir, maxSize: maxSize}
}

func (s *Storage) Name() string                    { return "storage" }
func (s *Storage) Version() string                 { return version }
func (s *Storage) Dependencies() []string          { return []string{"db"} }
func (s *Storage) ConfigSchema() json.RawMessage   { return nil }
func (s *Storage) Hooks() []kernel.HookDef         { return nil }

func (s *Storage) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (s *Storage) Start(ctx *kernel.Context) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	if err := s.db.Migrate(`CREATE TABLE IF NOT EXISTS storage_files (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT NOT NULL,
		path TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate storage_files table: %w", err)
	}
	s.logger.Info("storage component ready", "dir", s.dir)
	return nil
}

func (s *Storage) Stop(ctx *kernel.Context) error {
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (s *Storage) Dir() string { return s.dir }

func (s *Storage) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/storage/upload", s.handleUpload)
	mux.HandleFunc("/api/storage/files/", s.handleFiles)
	mux.HandleFunc("/api/storage/files", s.handleFileList)
	return mux
}

type FileInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	CreatedAt string `json:"created_at"`
}

func (s *Storage) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	content, filename, err := s.readUpload(w, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	mimeType := detectMIME(content)
	if !isMIMEAllowed(mimeType) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file type not allowed: " + mimeType})
		return
	}

	id, err := generateID()
	if err != nil {
		s.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	fullPath, err := s.writeFile(id, filename, content)
	if err != nil {
		s.logger.Error("write file", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	fileSize := int64(len(content))
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.insertFileMeta(r.Context(), id, filename, fileSize, mimeType, fullPath, now); err != nil {
		s.logger.Error("insert file metadata", "id", id, "error", err)
		_ = os.RemoveAll(filepath.Join(s.dir, id))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, FileInfo{
		ID:        id,
		Name:      filename,
		Size:      fileSize,
		MimeType:  mimeType,
		CreatedAt: now,
	})
}

func (s *Storage) readUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxSize+1<<20)
	if err := r.ParseMultipartForm(s.maxSize); err != nil {
		return nil, "", fmt.Errorf("file too large or invalid form")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, "", fmt.Errorf("file field 'file' is required")
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("read uploaded file: %w", err)
	}
	return content, filepath.Base(header.Filename), nil
}

func detectMIME(content []byte) string {
	return http.DetectContentType(content[:min(len(content), 512)])
}

func isMIMEAllowed(mimeType string) bool {
	for _, p := range allowedMimePrefixes {
		if strings.HasPrefix(mimeType, p) {
			return true
		}
	}
	return false
}

func (s *Storage) writeFile(id, filename string, content []byte) (string, error) {
	fileDir := filepath.Join(s.dir, id)
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(fileDir, filename), content, 0644); err != nil {
		_ = os.RemoveAll(fileDir)
		return "", err
	}
	return filepath.Join(id, filename), nil
}

func (s *Storage) insertFileMeta(ctx context.Context, id, name string, size int64, mimeType, path, createdAt string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO storage_files (id, name, size, mime_type, path, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, name, size, mimeType, path, createdAt)
	return err
}

func (s *Storage) handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	files, err := s.fetchFiles(r.Context())
	if err != nil {
		s.logger.Error("list files", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": files})
}

func (s *Storage) fetchFiles(ctx context.Context) ([]FileInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, "SELECT id, name, size, mime_type, created_at FROM storage_files ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	files := make([]FileInfo, 0)
	for rows.Next() {
		var f FileInfo
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.MimeType, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *Storage) handleFiles(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/storage/files/")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	// Check for thumbnail sub-path: /api/storage/files/:id/thumbnail
	if strings.HasSuffix(path, "/thumbnail") {
		id := strings.TrimSuffix(path, "/thumbnail")
		s.handleThumbnail(w, r, id)
		return
	}

	if r.Method == "DELETE" {
		s.handleFileDelete(w, r, path)
		return
	}
	if r.Method == "GET" {
		s.handleFileDownload(w, r, path)
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func (s *Storage) handleThumbnail(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var name, mimeType, filePath string
	err := s.db.QueryRowContext(ctx,
		"SELECT name, mime_type, path FROM storage_files WHERE id = ?", id).Scan(&name, &mimeType, &filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}

	if !strings.HasPrefix(mimeType, "image/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is not an image"})
		return
	}

	fullPath := filepath.Join(s.dir, filepath.Clean(filePath))
	absDir, _ := filepath.Abs(s.dir)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return
	}

	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, fullPath)
}

func (s *Storage) handleFileDownload(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var name, mimeType, filePath string
	err := s.db.QueryRowContext(ctx,
		"SELECT name, mime_type, path FROM storage_files WHERE id = ?", id).Scan(&name, &mimeType, &filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}

	fullPath := filepath.Join(s.dir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		s.logger.Error("file missing from disk", "id", id, "path", fullPath)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", `inline; filename="`+sanitizeFilename(name)+`"`)
	http.ServeFile(w, r, fullPath)
}

func (s *Storage) handleFileDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var filePath string
	err := s.db.QueryRowContext(ctx, "SELECT path FROM storage_files WHERE id = ?", id).Scan(&filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}

	if _, err := s.db.ExecContext(ctx, "DELETE FROM storage_files WHERE id = ?", id); err != nil {
		s.logger.Error("delete file metadata", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	_ = os.RemoveAll(filepath.Join(s.dir, id))
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		`"`, "", `\`, "", "/", "", "\n", "", "\r", "",
		"\t", "", "\x00", "", "\x1f", "", "..", "",
	)
	return r.Replace(name)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}


