package database

import (
	"context"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestMigrateAppliesPendingMigrationsTransactionally(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.ExpectBegin()
	database.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock($1)`)).WithArgs(migrationLockID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	database.ExpectExec(`CREATE TABLE IF NOT EXISTS "FlareSchemaMigrations"`).WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
	database.ExpectQuery(`SELECT EXISTS.*FlareSchemaMigrations`).WithArgs(int64(1)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	database.ExpectExec(`CREATE TABLE IF NOT EXISTS "AuditEvents"`).WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
	database.ExpectExec(`INSERT INTO "FlareSchemaMigrations"`).WithArgs(int64(1), "0001_baseline.sql").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	database.ExpectQuery(`SELECT EXISTS.*FlareSchemaMigrations`).WithArgs(int64(2)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	database.ExpectExec(`CREATE TABLE IF NOT EXISTS "Alerts"`).WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
	database.ExpectExec(`INSERT INTO "FlareSchemaMigrations"`).WithArgs(int64(2), "0002_alert_history.sql").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	database.ExpectCommit()

	if err = Migrate(context.Background(), database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateSkipsAppliedMigrations(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.ExpectBegin()
	database.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock($1)`)).WithArgs(migrationLockID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	database.ExpectExec(`CREATE TABLE IF NOT EXISTS "FlareSchemaMigrations"`).WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
	database.ExpectQuery(`SELECT EXISTS.*FlareSchemaMigrations`).WithArgs(int64(1)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	database.ExpectQuery(`SELECT EXISTS.*FlareSchemaMigrations`).WithArgs(int64(2)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	database.ExpectCommit()

	if err = Migrate(context.Background(), database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationVersionRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"baseline.sql", "0000_baseline.sql", "oops_baseline.sql"} {
		if _, err := migrationVersion(name); err == nil {
			t.Fatalf("migrationVersion(%q) should fail", name)
		}
	}
}
