package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/itsmangooo/flare/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const administratorRole = "Administrator"

type Handler struct {
	cfg     config.Config
	db      database
	logger  *slog.Logger
	limiter *ipLimiter
}

type ipWindow struct {
	started  time.Time
	requests int
}
type ipLimiter struct {
	mu      sync.Mutex
	clients map[string]ipWindow
}

type database interface {
	executor
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type bootstrapRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	BootstrapToken string `json:"bootstrapToken"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

type tokenResponse struct {
	AccessToken           string       `json:"accessToken"`
	AccessTokenExpiresAt  time.Time    `json:"accessTokenExpiresAt"`
	RefreshToken          string       `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time    `json:"refreshTokenExpiresAt"`
	User                  userResponse `json:"user"`
}

type identityUser struct {
	ID                uuid.UUID
	Email             string
	PasswordHash      string
	LockoutEnd        *time.Time
	AccessFailedCount int
}

func NewHandler(cfg config.Config, db database, logger *slog.Logger) http.Handler {
	handler := &Handler{cfg: cfg, db: db, logger: logger, limiter: &ipLimiter{clients: make(map[string]ipWindow)}}
	router := chi.NewRouter()
	router.Use(handler.rateLimit)
	router.Get("/bootstrap/status", handler.bootstrapStatus)
	router.Post("/bootstrap", handler.bootstrap)
	router.Post("/login", handler.login)
	router.Post("/refresh", handler.refresh)
	router.With(handler.authenticate).Post("/logout", handler.logout)
	router.With(handler.authenticate).Get("/me", handler.me)
	return router
}

func (h *Handler) bootstrapStatus(w http.ResponseWriter, r *http.Request) {
	var exists bool
	if err := h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM "Users")`).Scan(&exists); err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"required": !exists})
}

func (h *Handler) bootstrap(w http.ResponseWriter, r *http.Request) {
	var request bootstrapRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "Account could not be created.", err.Error())
		return
	}
	email, err := validEmail(request.Email)
	if err != nil || validatePassword(request.Password) != nil || request.BootstrapToken == "" || len(request.BootstrapToken) > 512 {
		writeProblem(w, http.StatusBadRequest, "Account could not be created.", "Email, password, or bootstrap token is invalid.")
		return
	}
	if h.cfg.BootstrapToken == "" || !fixedTimeEqual(h.cfg.BootstrapToken, request.BootstrapToken) {
		h.recordAudit(r.Context(), nil, email, "auth.bootstrap", "first-admin", 1, requestID(r), "Invalid bootstrap token.")
		writeProblem(w, http.StatusUnauthorized, "Bootstrap authorization failed.", "")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	var exists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM "Users")`).Scan(&exists); err != nil {
		h.internalError(w, r, err)
		return
	}
	if exists {
		writeProblem(w, http.StatusConflict, "Bootstrap is permanently disabled.", "An administrator account already exists.")
		return
	}

	passwordHash, err := hashPassword(request.Password)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	now := time.Now().UTC()
	userID, roleID := uuid.New(), uuid.New()
	_, err = tx.Exec(r.Context(), `INSERT INTO "Roles" ("Id", "Name", "NormalizedName", "ConcurrencyStamp")
		VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, roleID, administratorRole, strings.ToUpper(administratorRole), uuid.NewString())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	if err = tx.QueryRow(r.Context(), `SELECT "Id" FROM "Roles" WHERE "NormalizedName"=$1`, strings.ToUpper(administratorRole)).Scan(&roleID); err != nil {
		h.internalError(w, r, err)
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO "Users" ("Id","CreatedAt","UserName","NormalizedUserName","Email","NormalizedEmail","EmailConfirmed","PasswordHash","SecurityStamp","ConcurrencyStamp","PhoneNumberConfirmed","TwoFactorEnabled","LockoutEnabled","AccessFailedCount")
		VALUES ($1,$2,$3,$4,$3,$4,TRUE,$5,$6,$7,FALSE,FALSE,TRUE,0)`, userID, now, email, strings.ToUpper(email), passwordHash, uuid.NewString(), uuid.NewString())
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO "UserRoles" ("UserId","RoleId") VALUES ($1,$2)`, userID, roleID)
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response, _, err := h.issue(r.Context(), tx, identityUser{ID: userID, Email: email}, []string{administratorRole}, clientIP(r), now, uuid.New())
	if err == nil {
		err = insertAudit(r.Context(), tx, &userID, email, "auth.bootstrap", "first-admin", 0, requestID(r), "", now)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil || request.Password == "" || len(request.Password) > 128 {
		writeProblem(w, http.StatusBadRequest, "Sign-in failed.", "Email or password is invalid.")
		return
	}
	email, err := validEmail(request.Email)
	if err != nil {
		writeProblem(w, http.StatusUnauthorized, "Sign-in failed.", "Email or password is incorrect.")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	var user identityUser
	err = tx.QueryRow(r.Context(), `SELECT "Id","Email","PasswordHash","LockoutEnd","AccessFailedCount" FROM "Users" WHERE "NormalizedEmail"=$1 FOR UPDATE`, strings.ToUpper(email)).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.LockoutEnd, &user.AccessFailedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		h.recordAudit(r.Context(), nil, email, "auth.login", email, 1, requestID(r), "Unknown account.")
		writeProblem(w, http.StatusUnauthorized, "Sign-in failed.", "Email or password is incorrect.")
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	now := time.Now().UTC()
	if user.LockoutEnd != nil && user.LockoutEnd.After(now) {
		if err = insertAudit(r.Context(), tx, &user.ID, email, "auth.login", email, 1, requestID(r), "Account locked.", now); err == nil {
			err = tx.Commit(r.Context())
		}
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		writeProblem(w, http.StatusLocked, "Account temporarily locked.", "Try again later.")
		return
	}
	if !verifyPassword(user.PasswordHash, request.Password) {
		user.AccessFailedCount++
		var lockoutEnd *time.Time
		locked := false
		if user.AccessFailedCount >= 5 {
			lockedUntil := now.Add(15 * time.Minute)
			lockoutEnd, user.AccessFailedCount = &lockedUntil, 0
			locked = true
		}
		_, err = tx.Exec(r.Context(), `UPDATE "Users" SET "AccessFailedCount"=$2,"LockoutEnd"=$3 WHERE "Id"=$1`, user.ID, user.AccessFailedCount, lockoutEnd)
		if err == nil {
			detail := "Invalid credentials."
			if locked {
				detail = "Account locked."
			}
			err = insertAudit(r.Context(), tx, &user.ID, email, "auth.login", email, 1, requestID(r), detail, now)
		}
		if err == nil {
			err = tx.Commit(r.Context())
		}
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		if locked {
			writeProblem(w, http.StatusLocked, "Account temporarily locked.", "Try again later.")
		} else {
			writeProblem(w, http.StatusUnauthorized, "Sign-in failed.", "Email or password is incorrect.")
		}
		return
	}

	if _, err = tx.Exec(r.Context(), `UPDATE "Users" SET "AccessFailedCount"=0 WHERE "Id"=$1`, user.ID); err != nil {
		h.internalError(w, r, err)
		return
	}
	roles, err := rolesForUser(r.Context(), tx, user.ID)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response, _, err := h.issue(r.Context(), tx, user, roles, clientIP(r), now, uuid.New())
	if err == nil {
		err = insertAudit(r.Context(), tx, &user.ID, email, "auth.login", email, 0, requestID(r), "", now)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) issue(ctx context.Context, tx pgx.Tx, user identityUser, roles []string, ip string, now time.Time, familyID uuid.UUID) (tokenResponse, uuid.UUID, error) {
	accessExpiry, refreshExpiry := now.Add(h.cfg.AccessTokenTTL), now.Add(h.cfg.RefreshTokenTTL)
	claims := jwt.MapClaims{"sub": user.ID.String(), "email": user.Email, "jti": uuid.NewString(), "iss": h.cfg.JWTIssuer, "aud": h.cfg.JWTAudience, "nbf": now.Unix(), "exp": accessExpiry.Unix(),
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier": user.ID.String(), "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": user.Email}
	if len(roles) == 1 {
		claims["http://schemas.microsoft.com/ws/2008/06/identity/claims/role"] = roles[0]
	} else if len(roles) > 1 {
		claims["http://schemas.microsoft.com/ws/2008/06/identity/claims/role"] = roles
	}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.cfg.JWTSigningKey))
	if err != nil {
		return tokenResponse{}, uuid.Nil, err
	}
	random := make([]byte, 64)
	if _, err = rand.Read(random); err != nil {
		return tokenResponse{}, uuid.Nil, err
	}
	refresh := base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(refresh))
	tokenID := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO "RefreshTokens" ("Id","TokenHash","FamilyId","UserId","CreatedAt","ExpiresAt","CreatedByIp") VALUES ($1,$2,$3,$4,$5,$6,$7)`, tokenID, strings.ToUpper(hex.EncodeToString(hash[:])), familyID, user.ID, now, refreshExpiry, nullable(ip))
	return tokenResponse{access, accessExpiry, refresh, refreshExpiry, userResponse{user.ID, user.Email, roles}}, tokenID, err
}

