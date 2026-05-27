package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
	"golang.org/x/crypto/bcrypt"
)

const (
	version        = "0.1.0"
	maxBodyBytes   = 1 << 20
	minPasswordLen = 6
)

type contextKey string

const (
	ctxUserID    contextKey = "user_id"
	ctxUserEmail contextKey = "user_email"
	ctxUserRole  contextKey = "user_role"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

type DBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Migrate(migration string) error
}

type Options struct {
	DB     DBer
	Logger Logger
	Secret string
}

type Auth struct {
	db     DBer
	logger Logger
	secret []byte
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
	return &Auth{
		db:     opts.DB,
		logger: logger,
		secret: secret,
	}
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
	a.logger.Info("auth component ready")
	return nil
}

func (a *Auth) Stop(ctx *kernel.Context) error {
	return nil
}

func (a *Auth) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", a.handleRegister)
	mux.HandleFunc("/api/auth/login", a.handleLogin)
	return mux
}

func (a *Auth) ProtectedHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/users", a.handleUsers)
	mux.HandleFunc("DELETE /api/auth/users/{id}", a.handleUserByID)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)
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
		claims, err := verifyJWT(token, a.secret)
		if err != nil {
			a.logger.Error("jwt verification failed", "error", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxUserEmail, claims.Email)
		ctx = context.WithValue(ctx, ctxUserRole, claims.Role)
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
	writeJSON(w, status, map[string]any{
		"token": token,
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

func (a *Auth) insertUser(ctx context.Context, email, passwordHash string) (int64, string, error) {
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

	res, err := a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, ?, ?)",
		email, passwordHash, role, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, "", err
	}
	id, _ := res.LastInsertId()
	return id, role, nil
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

	userID, role, err := a.insertUser(r.Context(), email, hash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		a.logger.Error("insert user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	token, err := createJWT(userID, email, role, a.secret)
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

	var userID int64
	var passwordHash, role string
	err := a.db.QueryRowContext(ctx,
		"SELECT id, password_hash, role FROM users WHERE email = ?", email).Scan(&userID, &passwordHash, &role)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	token, err := createJWT(userID, email, role, a.secret)
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

	rows, err := a.db.QueryContext(ctx, "SELECT id, email, created_at FROM users ORDER BY id")
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

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
