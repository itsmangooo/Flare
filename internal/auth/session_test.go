package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestRefreshRotatesTokenWithinExistingFamily(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	userID, tokenID, familyID := uuid.New(), uuid.New(), uuid.New()
	db.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable})
	db.ExpectQuery(`SELECT rt\."Id",rt\."FamilyId"`).WithArgs(pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows([]string{"Id", "FamilyId", "UserId", "ExpiresAt", "RevokedAt", "Email"}).AddRow(tokenID, familyID, userID, time.Now().Add(time.Hour), nil, "admin@example.com"))
	db.ExpectQuery(`SELECT r\."Name"`).WithArgs(userID).WillReturnRows(pgxmock.NewRows([]string{"Name"}).AddRow(administratorRole))
	db.ExpectExec(`INSERT INTO "RefreshTokens"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), familyID, userID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec(`UPDATE "RefreshTokens" SET "RevokedAt"`).WithArgs(tokenID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	db.ExpectExec(`INSERT INTO "AuditEvents"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "admin@example.com", "auth.refresh", "session", pgxmock.AnyArg(), 0, "", nil).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	response := httptest.NewRecorder()
	NewHandler(testConfig(), db, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(`{"refreshToken":"current-token"}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	var tokens tokenResponse
	if err := json.Unmarshal(response.Body.Bytes(), &tokens); err != nil || tokens.RefreshToken == "" || tokens.RefreshToken == "current-token" {
		t.Fatalf("invalid rotated response: %#v, %v", tokens, err)
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshReplayRevokesActiveTokenFamily(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	userID, tokenID, familyID := uuid.New(), uuid.New(), uuid.New()
	revokedAt := time.Now().Add(-time.Minute)
	db.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable})
	db.ExpectQuery(`SELECT rt\."Id",rt\."FamilyId"`).WithArgs(pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows([]string{"Id", "FamilyId", "UserId", "ExpiresAt", "RevokedAt", "Email"}).AddRow(tokenID, familyID, userID, time.Now().Add(time.Hour), &revokedAt, "admin@example.com"))
	db.ExpectExec(`UPDATE "RefreshTokens" SET "RevokedAt"`).WithArgs(userID, familyID, pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	db.ExpectExec(`INSERT INTO "AuditEvents"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "admin@example.com", "auth.refresh", "session", pgxmock.AnyArg(), 1, "", "Refresh token was invalid, expired, or reused.").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	response := httptest.NewRecorder()
	NewHandler(testConfig(), db, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(`{"refreshToken":"replayed-token"}`)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("response = %d %s; database = %v", response.Code, response.Body.String(), db.ExpectationsWereMet())
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBearerMeAcceptsCompatibleJWT(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	userID := uuid.New()
	db.ExpectQuery(`SELECT "Email" FROM "Users"`).WithArgs(userID).WillReturnRows(pgxmock.NewRows([]string{"Email"}).AddRow("admin@example.com"))
	db.ExpectQuery(`SELECT r\."Name"`).WithArgs(userID).WillReturnRows(pgxmock.NewRows([]string{"Name"}).AddRow(administratorRole))
	cfg := testConfig()
	claims := jwt.MapClaims{"sub": userID.String(), "email": "admin@example.com", "iss": cfg.JWTIssuer, "aud": cfg.JWTAudience, "exp": time.Now().Add(time.Minute).Unix()}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSigningKey))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer "+access)
	response := httptest.NewRecorder()
	NewHandler(cfg, db, testLogger()).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"roles":["Administrator"]`)) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBearerRejectsWrongSigningKey(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	claims := jwt.MapClaims{"sub": uuid.NewString(), "iss": "Flare.Api", "aud": "Flare.Mobile", "exp": time.Now().Add(time.Minute).Unix()}
	access, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("different-signing-key-with-32-bytes-minimum"))
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer "+access)
	response := httptest.NewRecorder()
	NewHandler(testConfig(), db, testLogger()).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
