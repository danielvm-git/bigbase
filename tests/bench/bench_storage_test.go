package bench

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/storage"
	"github.com/danielvm/bigbase/kernel"
)

func BenchmarkStorageUpload(b *testing.B) {
	logger := kernel.NoopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	s := storage.New(storage.Options{DB: d, Logger: logger, Dir: b.TempDir()})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(s)
	if err := k.Start(); err != nil {
		b.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := s.Handler()
	content := bytes.Repeat([]byte("benchmark-data-"), 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		mp := multipart.NewWriter(&buf)
		fw, _ := mp.CreateFormFile("file", "bench-file.txt")
		_, _ = fw.Write(content)
		_ = mp.Close()

		req := httptest.NewRequest("POST", "/api/storage/upload?collection_id=bench", &buf)
		req.Header.Set("Content-Type", mp.FormDataContentType())
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		_, _ = io.Copy(io.Discard, w.Result().Body)
	}
}
