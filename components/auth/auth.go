package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
	"golang.org/x/crypto/bcrypt"
)

const (
	version        = "0.1.0"
	maxBodyBytes   = 1 << 20
	minPasswordLen = 6
	backfillTimeout = 30 * time.Second
)

type contextKey string

const (
	ctxUserID    contextKey = "user_id"
	ctxUserEmail contextKey = "user_email"
	ctxUserRole  contextKey = "user_role"
	ctxOrgID     contextKey = "org_id"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GoogleUser struct {
	GoogleID string
	Email    string
	Name     string
	Avatar   string
}

type GoogleVerifier interface {
	Verify(ctx context.Context, code string) (*GoogleUser, error)
}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Options struct {
	DB                  DBer
	Logger              Logger
	Secret              string
	GoogleClientID      string
	GoogleClientSecret  string
	EmailSender         EmailSender
	PhoneSender         PhoneSender
	CORSAllowedOrigins  []string
	PostLoginRedirect   string   // Default: "/admin/"
	SPAOriginAllowlist  []string // Allowed origins for SPA redirect with #token=
	PublicURL           string   // Public base URL for OAuth redirects (takes precedence over Host header)
}

type Auth struct {
	db                  DBer
	logger              Logger
	secret              []byte
	googleClientID      string
	googleClientSecret  string
	googleVerifier      GoogleVerifier
	emailSender         EmailSender
	phoneSender         PhoneSender
	corsAllowedOrigins  []string
	postLoginRedirect   string
	spaOriginAllowlist  []string
	publicURL           string
}

func (a *Auth) SetGoogleVerifier(v GoogleVerifier) {
	a.googleVerifier = v
}

// CORSMiddleware returns a CORS middleware configured with the allowed origins from Options.
func (a *Auth) CORSMiddleware() func(http.Handler) http.Handler {
	return CORS(a.corsAllowedOrigins)
}

// isSPAOriginAllowed checks if the given URL starts with one of the allowed SPA origins.
func (a *Auth) isSPAOriginAllowed(redirectURL string) bool {
	if len(a.spaOriginAllowlist) == 0 {
		return false
	}
	for _, origin := range a.spaOriginAllowlist {
		if strings.HasPrefix(redirectURL, origin) {
			return true
		}
	}
	return false
}

// PublicURLOrDefault returns the configured public URL, or constructs one
// from the request's Host header if PublicURL is not set.
func (a *Auth) PublicURLOrDefault(r *http.Request) string {
	if a.publicURL != "" {
		return a.publicURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if a.logger != nil {
		a.logger.Info("public_url not configured, using request Host header", "host", r.Host)
	}
	return scheme + "://" + r.Host
}

func New(opts Options) *Auth {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	if opts.DB == nil {
		panic("auth: DB is required")
	}

	secret := []byte(opts.Secret)
	if len(secret) == 0 {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			panic("auth: failed to generate random secret: " + err.Error())
		}
		secret = []byte(hex.EncodeToString(raw))
	}
	postLoginRedirect := opts.PostLoginRedirect
	if postLoginRedirect == "" {
		postLoginRedirect = "/admin/"
	}

	// Warn if Google OAuth is configured but no PublicURL is set.
	a := &Auth{
		db:                  opts.DB,
		logger:              logger,
		secret:              secret,
		googleClientID:      opts.GoogleClientID,
		googleClientSecret:  opts.GoogleClientSecret,
		emailSender:         opts.EmailSender,
		phoneSender:         opts.PhoneSender,
		corsAllowedOrigins:  opts.CORSAllowedOrigins,
		postLoginRedirect:   postLoginRedirect,
		spaOriginAllowlist:  opts.SPAOriginAllowlist,
		publicURL:           strings.TrimRight(opts.PublicURL, "/"),
	}
	if a.googleClientID != "" && a.publicURL == "" {
		a.logger.Warn("public_url not configured; OAuth redirect URIs will use the Host header (vulnerable to Host header poisoning)")
	}
	return a
}

func (a *Auth) Name() string                    { return "auth" }
func (a *Auth) Version() string                 { return version }
func (a *Auth) Dependencies() []string          { return []string{"db"} }
func (a *Auth) ConfigSchema() json.RawMessage   { return nil }
func (a *Auth) Hooks() []kernel.HookDef         { return nil }

func (a *Auth) ValidateToken(token string) (*Claims, error) {
	return verifyJWT(token, a.secret)
}

func (a *Auth) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (a *Auth) Start(ctx *kernel.Context) error {
	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("migrate users table: %w", err)
	}
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN google_id TEXT DEFAULT ''")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN avatar_url TEXT DEFAULT ''")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN name TEXT DEFAULT ''")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN phone TEXT DEFAULT ''")
	_ = a.db.Migrate("ALTER TABLE users ADD COLUMN default_org_id INTEGER")

	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS orgs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		owner_id INTEGER NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("migrate orgs table: %w", err)
	}

	if err := a.migrateInvitesTable(); err != nil {
		return err
	}
	if err := a.migrateAPIKeysTable(); err != nil {
		return fmt.Errorf("migrate api keys table: %w", err)
	}

	// Backfill: create personal orgs for existing users with NULL default_org_id
	a.BackfillDefaultOrgs()

	// Email verification columns (idempotent ALTER TABLE).
	if err := a.migrateEmailVerify(context.Background()); err != nil {
		return fmt.Errorf("migrate email verify: %w", err)
	}

	// Password reset tokens table.
	if err := a.migratePasswordReset(context.Background()); err != nil {
		return fmt.Errorf("migrate password reset: %w", err)
	}

	// Refresh tokens table.
	if err := a.migrateRefreshTokens(context.Background()); err != nil {
		return fmt.Errorf("migrate refresh tokens: %w", err)
	}

	a.logger.Info("auth component ready")
	return nil
}

