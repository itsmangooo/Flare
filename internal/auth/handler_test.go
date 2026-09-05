package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/itsmangooo/flare/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestBootstrapStatusUsesExistingIdentityUsers(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM "Users")`)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	response := httptest.NewRecorder()
	NewHandler(testConfig(), db, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/bootstrap/status", nil))
	if response.Code != http.StatusOK || response.Body.String() != "{\"required\":true}\n" {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapCreatesFirstAdministratorTransactionally(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	roleID := uuid.New()
	db.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable})
	db.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM "Users")`)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	db.ExpectExec(`INSERT INTO "Roles"`).WithArgs(pgxmock.AnyArg(), administratorRole, "ADMINISTRATOR", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectQuery(`SELECT "Id" FROM "Roles"`).WithArgs("ADMINISTRATOR").WillReturnRows(pgxmock.NewRows([]string{"Id"}).AddRow(roleID))
	db.ExpectExec(`INSERT INTO "Users"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "admin@example.com", "ADMIN@EXAMPLE.COM", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec(`INSERT INTO "UserRoles"`).WithArgs(pgxmock.AnyArg(), roleID).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec(`INSERT INTO "RefreshTokens"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec(`INSERT INTO "AuditEvents"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "admin@example.com", "auth.bootstrap", "first-admin", pgxmock.AnyArg(), 0, "", nil).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	cfg := testConfig()
	cfg.BootstrapToken = "one-time-secret"
	body := bytes.NewBufferString(`{"email":"ADMIN@example.com","password":"Correct-Horse-7-Battery","bootstrapToken":"one-time-secret"}`)
	response := httptest.NewRecorder()
	NewHandler(cfg, db, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/bootstrap", body))
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s; database = %v", response.Code, response.Body.String(), db.ExpectationsWereMet())
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginAcceptsAspNetIdentityUserAndIssuesCompatibleTokens(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	userID := uuid.New()
	const passwordHash = "AQAAAAIAAYagAAAAEOpDQh8uKduMC2BA3YxLz9An2HSKonwpqpmFuAV4pQCSCdsJg2A+9k5nkCUx6Y0Qeg=="
	db.ExpectBegin()
	db.ExpectQuery(`SELECT "Id","Email","PasswordHash","LockoutEnd","AccessFailedCount"`).WithArgs("ADMIN@EXAMPLE.COM").WillReturnRows(
		pgxmock.NewRows([]string{"Id", "Email", "PasswordHash", "LockoutEnd", "AccessFailedCount"}).AddRow(userID, "admin@example.com", passwordHash, nil, 0))
	db.ExpectExec(`UPDATE "Users" SET "AccessFailedCount"=0`).WithArgs(userID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	db.ExpectQuery(`SELECT r\."Name"`).WithArgs(userID).WillReturnRows(pgxmock.NewRows([]string{"Name"}).AddRow(administratorRole))
	db.ExpectExec(`INSERT INTO "RefreshTokens"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), userID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec(`INSERT INTO "AuditEvents"`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "admin@example.com", "auth.login", "admin@example.com", pgxmock.AnyArg(), 0, "", nil).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	body := bytes.NewBufferString(`{"email":"admin@example.com","password":"Flare-Test-Password-9!"}`)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", body)
	NewHandler(testConfig(), db, testLogger()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	var tokens tokenResponse
	if err := json.Unmarshal(response.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse(tokens.AccessToken, func(*jwt.Token) (any, error) { return []byte(testConfig().JWTSigningKey), nil }, jwt.WithIssuer("Flare.Api"), jwt.WithAudience("Flare.Mobile"))
	if err != nil || !parsed.Valid || tokens.RefreshToken == "" || tokens.User.ID != userID {
		t.Fatalf("invalid token response: parsed=%v err=%v", parsed, err)
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRejectsMalformedBodyWithoutDatabaseAccess(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	response := httptest.NewRecorder()
	NewHandler(testConfig(), db, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":1}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticationRateLimit(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewHandler(testConfig(), db, testLogger())
	for attempt := 1; attempt <= 11; attempt++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{}`)))
		if attempt <= 10 && response.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d was limited early", attempt)
		}
		if attempt == 11 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt 11 status = %d", response.Code)
		}
	}
}

func testConfig() config.Config {
	return config.Config{JWTSigningKey: "test-signing-key-with-at-least-32-bytes", JWTIssuer: "Flare.Api", JWTAudience: "Flare.Mobile", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 30 * 24 * time.Hour}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
