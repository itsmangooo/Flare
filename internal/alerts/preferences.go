package alerts

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

const preferenceQuery = `SELECT "Enabled","MinimumSeverity","RecoveryEnabled","DockerEnabled",
       "CoolifyEnabled","CloudflareEnabled","HostEnabled"
FROM "NotificationPreferences" WHERE "Id"=1`

type preferenceDatabase interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Preferences struct {
	Enabled           bool   `json:"enabled"`
	MinimumSeverity   string `json:"minimumSeverity"`
	RecoveryEnabled   bool   `json:"recoveryEnabled"`
	DockerEnabled     bool   `json:"dockerEnabled"`
	CoolifyEnabled    bool   `json:"coolifyEnabled"`
	CloudflareEnabled bool   `json:"cloudflareEnabled"`
	HostEnabled       bool   `json:"hostEnabled"`
}

type PreferenceStore struct {
	database preferenceDatabase
}

func NewPreferenceStore(database preferenceDatabase) *PreferenceStore {
	return &PreferenceStore{database: database}
}

func DefaultPreferences() Preferences {
	return Preferences{
		Enabled: true, MinimumSeverity: "warning", RecoveryEnabled: true,
		DockerEnabled: true, CoolifyEnabled: true, CloudflareEnabled: true, HostEnabled: true,
	}
}

func (store *PreferenceStore) Get(ctx context.Context) (Preferences, error) {
	preferences := DefaultPreferences()
	if store == nil || store.database == nil {
		return preferences, nil
	}
	err := store.database.QueryRow(ctx, preferenceQuery).Scan(
		&preferences.Enabled, &preferences.MinimumSeverity, &preferences.RecoveryEnabled, &preferences.DockerEnabled,
		&preferences.CoolifyEnabled, &preferences.CloudflareEnabled, &preferences.HostEnabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultPreferences(), nil
	}
	return preferences, err
}

func (store *PreferenceStore) Allow(ctx context.Context, signal Signal) (bool, error) {
	preferences, err := store.Get(ctx)
	if err != nil {
		return false, err
	}
	if !preferences.Enabled || signal.Recovery && !preferences.RecoveryEnabled {
		return false, nil
	}
	if !signal.Recovery && severityRank(signal.Severity) < severityRank(preferences.MinimumSeverity) {
		return false, nil
	}
	switch strings.ToLower(signal.Source) {
	case "docker":
		return preferences.DockerEnabled, nil
	case "coolify":
		return preferences.CoolifyEnabled, nil
	case "cloudflare":
		return preferences.CloudflareEnabled, nil
	case "host":
		return preferences.HostEnabled, nil
	default:
		return true, nil
	}
}

func validatePreferences(preferences Preferences) error {
	preferences.MinimumSeverity = strings.ToLower(strings.TrimSpace(preferences.MinimumSeverity))
	if severityRank(preferences.MinimumSeverity) == 0 {
		return errors.New("minimumSeverity must be info, warning, or critical")
	}
	return nil
}

func severityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "info":
		return 1
	case "warning":
		return 2
	case "critical":
		return 3
	default:
		return 0
	}
}
