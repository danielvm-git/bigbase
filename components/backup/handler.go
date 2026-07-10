package backup

import (
	"bytes"
	"net/http"

	"github.com/danielvm/bigbase/kernel"
)

// Handler returns an http.Handler for POST /api/backup and POST /api/restore.
// db must be an open *sql.DB.
func Handler(db SQLiteDB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/backup", handleBackup(db))
	mux.HandleFunc("POST /api/restore", handleRestore(db))
	return mux
}

func handleBackup(db SQLiteDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := Dump(r.Context(), db, &buf); err != nil {
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "backup failed"})
			return
		}
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", `attachment; filename="backup.sql"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func handleRestore(db SQLiteDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 256<<20) // 256 MB limit
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r.Body); err != nil {
			kernel.WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request too large"})
			return
		}
		if err := Restore(r.Context(), db, buf.String()); err != nil {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "restore failed"})
			return
		}
		kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "restored"})
	}
}
