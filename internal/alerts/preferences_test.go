package alerts

import (
	"context"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPreferenceStoreFiltersSeverityRecoveryAndSource(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	columns := []string{"Enabled", "MinimumSeverity", "RecoveryEnabled", "DockerEnabled", "CoolifyEnabled", "CloudflareEnabled", "HostEnabled"}
	for range 3 {
		database.ExpectQuery(regexp.QuoteMeta(preferenceQuery)).WillReturnRows(
			pgxmock.NewRows(columns).AddRow(true, "critical", false, true, true, true, false))
	}
	store := NewPreferenceStore(database)
	allowed, err := store.Allow(context.Background(), Signal{Severity: "warning", Source: "docker"})
	if err != nil || allowed {
		t.Fatalf("warning allowed = %v, err = %v", allowed, err)
	}
	allowed, err = store.Allow(context.Background(), Signal{Severity: "critical", Source: "docker", Recovery: true})
	if err != nil || allowed {
		t.Fatalf("recovery allowed = %v, err = %v", allowed, err)
	}
	allowed, err = store.Allow(context.Background(), Signal{Severity: "critical", Source: "host"})
	if err != nil || allowed {
		t.Fatalf("host allowed = %v, err = %v", allowed, err)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPreferenceStoreAllowsCriticalDockerAndRecoveryByDefault(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	columns := []string{"Enabled", "MinimumSeverity", "RecoveryEnabled", "DockerEnabled", "CoolifyEnabled", "CloudflareEnabled", "HostEnabled"}
	database.ExpectQuery(regexp.QuoteMeta(preferenceQuery)).WillReturnRows(pgxmock.NewRows(columns))
	database.ExpectQuery(regexp.QuoteMeta(preferenceQuery)).WillReturnRows(pgxmock.NewRows(columns))
	store := NewPreferenceStore(database)
	allowed, err := store.Allow(context.Background(), Signal{Severity: "critical", Source: "docker"})
	if err != nil || !allowed {
		t.Fatalf("default allowed = %v, err = %v", allowed, err)
	}
	allowed, err = store.Allow(context.Background(), Signal{Severity: "info", Source: "docker", Recovery: true})
	if err != nil || !allowed {
		t.Fatalf("default recovery allowed = %v, err = %v", allowed, err)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
