package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
	"golang.org/x/crypto/bcrypt"
)

func (a *Auth) decodeBody(w http.ResponseWriter, r *http.Request) (*authRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return nil, false
	}
	if req.Email == "" || req.Password == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return nil, false
	}
	if !strings.Contains(req.Email, "@") {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return nil, false
	}
	return &req, true
}

func (a *Auth) writeAuthResponse(w http.ResponseWriter, r *http.Request, status int, userID int64, email, token string) {
	expiresAt := time.Now().Add(a.accessExpiry)
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   a.cookieSecure(r),
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   int(a.accessExpiry.Seconds()),
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

	kernel.WriteJSON(w, status, map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
		"expires_at":    expiresAt.UTC().Format(time.RFC3339),
		"expires_in":    int(a.accessExpiry.Seconds()),
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
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	req, ok := a.decodeBody(w, r)
	if !ok {
		return
	}

	email := strings.ToLower(req.Email)

	if len(req.Password) < minPasswordLen {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		a.logger.Error("bcrypt hash", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	userID, role, orgID, err := a.insertUser(r.Context(), email, hash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}
		a.logger.Error("insert user", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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

	token, err := createJWT(userID, email, role, orgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("create jwt", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusCreated, userID, email, token)
	a.recordAudit("auth.register", userID, email, getIP(r), nil)
}

func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	req, ok := a.decodeBody(w, r)
	if !ok {
		return
	}

	email := strings.ToLower(req.Email)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if locked, retryAfter, err := a.loginLockoutStore.CheckLocked(ctx, email); err != nil {
		a.logger.Error("login lockout check failed", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	} else if locked {
		w.Header().Set("Retry-After", retryAfterHeader(retryAfter))
		kernel.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "account temporarily locked"})
		return
	}

	var userID, defaultOrgID int64
	var passwordHash, role string
	err := a.db.QueryRowContext(ctx,
		"SELECT id, password_hash, role, COALESCE(default_org_id, 0) FROM users WHERE email = ?", email).Scan(&userID, &passwordHash, &role, &defaultOrgID)
	if err != nil {
		a.respondLoginFailure(w, r, email)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		a.respondLoginFailure(w, r, email)
		return
	}

	if err := a.loginLockoutStore.ClearFailures(ctx, email); err != nil {
		a.logger.Error("clear login lockout failed", "error", err)
	}

	token, err := createJWT(userID, email, role, defaultOrgID, a.secret, a.accessExpiry)
	if err != nil {
		a.logger.Error("create jwt", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, userID, email, token)
	a.recordAudit("auth.login", userID, email, getIP(r), nil)
}

type userRow struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (a *Auth) handleUsers(w http.ResponseWriter, r *http.Request) {
	role, ok := UserRoleFromContext(r.Context())
	if !ok || role != "admin" {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	users, err := a.fetchUsers(r.Context())
	if err != nil {
		a.logger.Error("fetch users", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": users})
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
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	requesterID, _ := UserIDFromContext(r.Context())
	requesterRole, _ := UserRoleFromContext(r.Context())
	if requesterRole != "admin" && fmt.Sprint(requesterID) != id {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := a.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		a.logger.Error("delete user", "id", id, "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	var targetUserID int64
	_, _ = fmt.Sscan(id, &targetUserID)
	a.recordAudit("auth.user_deleted", targetUserID, "", getIP(r), nil)
}

func (a *Auth) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	email, _ := UserEmailFromContext(r.Context())
	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"id":    userID,
		"email": email,
	})
}
