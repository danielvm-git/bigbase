package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

// secret_audit.go emits value-free audit records for secret actions
// (SC-e89s04-P0-04). Records carry actor and scope metadata — actor,
// organization, Project, environment, secret reference, action, and request
// ID — and never plaintext or ciphertext. The write is fire-and-forget with
// its own timeout so records survive request cancellation, mirroring the auth
// component's recordAudit contract.

const auditTableMigration = `CREATE TABLE IF NOT EXISTS audit_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	event_type TEXT NOT NULL,
	user_id INTEGER,
	email TEXT,
	ip_address TEXT,
	metadata TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

// secretAudit writes secret audit records into the shared audit_events table.
// The table DDL matches the auth component's migration; the idempotent
// CREATE IF NOT EXISTS makes the adapter self-contained for component tests
// while production startup ordering still guarantees the table exists.
type secretAudit struct {
	db      kernel.DBer
	logger  kernel.Logger
	once    sync.Once
	initErr error
}

func newSecretAudit(db kernel.DBer, logger kernel.Logger) *secretAudit {
	return &secretAudit{db: db, logger: logger}
}

// ensureTable applies the audit table migration exactly once per adapter.
func (a *secretAudit) ensureTable() {
	a.once.Do(func() {
		if a.db == nil {
			return
		}
		if err := a.db.Migrate(auditTableMigration); err != nil {
			a.initErr = err
			a.logger.Error("secret audit table migration failed", "error", err)
		}
	})
}

// record writes one audit row. Metadata carries only actor and scope fields;
// the secret value and ciphertext are never passed here by construction.
func (a *secretAudit) record(r *http.Request, eventType, secretKey, projectID, envID, action string) {
	a.ensureTable()
	if a.db == nil || a.initErr != nil {
		return
	}
	ctx := r.Context()
	userID, hasUser := auth.UserIDFromContext(ctx)
	email, _ := auth.UserEmailFromContext(ctx)
	orgID, _ := auth.OrgIDFromContext(ctx)

	actor := "orgkey:" + strconv.FormatInt(orgID, 10)
	actorType := "org_key"
	if hasUser {
		actor = "user:" + strconv.FormatInt(userID, 10)
		actorType = "user"
	}
	metadata := map[string]any{
		"actor":          actor,
		"actor_type":     actorType,
		"org_id":         orgID,
		"project_id":     projectID,
		"environment_id": envID,
		"secret":         secretKey,
		"action":         action,
		"request_id":     requestID(r),
	}
	metaJSON := "{}"
	if b, err := json.Marshal(metadata); err == nil {
		metaJSON = string(b)
	}

	var userIDCol any
	var emailCol any
	if hasUser {
		userIDCol = userID
		if email != "" {
			emailCol = email
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := a.db.ExecContext(ctx,
			`INSERT INTO audit_events (event_type, user_id, email, ip_address, metadata, created_at)
			 VALUES (?, ?, ?, ?, ?, datetime('now'))`,
			eventType, userIDCol, emailCol, clientIP(r), metaJSON); err != nil {
			a.logger.Error("secret audit write failed", "event_type", eventType, "error", err)
		}
	}()
}

// requestID returns the caller-provided request id when present, else a fresh
// random identifier so every audit record is traceable.
func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	id, err := kernel.GenerateID()
	if err != nil {
		return ""
	}
	return id
}

// clientIP extracts the client IP from the request, stripping the port.
func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return strings.Trim(host, "[]")
}
