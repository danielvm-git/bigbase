package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const (
	version         = "0.1.0"
	maxBodyBytes    = 1 << 20
	minPasswordLen  = 6
	backfillTimeout = 30 * time.Second
)

type contextKey string

const (
	ctxUserID       contextKey = "user_id"
	ctxUserEmail    contextKey = "user_email"
	ctxUserRole     contextKey = "user_role"
	ctxOrgID        contextKey = "org_id"
	ctxOrgKeyScopes contextKey = "org_key_scopes"
)

// CtxOrgID is the exported context key for org_id, used by tests and
// components that need to inject org_id into a request context.
const CtxOrgID = ctxOrgID

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

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Options struct {
	DB                 DBer
	Logger             kernel.Logger
	Secret             string
	AccessExpiry       time.Duration // Default: 24h
	RefreshExpiry      time.Duration // Default: 30 days
	GoogleClientID     string
	GoogleClientSecret string
	EmailSender        EmailSender
	PhoneSender        PhoneSender
	CORSAllowedOrigins []string
	PostLoginRedirect  string   // Default: "/admin/"
	SPAOriginAllowlist []string // Allowed origins for SPA redirect with #token=
	PublicURL          string   // Public base URL for OAuth redirects (takes precedence over Host header)
}

type Auth struct {
	db                 DBer
	logger             kernel.Logger
	secret             []byte
	accessExpiry       time.Duration
	refreshExpiry      time.Duration
	googleClientID     string
	googleClientSecret string
	googleVerifier     GoogleVerifier
	emailSender        EmailSender
	phoneSender        PhoneSender
	corsAllowedOrigins []string
	postLoginRedirect  string
	spaOriginAllowlist []string
	publicURL          string
	otpStore           OTPStore
	rateLimitStore     RateLimitStore
	loginLockoutStore  LoginLockoutStore
}

var _ kernel.Component = (*Auth)(nil)

func (a *Auth) SetGoogleVerifier(v GoogleVerifier) {
	a.googleVerifier = v
}

// CORSMiddleware returns a CORS middleware configured with the allowed origins from Options.
func (a *Auth) CORSMiddleware() func(http.Handler) http.Handler {
	return CORS(a.corsAllowedOrigins)
}

// isSPAOriginAllowed checks if the given URL matches one of the allowed SPA origins.
// It parses the URL and compares scheme + hostname exactly to prevent subdomain
// bypass attacks like https://trusted.example.com.evil.com (CWE-601).
// Relative URLs (no host) are always safe — they resolve against the current origin.
func (a *Auth) isSPAOriginAllowed(redirectURL string) bool {
	if len(a.spaOriginAllowlist) == 0 {
		return false
	}
	clean := strings.ReplaceAll(redirectURL, "\\", "/")
	parsed, err := url.Parse(clean)
	if err != nil {
		return false
	}
	// Relative redirects (no host) are always safe — same origin.
	if parsed.Hostname() == "" {
		return true
	}
	for _, allowed := range a.spaOriginAllowlist {
		allowedParsed, err := url.Parse(strings.ReplaceAll(allowed, "\\", "/"))
		if err != nil {
			continue
		}
		if parsed.Scheme == allowedParsed.Scheme && parsed.Hostname() == allowedParsed.Hostname() {
			return true
		}
	}
	return false
}

// PublicURLOrDefault returns the configured public URL.
// It never falls back to the request Host header (CWE-601 open redirect).
// Callers that need an absolute URL for OAuth must configure PublicURL.
func (a *Auth) PublicURLOrDefault(r *http.Request) string {
	_ = r // request retained for API compatibility; Host must not influence the result
	return a.publicURL
}

