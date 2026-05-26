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
	version      = "0.1.0"
	maxBodyBytes = 1 << 20
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

type DBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
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
	secret := []byte(opts.Secret)
	if len(secret) == 0 {
		raw := make([]byte, 32)
		_, _ = rand.Read(raw)
		secret = []byte(hex.EncodeToString(raw))
	}
	return &Auth{
		db:     opts.DB,
		logger: opts.Logger,
		secret: secret,
	}
}

func (a *Auth) Name() string                    { return "auth" }
func (a *Auth) Version() string                 { return version }
func (a *Auth) Dependencies() []string          { return []string{"db"} }
func (a *Auth) ConfigSchema() json.RawMessage   { return nil }
func (a *Auth) Hooks() []kernel.HookDef         { return nil }

func (a *Auth) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (a *Auth) Start(ctx *kernel.Context) error {
	if err := a.db.Migrate(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("migrate users table: %w", err)
	}
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

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization header required"})
			return
		}
		claims, err := verifyJWT(token, a.secret)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
		r.Header.Set("X-User-Email", claims.Email)
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
	return &req, true
}

func (a *Auth) writeAuthResponse(w http.ResponseWriter, status int, userID int64, email, token string) {
	writeJSON(w, status, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    userID,
			"email": email,
		},
	})
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
	if len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		a.logger.Error("bcrypt hash", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = a.db.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, created_at) VALUES (?, ?, ?)",
		strings.ToLower(req.Email), string(hash), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		a.logger.Error("insert user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	var userID int64
	_ = a.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", strings.ToLower(req.Email)).Scan(&userID)

	token, err := createJWT(userID, req.Email, a.secret)
	if err != nil {
		a.logger.Error("create jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, http.StatusCreated, userID, req.Email, token)
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var userID int64
	var passwordHash string
	err := a.db.QueryRowContext(ctx,
		"SELECT id, password_hash FROM users WHERE email = ?", strings.ToLower(req.Email)).Scan(&userID, &passwordHash)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	token, err := createJWT(userID, req.Email, a.secret)
	if err != nil {
		a.logger.Error("create jwt", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, http.StatusOK, userID, req.Email, token)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