func (h *Handler) recordAudit(ctx context.Context, userID *uuid.UUID, actor, action, target string, result int, correlationID, detail string) {
	if err := insertAudit(ctx, h.db, userID, actor, action, target, result, correlationID, detail, time.Now().UTC()); err != nil {
		h.logger.Error("audit write failed", "action", action, "request_id", correlationID, "error", err)
	}
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertAudit(ctx context.Context, db executor, userID *uuid.UUID, actor, action, target string, result int, correlationID, detail string, now time.Time) error {
	_, err := db.Exec(ctx, `INSERT INTO "AuditEvents" ("Id","UserId","Actor","Action","Target","Timestamp","Result","CorrelationId","Detail") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, uuid.New(), userID, nullable(actor), action, target, now, result, correlationID, nullable(detail))
	return err
}

func validEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || len(value) > 254 {
		return "", errors.New("invalid email")
	}
	return value, nil
}

func fixedTimeEqual(expected, supplied string) bool {
	a, b := sha256.Sum256([]byte(expected)), sha256.Sum256([]byte(supplied))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("request body is invalid")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func clientIP(r *http.Request) string {
	value := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

func (h *Handler) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now, key := time.Now(), clientIP(r)
		h.limiter.mu.Lock()
		window := h.limiter.clients[key]
		if window.started.IsZero() || now.Sub(window.started) >= time.Minute {
			window = ipWindow{started: now}
		}
		window.requests++
		h.limiter.clients[key] = window
		allowed := window.requests <= 10
		if len(h.limiter.clients) > 1024 {
			for ip, candidate := range h.limiter.clients {
				if now.Sub(candidate.started) >= time.Minute {
					delete(h.limiter.clients, ip)
				}
			}
		}
		h.limiter.mu.Unlock()
		if !allowed {
			w.Header().Set("Retry-After", "60")
			writeProblem(w, http.StatusTooManyRequests, "Too many requests.", "Try again later.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func requestID(r *http.Request) string { return middleware.GetReqID(r.Context()) }
func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("authentication request failed", "request_id", requestID(r), "error", err)
	writeProblem(w, http.StatusInternalServerError, "Request failed.", "The server could not complete the request.")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	value := map[string]any{"type": "about:blank", "title": title, "status": status}
	if detail != "" {
		value["detail"] = detail
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