// cookieSecure reports whether auth cookies should set the Secure flag.
// Behind a TLS-terminating proxy r.TLS is nil, so also honor https PublicURL
// and X-Forwarded-Proto (CWE-319).
func (a *Auth) cookieSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.HasPrefix(a.publicURL, "https://") {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// oauthCallbackRedirectURI returns the absolute Google OAuth callback URL.
// Returns false when PublicURL is unset so callers fail closed instead of
// using an attacker-controlled Host header.
func (a *Auth) oauthCallbackRedirectURI() (string, bool) {
	if a.publicURL == "" {
		return "", false
	}
	return a.publicURL + "/api/auth/oauth/google/callback", true
}

func New(opts Options) *Auth {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	if opts.DB == nil {
		panic("auth: DB is required")
	}

	var secret []byte
	if opts.Secret != "" {
		secret = []byte(opts.Secret)
		// Validate programmatic secret unless running in a unit test.
		if !isTestMode() {
			validateJWTSecret(opts.Secret)
		}
	} else {
		secret = resolveJWTSecret(logger)
	}

	accessExpiry := opts.AccessExpiry
	if accessExpiry <= 0 {
		accessExpiry = 24 * time.Hour
	}
	refreshExpiry := opts.RefreshExpiry
	if refreshExpiry <= 0 {
		refreshExpiry = 30 * 24 * time.Hour
	}

	postLoginRedirect := opts.PostLoginRedirect
	if postLoginRedirect == "" {
		postLoginRedirect = "/admin/"
	}

	a := &Auth{
		db:                 opts.DB,
		logger:             logger,
		secret:             secret,
		accessExpiry:       accessExpiry,
		refreshExpiry:      refreshExpiry,
		googleClientID:     opts.GoogleClientID,
		googleClientSecret: opts.GoogleClientSecret,
		emailSender:        opts.EmailSender,
		phoneSender:        opts.PhoneSender,
		corsAllowedOrigins: opts.CORSAllowedOrigins,
		postLoginRedirect:  postLoginRedirect,
		spaOriginAllowlist: opts.SPAOriginAllowlist,
		publicURL:          strings.TrimRight(opts.PublicURL, "/"),
	}
	a.otpStore = NewDBOTPStore(opts.DB)
	a.rateLimitStore = NewDBRateLimitStore(opts.DB)
	a.loginLockoutStore = NewDBLoginLockoutStore(opts.DB)
	// CWE-601: OAuth redirect URI must not fall back to attacker-controlled Host header.
	// Require a configured publicURL when OAuth (Google) is enabled.
	// Tests bypass this guard — production must always set PUBLIC_URL.
	if a.googleClientID != "" && a.publicURL == "" {
		if !isTestMode() {
			panic("auth: publicURL is required when OAuth (googleClientID) is configured. Set BIGBASE_PUBLIC_URL or Options.PublicURL to prevent Host header poisoning (CWE-601).")
		}
		a.logger.Warn("public_url not configured; OAuth redirect URIs will use the Host header (vulnerable to Host header poisoning)")
	}
	return a
}

var testModeOverride *bool

func isTestMode() bool {
	if testModeOverride != nil {
		return *testModeOverride
	}
	return flag.Lookup("test.v") != nil
}

// knownDefaultSecrets lists secrets that must never be used in production.
var knownDefaultSecrets = []string{
	"test-secret-32-chars-long-key!!!",
	"test-secret-32-chars!!!",
	"secret",
	"changeme",
	"your-secret-here",
}

// validateJWTSecret validates that the secret is of safe length and is not a known default.
func validateJWTSecret(val string) {
	for _, known := range knownDefaultSecrets {
		if val == known {
			panic("auth: JWT secret is a known default and must not be used in production")
		}
	}
	if len(val) < 32 {
		panic("auth: JWT secret must be at least 32 bytes")
	}
}

// resolveJWTSecret reads BIGBASE_JWT_SECRET from the environment, validates it,
// and falls back to a random secret with a warning if unset.
func resolveJWTSecret(logger kernel.Logger) []byte {
	val := os.Getenv("BIGBASE_JWT_SECRET")
	if val == "" {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			panic("auth: failed to generate random secret: " + err.Error())
		}
		secret := []byte(hex.EncodeToString(raw))
		logger.Warn("JWT secret not configured, using auto-generated secret. Set BIGBASE_JWT_SECRET for persistent tokens across restarts.")
		return secret
	}
	validateJWTSecret(val)
	logger.Info("JWT secret loaded from configuration")
	return []byte(val)
}

func (a *Auth) Name() string           { return "auth" }
func (a *Auth) Version() string        { return version }
func (a *Auth) Dependencies() []string { return []string{"db"} }

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

	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS otp_codes (
		key TEXT PRIMARY KEY,
		code_hash TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		return fmt.Errorf("migrate otp_codes table: %w", err)
	}

	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS otp_rate_limits (
		key TEXT PRIMARY KEY,
		count INTEGER NOT NULL DEFAULT 0,
		window_start TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("migrate otp_rate_limits table: %w", err)
	}

	if err := a.db.Migrate(loginLockoutsMigration); err != nil {
		return fmt.Errorf("migrate login_lockouts table: %w", err)
	}

	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS audit_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		user_id INTEGER,
		email TEXT,
		ip_address TEXT,
		metadata TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate audit_events table: %w", err)
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

func (a *Auth) SetOTPStore(s OTPStore) {
	a.otpStore = s
}

// SetRateLimitStore sets the rate limit store. Used for testing.
func (a *Auth) SetRateLimitStore(s RateLimitStore) {
	a.rateLimitStore = s
}

// SetLoginLockoutStore sets the login lockout store. Used for testing.
func (a *Auth) SetLoginLockoutStore(s LoginLockoutStore) {
	a.loginLockoutStore = s
}

func (a *Auth) respondLoginFailure(w http.ResponseWriter, r *http.Request, email string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	locked, retryAfter, err := a.loginLockoutStore.RecordFailure(ctx, email)
	if err != nil {
		a.logger.Error("login lockout record failed", "error", err)
	}
	a.recordAudit("auth.login_failed", 0, email, getIP(r), nil)
	if locked {
		w.Header().Set("Retry-After", retryAfterHeader(retryAfter))
		kernel.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "account temporarily locked"})
		return
	}
	kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
}

// getIP extracts the IP address from request, stripping the port.
func getIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	ip = strings.Trim(ip, "[]")
	return ip
}

// recordAudit logs a security-sensitive event to the audit_events table.
// The write runs in a background goroutine with its own context so audit
// events survive request cancellation or client disconnect.
func (a *Auth) recordAudit(eventType string, userID int64, email, ip string, metadata map[string]any) {
	metaJSON := "{}"
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err == nil {
			metaJSON = string(b)
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := a.db.ExecContext(ctx,
			`INSERT INTO audit_events (event_type, user_id, email, ip_address, metadata, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))`,
			eventType, userID, email, ip, metaJSON,
		)
		if err != nil {
			a.logger.Error("audit write failed", "event_type", eventType, "error", err)
		}
	}()
}
