package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// User is the authenticated identity attached to an API request.
type User struct {
	ID    uuid.UUID
	Email string
	Roles []string
}
type sessionKey struct{}

// UserFromContext returns the identity verified by Authenticate.
func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(sessionKey{}).(User)
	return user, ok
}

// HasRole reports whether the authenticated identity has the requested role.
func HasRole(ctx context.Context, role string) bool {
	user, ok := UserFromContext(ctx)
	if !ok {
		return false
	}
	for _, candidate := range user.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if err := decodeJSON(w, r, &request); err != nil || request.RefreshToken == "" || len(request.RefreshToken) > 512 {
		writeProblem(w, http.StatusBadRequest, "Session could not be refreshed.", "Refresh token is invalid.")
		return
	}
	hash := sha256.Sum256([]byte(request.RefreshToken))
	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	var tokenID, familyID, userID uuid.UUID
	var expiresAt time.Time
	var revokedAt *time.Time
	var email string
	err = tx.QueryRow(r.Context(), `SELECT rt."Id",rt."FamilyId",rt."UserId",rt."ExpiresAt",rt."RevokedAt",u."Email" FROM "RefreshTokens" rt JOIN "Users" u ON u."Id"=rt."UserId" WHERE rt."TokenHash"=$1 FOR UPDATE`, strings.ToUpper(hex.EncodeToString(hash[:]))).Scan(&tokenID, &familyID, &userID, &expiresAt, &revokedAt, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		h.recordAudit(r.Context(), nil, "", "auth.refresh", "session", 1, requestID(r), "Refresh token was invalid, expired, or reused.")
		writeProblem(w, http.StatusUnauthorized, "Session expired.", "Sign in again to continue.")
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	now := time.Now().UTC()
	if revokedAt != nil || !expiresAt.After(now) {
		if revokedAt != nil {
			_, err = tx.Exec(r.Context(), `UPDATE "RefreshTokens" SET "RevokedAt"=$3,"RevokedByIp"=$4 WHERE "UserId"=$1 AND "FamilyId"=$2 AND "RevokedAt" IS NULL`, userID, familyID, now, nullable(clientIP(r)))
		}
		if err == nil {
			err = insertAudit(r.Context(), tx, &userID, email, "auth.refresh", "session", 1, requestID(r), "Refresh token was invalid, expired, or reused.", now)
		}
		if err == nil {
			err = tx.Commit(r.Context())
		}
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		writeProblem(w, http.StatusUnauthorized, "Session expired.", "Sign in again to continue.")
		return
	}
	roles, err := rolesForUser(r.Context(), tx, userID)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response, replacementID, err := h.issue(r.Context(), tx, identityUser{ID: userID, Email: email}, roles, clientIP(r), now, familyID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE "RefreshTokens" SET "RevokedAt"=$2,"RevokedByIp"=$3,"ReplacedById"=$4 WHERE "Id"=$1`, tokenID, now, nullable(clientIP(r)), replacementID)
	}
	if err == nil {
		err = insertAudit(r.Context(), tx, &userID, email, "auth.refresh", "session", 0, requestID(r), "", now)
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

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if err := decodeJSON(w, r, &request); err != nil || request.RefreshToken == "" || len(request.RefreshToken) > 512 {
		writeProblem(w, http.StatusBadRequest, "Logout failed.", "Refresh token is invalid.")
		return
	}
	user, _ := UserFromContext(r.Context())
	hash := sha256.Sum256([]byte(request.RefreshToken))
	now := time.Now().UTC()
	result, err := h.db.Exec(r.Context(), `UPDATE "RefreshTokens" SET "RevokedAt"=$3,"RevokedByIp"=$4 WHERE "TokenHash"=$1 AND "UserId"=$2 AND "RevokedAt" IS NULL`, strings.ToUpper(hex.EncodeToString(hash[:])), user.ID, now, nullable(clientIP(r)))
	succeeded := result.RowsAffected() > 0
	if err == nil {
		value := 1
		detail := "Token was already inactive."
		if succeeded {
			value, detail = 0, ""
		}
		err = insertAudit(r.Context(), h.db, &user.ID, user.Email, "auth.logout", "session", value, requestID(r), detail, now)
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())
	var email string
	if err := h.db.QueryRow(r.Context(), `SELECT "Email" FROM "Users" WHERE "Id"=$1`, user.ID).Scan(&email); errors.Is(err, pgx.ErrNoRows) {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized.", "")
		return
	} else if err != nil {
		h.internalError(w, r, err)
		return
	}
	roles, err := rolesForUser(r.Context(), h.db, user.ID)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse{user.ID, email, roles})
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func rolesForUser(ctx context.Context, db queryer, userID uuid.UUID) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT r."Name" FROM "Roles" r JOIN "UserRoles" ur ON ur."RoleId"=r."Id" WHERE ur."UserId"=$1 ORDER BY r."Name"`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (h *Handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			writeProblem(w, http.StatusUnauthorized, "Unauthorized.", "")
			return
		}
		token, err := jwt.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(h.cfg.JWTSigningKey), nil
		}, jwt.WithIssuer(h.cfg.JWTIssuer), jwt.WithAudience(h.cfg.JWTAudience), jwt.WithExpirationRequired(), jwt.WithLeeway(30*time.Second))
		if err != nil || !token.Valid {
			writeProblem(w, http.StatusUnauthorized, "Unauthorized.", "")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeProblem(w, http.StatusUnauthorized, "Unauthorized.", "")
			return
		}
		id, err := uuid.Parse(stringClaim(claims, "sub"))
		if err != nil {
			writeProblem(w, http.StatusUnauthorized, "Unauthorized.", "")
			return
		}
		email := stringClaim(claims, "email")
		roles := stringClaims(claims, "http://schemas.microsoft.com/ws/2008/06/identity/claims/role")
		ctx := context.WithValue(r.Context(), sessionKey{}, User{ID: id, Email: email, Roles: roles})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func stringClaim(claims jwt.MapClaims, name string) string {
	value, _ := claims[name].(string)
	return value
}

func stringClaims(claims jwt.MapClaims, name string) []string {
	switch value := claims[name].(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []any:
		result := make([]string, 0, len(value))
		for _, candidate := range value {
			if text, ok := candidate.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}