// BackfillDefaultOrgs creates personal orgs for all users with NULL default_org_id.
func (a *Auth) BackfillDefaultOrgs() {
	ctx, cancel := context.WithTimeout(context.Background(), backfillTimeout)
	defer cancel()

	users := a.collectLegacyUsers(ctx)
	for _, u := range users {
		now := time.Now().UTC().Format(time.RFC3339)
		res, err := a.db.ExecContext(ctx,
			`INSERT INTO orgs (name, slug, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			u.email, u.email, u.id, now, now)
		if err != nil {
			a.logger.Error("backfill create org", "user_id", u.id, "error", err)
			continue
		}

		orgID, _ := res.LastInsertId()
		_, _ = a.db.ExecContext(ctx,
			"UPDATE users SET default_org_id = ? WHERE id = ?", orgID, u.id)
		a.logger.Info("backfill created default org", "user_id", u.id, "org_id", orgID)
	}
}

// collectLegacyUsers returns users whose default_org_id is NULL.
func (a *Auth) collectLegacyUsers(ctx context.Context) []legacyUser {
	rows, err := a.db.QueryContext(ctx,
		"SELECT id, email FROM users WHERE default_org_id IS NULL")
	if err != nil {
		a.logger.Error("backfill query", "error", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	var users []legacyUser
	for rows.Next() {
		var u legacyUser
		if err := rows.Scan(&u.id, &u.email); err != nil {
			a.logger.Error("backfill scan", "error", err)
			continue
		}
		users = append(users, u)
	}
	return users
}

type legacyUser struct {
	id    int64
	email string
}

func (a *Auth) Stop(ctx *kernel.Context) error {
	return nil
}

func (a *Auth) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", a.handleRegister)
	mux.HandleFunc("/api/auth/login", a.handleLogin)
	mux.HandleFunc("/api/auth/oauth/google", a.handleGoogleOAuth)
	mux.HandleFunc("/api/auth/oauth/google/callback", a.handleGoogleCallback)
	mux.HandleFunc("/api/auth/verify-email", a.handleVerifyEmail)
	mux.HandleFunc("/api/auth/forgot-password", a.handleForgotPassword)
	mux.HandleFunc("/api/auth/reset-password", a.handleResetPassword)
	mux.HandleFunc("/api/auth/refresh", a.handleRefresh)
	mux.HandleFunc("/api/auth/logout", a.handleLogout)
	mux.HandleFunc("/api/auth/otp/send", a.handleSendOTP)
	mux.HandleFunc("/api/auth/otp/verify", a.handleVerifyOTP)
	mux.HandleFunc("/api/auth/magic-link/send", a.handleSendMagicLink)
	mux.HandleFunc("/api/auth/magic-link/verify", a.handleVerifyMagicLink)
	mux.HandleFunc("/api/auth/phone/send", a.handleSendPhoneOTP)
	mux.HandleFunc("/api/auth/phone/verify", a.handleVerifyPhoneOTP)
	mux.HandleFunc("/api/auth/anonymous", a.handleAnonymousToken)
	mux.HandleFunc("/api/auth/oauth/google/popup", a.handlePopupCallback)
	return mux
}

func (a *Auth) ProtectedHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/users", a.handleUsers)
	mux.HandleFunc("DELETE /api/auth/users/{id}", a.handleUserByID)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)
	mux.HandleFunc("PATCH /api/auth/me", a.handleUpdateMe)
	mux.HandleFunc("GET /api/auth/me/identities", a.handleListIdentities)
	mux.HandleFunc("POST /api/auth/me/identities", a.handleLinkIdentity)
	mux.HandleFunc("DELETE /api/auth/me/identities/{provider}", a.handleUnlinkIdentity)
	mux.HandleFunc("POST /api/orgs", a.handleCreateOrg)
	mux.HandleFunc("GET /api/orgs", a.handleListOrgs)
	mux.HandleFunc("GET /api/orgs/{id}", a.handleGetOrg)
	mux.HandleFunc("PATCH /api/orgs/{id}", a.handleUpdateOrg)
	mux.HandleFunc("DELETE /api/orgs/{id}", a.handleDeleteOrg)
	mux.HandleFunc("POST /api/orgs/{id}/invites", a.handleCreateInvite)
	mux.HandleFunc("POST /api/orgs/{id}/invites/{token}/accept", a.handleAcceptInvite)
	mux.HandleFunc("GET /api/orgs/{id}/members", a.handleListMembers)
	mux.HandleFunc("POST /api/orgs/{id}/api-keys", a.handleCreateAPIKey)
	mux.HandleFunc("GET /api/orgs/{id}/api-keys", a.handleListAPIKeys)
	mux.HandleFunc("DELETE /api/orgs/{id}/api-keys/{keyID}", a.handleDeleteAPIKey)
	return a.Middleware(mux)
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			if c, err := r.Cookie("token"); err == nil {
				token = c.Value
			}
		}
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization required"})
			return
		}

		// API key authentication — resolve org_id and skip JWT checks.
		if strings.HasPrefix(token, "bb_") {
			orgID, err := a.ResolveAPIKey(token)
			if err != nil {
				a.logger.Error("api key resolution failed", "error", err)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
				return
			}
			ctx := context.WithValue(r.Context(), ctxOrgID, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := verifyJWT(token, a.secret)
		if err != nil {
			a.logger.Error("jwt verification failed", "error", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxUserEmail, claims.Email)
		ctx = context.WithValue(ctx, ctxUserRole, claims.Role)
		ctx = context.WithValue(ctx, ctxOrgID, claims.OrgID)

		// Anonymous tokens bypass org isolation and email verification
		// but are restricted to read-only methods.
		if claims.Role == "anonymous" {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r.WithContext(ctx))
			default:
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "write access not allowed for anonymous users"})
			}
			return
		}

		// Block unverified users when email verification is enabled.
		verified, vErr := a.isEmailVerified(ctx, claims.UserID)
		if vErr != nil {
			a.logger.Error("check email verified", "user_id", claims.UserID, "error", vErr)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if !verified {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "email not verified"})
			return
		}

		// Fail closed: tokens missing org_id are rejected.
		if claims.OrgID == 0 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "no organization"})
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxUserID).(int64)
	return id, ok
}

func UserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(ctxUserEmail).(string)
	return email, ok
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(ctxUserRole).(string)
	return role, ok
}

func OrgIDFromContext(ctx context.Context) (int64, bool) {
	orgID, ok := ctx.Value(ctxOrgID).(int64)
	return orgID, ok
}

// RequireAdmin returns middleware that rejects non-admin requests with 403.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := UserRoleFromContext(r.Context())
		if !ok || role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) decodeBody(w http.ResponseWriter, r *http.Request) (*authRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return nil, false
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return nil, false
	}
	if !strings.Contains(req.Email, "@") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return nil, false
	}
	return &req, true
}

func (a *Auth) writeAuthResponse(w http.ResponseWriter, r *http.Request, status int, userID int64, email, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   86400,
	})

	// Issue a refresh token for every successful auth response.
	refreshToken := ""
	familyID, famErr := generateFamilyID()
	if famErr == nil {
		rt, rtErr := a.issueRefreshToken(r.Context(), userID, familyID)
		if rtErr != nil {
			a.logger.Error("issue refresh token", "error", rtErr)
		} else {
			refreshToken = rt
		}
	} else {
		a.logger.Error("generate family id", "error", famErr)
	}

	writeJSON(w, status, map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
		"user": map[string]any{
			"id":    userID,
			"email": email,
		},
	})
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hash), nil
}

func (a *Auth) insertUser(ctx context.Context, email, passwordHash string) (int64, string, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	role := "user"
	var count int64
	if err := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		a.logger.Error("count users", "error", err)
	}
	if count == 0 {
		role = "admin"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, ?, ?)",
		email, passwordHash, role, now)
	if err != nil {
		return 0, "", 0, err
	}
	id, _ := res.LastInsertId()

	// Auto-create personal org with email as slug (use CreateOrg for consistency)
	org, err := a.CreateOrg(ctx, email, email, id)
	if err != nil {
		a.logger.Error("create personal org", "error", err)
		return id, role, 0, nil // don't fail registration if org creation fails
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, id)

	return id, role, org.ID, nil
}

func (a *Auth) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	req, ok := a.decodeBody(w, r)
	if !ok {
		return
	}

	email := strings.ToLower(req.Email)

	if len(req.Password) < minPasswordLen {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		a.logger.Error("bcrypt hash", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	userID, role, orgID, err := a.insertUser(r.Context(), email, hash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		a.logger.Error("insert user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// If an email sender is configured, generate and store a verification token.
	if a.emailSender != nil {
		vToken, vErr := generateVerifyToken()
		if vErr != nil {
			a.logger.Error("generate verify token", "error", vErr)
		} else {
			if sErr := a.storeVerifyToken(r.Context(), userID, vToken); sErr != nil {
				a.logger.Error("store verify token", "error", sErr)
			} else {
				a.sendVerificationEmail(email, vToken)
			}
		}
	}

	token, err := createJWT(userID, email, role, orgID, a.secret)
	if err != nil {
		a.logger.Error("create jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusCreated, userID, email, token)
}

func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	req, ok := a.decodeBody(w, r)
	if !ok {
		return
	}

	email := strings.ToLower(req.Email)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var userID, defaultOrgID int64
	var passwordHash, role string
	err := a.db.QueryRowContext(ctx,
		"SELECT id, password_hash, role, COALESCE(default_org_id, 0) FROM users WHERE email = ?", email).Scan(&userID, &passwordHash, &role, &defaultOrgID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	token, err := createJWT(userID, email, role, defaultOrgID, a.secret)
	if err != nil {
		a.logger.Error("create jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, userID, email, token)
}

type userRow struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (a *Auth) handleUsers(w http.ResponseWriter, r *http.Request) {
	role, ok := UserRoleFromContext(r.Context())
	if !ok || role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	users, err := a.fetchUsers(r.Context())
	if err != nil {
		a.logger.Error("fetch users", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": users})
}

func (a *Auth) fetchUsers(ctx context.Context) ([]userRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	orgID, ok := OrgIDFromContext(ctx)
	if ok && orgID > 0 {
		rows, err = a.db.QueryContext(ctx, `
			SELECT u.id, u.email, u.created_at 
			FROM users u
			WHERE u.id IN (
				SELECT owner_id FROM orgs WHERE id = ?
				UNION
				SELECT user_id FROM org_members WHERE org_id = ?
			)
			ORDER BY u.id`, orgID, orgID)
	} else {
		rows, err = a.db.QueryContext(ctx, "SELECT id, email, created_at FROM users ORDER BY id")
	}

	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			a.logger.Error("close rows in fetchUsers", "error", cerr)
		}
	}()

	users := make([]userRow, 0)
	for rows.Next() {
		var u userRow
		if err := rows.Scan(&u.ID, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (a *Auth) handleUserByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	requesterID, _ := UserIDFromContext(r.Context())
	requesterRole, _ := UserRoleFromContext(r.Context())
	if requesterRole != "admin" && fmt.Sprint(requesterID) != id {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := a.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		a.logger.Error("delete user", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *Auth) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	email, _ := UserEmailFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    userID,
		"email": email,
	})
}

func (a *Auth) handleGoogleOAuth(w http.ResponseWriter, r *http.Request) {
	if a.googleClientID == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	rawState, _ := generateID()

	// Accept optional ?redirect= for SPA token delivery.
	spaRedirect := r.URL.Query().Get("redirect")
	var googleState string
	var cookieValue string

	if spaRedirect != "" && len(a.spaOriginAllowlist) > 0 {
		if a.isSPAOriginAllowed(spaRedirect) {
			// Encode the SPA redirect into the signed state.
			cookieValue = SignSPAState(rawState, spaRedirect, a.secret)
			googleState = rawState // Google sees only the raw state.
		} else {
			// Redirect not allowed — fall back to standard flow.
			spaRedirect = ""
			cookieValue = SignState(rawState, a.secret)
			googleState = rawState
		}
	} else {
		cookieValue = SignState(rawState, a.secret)
		googleState = rawState
	}

	// Set a signed state cookie so the callback can verify this flow originated here.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    cookieValue,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   600,
	})

	redirectURI := a.PublicURLOrDefault(r) + "/api/auth/oauth/google/callback"
	url := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+email+profile&state=%s",
		a.googleClientID, url.QueryEscape(redirectURI), googleState)
	http.Redirect(w, r, url, http.StatusFound)
}

func (a *Auth) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if a.googleClientID == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}

	// Validate OAuth state to prevent CSRF.
	queryState := r.URL.Query().Get("state")
	stateCookie, stateErr := r.Cookie("oauth_state")

	// Try SPA state first (contains redirect URL), then fall back to standard state.
	var spaRedirect string
	stateValid := false
	if stateErr == nil {
		if s, r, ok := VerifySPAState(stateCookie.Value, a.secret); ok && s == queryState {
			stateValid = true
			spaRedirect = r
		}
	}
	if !stateValid {
		if stateErr != nil || !VerifyState(stateCookie.Value, queryState, a.secret) {
			a.clearOAuthStateCookie(w, r)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
			return
		}
	}

	verifier := a.googleVerifier
	if verifier == nil {
		verifier = &realGoogleVerifier{
			clientID:     a.googleClientID,
			clientSecret: a.googleClientSecret,
			redirectURI:  a.PublicURLOrDefault(r) + "/api/auth/oauth/google/callback",
		}
	}

	googleUser, err := verifier.Verify(r.Context(), code)
	if err != nil {
		a.clearOAuthStateCookie(w, r)
		a.logger.Error("google token verification failed", "error", err)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "google auth failed"})
		return
	}

	userID, orgID, err := a.findOrCreateGoogleUser(r.Context(), googleUser)
	if err != nil {
		a.clearOAuthStateCookie(w, r)
		a.logger.Error("find or create google user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := createJWT(userID, googleUser.Email, "user", orgID, a.secret)
	if err != nil {
		a.clearOAuthStateCookie(w, r)
		a.logger.Error("create jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// SPA token delivery: redirect to SPA with #token=... instead of setting cookie.
	if spaRedirect != "" && a.isSPAOriginAllowed(spaRedirect) {
		a.clearOAuthStateCookie(w, r)
		spaURL := spaRedirect + "#token=" + url.QueryEscape(token)
		http.Redirect(w, r, spaURL, http.StatusFound)
		return
	}

	// Clear the OAuth state cookie now that the flow is complete.
	a.clearOAuthStateCookie(w, r)

	// Standard flow: set HttpOnly cookie and redirect to postLoginRedirect.
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
	})
	http.Redirect(w, r, a.postLoginRedirect, http.StatusFound)
}

// handleLogout clears the authentication token cookie. This endpoint is for
// cookie-based clients. Bearer-token / localStorage clients should log out by
// dropping the token on the client. The server has no token blacklist.
func (a *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// findOrCreatePhoneUser finds or creates a user by phone number.
func (a *Auth) findOrCreatePhoneUser(ctx context.Context, phone string) (int64, int64, error) {
	email := fmt.Sprintf("phone-%s@bigbase.local", phone)
	var existingID, defaultOrgID int64
	err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(default_org_id, 0) FROM users WHERE phone = ?", phone).Scan(&existingID, &defaultOrgID)
	if err == nil {
		return existingID, defaultOrgID, nil
	}

	passwordHash, _ := hashPassword(generateTempPass())
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, phone, password_hash, role, created_at) VALUES (?, ?, ?, 'user', ?)",
		email, phone, passwordHash, now)
	if err != nil {
		return 0, 0, fmt.Errorf("insert phone user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("phone user last insert id: %w", err)
	}

	org, err := a.CreateOrg(ctx, email, email, userID)
	if err != nil {
		a.logger.Error("create personal org for phone user", "error", err)
		return userID, 0, nil
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, userID)

	return userID, org.ID, nil
}

func (a *Auth) clearOAuthStateCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) findOrCreateGoogleUser(ctx context.Context, gu *GoogleUser) (int64, int64, error) {
	if gu.GoogleID != "" {
		var existingID, defaultOrgID int64
		err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(default_org_id, 0) FROM users WHERE google_id = ?", gu.GoogleID).Scan(&existingID, &defaultOrgID)
		if err == nil {
			return existingID, defaultOrgID, nil
		}
	}

	if gu.Email != "" {
		var existingID, defaultOrgID int64
		var existingGoogleID string
		err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(google_id,''), COALESCE(default_org_id, 0) FROM users WHERE email = ?", gu.Email).Scan(&existingID, &existingGoogleID, &defaultOrgID)
		if err == nil {
			if existingGoogleID == "" || existingGoogleID != gu.GoogleID {
				_, _ = a.db.ExecContext(ctx, "UPDATE users SET google_id = ?, avatar_url = ? WHERE id = ?", gu.GoogleID, gu.Avatar, existingID)
			}
			return existingID, defaultOrgID, nil
		}
	}

	passwordHash, _ := hashPassword(generateTempPass())
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, google_id, avatar_url, role, created_at) VALUES (?, ?, ?, ?, 'user', ?)",
		gu.Email, passwordHash, gu.GoogleID, gu.Avatar, now)
	if err != nil {
		return 0, 0, fmt.Errorf("insert google user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("google user last insert id: %w", err)
	}

	// Auto-create personal org for Google OAuth users
	org, err := a.CreateOrg(ctx, gu.Email, gu.Email, userID)
	if err != nil {
		a.logger.Error("create personal org for google user", "error", err)
		return userID, 0, nil
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, userID)

	return userID, org.ID, nil
}

// findOrCreateEmailUser finds an existing user by email or creates a passwordless user.
func (a *Auth) findOrCreateEmailUser(ctx context.Context, email string) (int64, int64, error) {
	var existingID, defaultOrgID int64
	err := a.db.QueryRowContext(ctx, "SELECT id, COALESCE(default_org_id, 0) FROM users WHERE email = ?", email).Scan(&existingID, &defaultOrgID)
	if err == nil {
		return existingID, defaultOrgID, nil
	}

	// Create new passwordless user.
	passwordHash, _ := hashPassword(generateTempPass())
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, 'user', ?)",
		email, passwordHash, now)
	if err != nil {
		return 0, 0, fmt.Errorf("insert email user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("email user last insert id: %w", err)
	}

	org, err := a.CreateOrg(ctx, email, email, userID)
	if err != nil {
		a.logger.Error("create personal org", "error", err)
		return userID, 0, nil
	}
	_, _ = a.db.ExecContext(ctx, "UPDATE users SET default_org_id = ? WHERE id = ?", org.ID, userID)

	return userID, org.ID, nil
}

type realGoogleVerifier struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

func (v *realGoogleVerifier) Verify(ctx context.Context, code string) (*GoogleUser, error) {
	body := fmt.Sprintf("code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code, v.clientID, v.clientSecret, v.redirectURI)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token error: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	claims, err := decodeGoogleIDToken(tokenResp.IDToken)
	if err != nil {
		return nil, err
	}

	return &GoogleUser{
		GoogleID: claims.Sub,
		Email:    claims.Email,
		Name:     claims.Name,
		Avatar:   claims.Picture,
	}, nil
}

type googleIDClaims struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func decodeGoogleIDToken(token string) (*googleIDClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token: expected 3 parts, got %d", len(parts))
	}

	payload, err := jwtDecodePart(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode id_token payload: %w", err)
	}

	var claims googleIDClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse id_token claims: %w", err)
	}
	return &claims, nil
}

// jwtDecodePart decodes a base64url-encoded JWT part without padding
func jwtDecodePart(part string) ([]byte, error) {
	// Add padding if needed
	switch len(part) % 4 {
	case 2:
		part += "=="
	case 3:
		part += "="
	}
	return base64.RawURLEncoding.DecodeString(part)
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SignSPAState signs a state + spaRedirect pair with the server secret.
// Format: "rawState|base64(spaRedirect).hex(signature)"
func SignSPAState(state, spaRedirect string, secret []byte) string {
	payload := state + "|" + base64.RawURLEncoding.EncodeToString([]byte(spaRedirect))
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifySPAState verifies a signed SPA state and extracts the original state and redirect.
func VerifySPAState(signed string, secret []byte) (state, redirect string, ok bool) {
	lastDot := strings.LastIndex(signed, ".")
	if lastDot < 1 || lastDot == len(signed)-1 {
		return "", "", false
	}
	payload := signed[:lastDot]
	expectedMAC := signed[lastDot+1:]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedMAC), []byte(expected)) {
		return "", "", false
	}
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	redirectBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}
	return parts[0], string(redirectBytes), true
}

// signState signs a random state string with the server secret using HMAC-SHA256.
// Returns "state.hex(signature)" — a tamper-evident token bound to this server instance.
func SignState(state string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(state))
	return state + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyState checks that value (a signed state from signState) matches the
// original state and was signed with the given secret. Uses constant-time comparison.
func VerifyState(value, state string, secret []byte) bool {
	lastDot := strings.LastIndex(value, ".")
	if lastDot < 1 || lastDot == len(value)-1 {
		return false
	}
	gotState := value[:lastDot]
	if gotState != state {
		return false
	}
	expectedMAC := value[lastDot+1:]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(state))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(expected))
}

func (a *Auth) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and slug required"})
		return
	}
	if !isValidSlug(req.Slug) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be lowercase letters, numbers, and hyphens"})
		return
	}

	org, err := a.CreateOrg(r.Context(), req.Name, req.Slug, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "slug already exists"})
			return
		}
		a.logger.Error("create org", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": org})
}

func (a *Auth) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())

	orgs, err := a.ListOrgsByOwner(r.Context(), userID)
	if err != nil {
		a.logger.Error("list orgs", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": orgs})
}

func (a *Auth) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	id, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	org, orgErr := a.lookupOwnedOrg(r.Context(), id, userID)
	if orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": org})
}

func (a *Auth) handleUpdateOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	id, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), id, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" && req.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name or slug required"})
		return
	}
	if req.Slug != "" && !isValidSlug(req.Slug) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug must be lowercase letters, numbers, and hyphens"})
		return
	}

	updatedOrg, err := a.UpdateOrg(r.Context(), id, req.Name, req.Slug)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "slug already exists"})
			return
		}
		a.logger.Error("update org", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": updatedOrg})
}

func (a *Auth) handleDeleteOrg(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	id, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), id, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	if err := a.DeleteOrg(r.Context(), id); err != nil {
		a.logger.Error("delete org", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// orgError is a structured error returned by lookupOwnedOrg.
type orgError struct {
	code    int
	message string
}

// lookupOwnedOrg fetches an org by ID and ensures the caller is the owner.
// Returns the org on success, or an orgError when not found/forbidden/errored.
func (a *Auth) lookupOwnedOrg(ctx context.Context, orgID, userID int64) (*Organization, *orgError) {
	org, err := a.GetOrgByID(ctx, orgID)
	if err != nil {
		a.logger.Error("lookup org", "error", err)
		return nil, &orgError{http.StatusInternalServerError, "internal error"}
	}
	if org == nil {
		return nil, &orgError{http.StatusNotFound, "org not found"}
	}
	if org.OwnerID != userID {
		return nil, &orgError{http.StatusForbidden, "insufficient permissions"}
	}
	return org, nil
}

func parseOrgID(idStr string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id: %s", idStr)
	}
	return id, nil
}

// isValidSlug returns true if s contains only lowercase letters, digits, and hyphens.
func isValidSlug(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return len(s) > 0
}

func generateTempPass() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *Auth) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid email required"})
		return
	}
	if !validMemberRoles[req.Role] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be admin or member"})
		return
	}

	invite, err := a.CreateInvite(r.Context(), orgID, strings.ToLower(req.Email), req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invite already exists for this email"})
			return
		}
		a.logger.Error("create invite", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": invite})
}

func (a *Auth) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")
	token := r.PathValue("token")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	invite, err := a.GetInviteByToken(r.Context(), orgID, token)
	if err != nil {
		a.logger.Error("get invite by token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if invite == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invite not found or already accepted"})
		return
	}

	if err := a.AcceptInvite(r.Context(), invite, userID); err != nil {
		a.logger.Error("accept invite", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (a *Auth) handleListMembers(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	members, err := a.ListOrgMembers(r.Context(), orgID)
	if err != nil {
		a.logger.Error("list org members", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": members})
}

func (a *Auth) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	created, err := a.CreateAPIKey(r.Context(), orgID, req.Name, req.Scopes)
	if err != nil {
		a.logger.Error("create api key", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": created})
}

func (a *Auth) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	keys, err := a.ListAPIKeys(r.Context(), orgID)
	if err != nil {
		a.logger.Error("list api keys", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": keys})
}

func (a *Auth) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	idStr := r.PathValue("id")
	keyIDStr := r.PathValue("keyID")

	orgID, err := parseOrgID(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, orgErr := a.lookupOwnedOrg(r.Context(), orgID, userID); orgErr != nil {
		writeJSON(w, orgErr.code, map[string]string{"error": orgErr.message})
		return
	}

	keyID, err := parseOrgID(keyIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key id"})
		return
	}

	if err := a.DeleteAPIKey(r.Context(), orgID, keyID); err != nil {
		a.logger.Error("delete api key", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
