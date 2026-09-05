package config

import (
	"net/url"
	"strings"
	"testing"
)

func TestLoadAcceptsExistingNpgsqlConfiguration(t *testing.T) {
	t.Setenv("FLARE_JWT_SIGNING_KEY", "test-signing-key-with-at-least-32-bytes")
	t.Setenv("ConnectionStrings__Postgres", "Host=db;Port=5433;Database=flare;Username=app;Password=p@ss word;SSL Mode=Require")
	t.Setenv("COOLIFY_BASE_URL", "https://coolify.example.test/")
	t.Setenv("COOLIFY_API_TOKEN", "secret")
	t.Setenv("CLOUDFLARE_API_TOKEN", "cloudflare-secret")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "0123456789abcdef0123456789abcdef")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddress != ":8080" || cfg.CoolifyBaseURL != "https://coolify.example.test" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.CloudflareToken != "cloudflare-secret" || cfg.CloudflareAccountID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected Cloudflare config: %#v", cfg)
	}
	if cfg.HostName != "homelab" || cfg.HostProcPath != "/host/proc" || cfg.HostRootFSPath != "/host/rootfs" {
		t.Fatalf("unexpected host telemetry defaults: %#v", cfg)
	}
	for _, expected := range []string{"postgresql://app:p%40ss%20word@db:5433/flare", "sslmode=require"} {
		if !strings.Contains(cfg.DatabaseURL, expected) {
			t.Fatalf("DatabaseURL %q does not contain %q", cfg.DatabaseURL, expected)
		}
	}
}

func TestLoadRejectsPartialCoolifyConfiguration(t *testing.T) {
	t.Setenv("FLARE_JWT_SIGNING_KEY", "test-signing-key-with-at-least-32-bytes")
	t.Setenv("ConnectionStrings__Postgres", "postgresql://app:test@db/flare")
	t.Setenv("COOLIFY_BASE_URL", "https://coolify.example.test")
	t.Setenv("COOLIFY_API_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() should reject a partial Coolify configuration")
	}
}

func TestLoadValidatesOptionalCloudflareConfiguration(t *testing.T) {
	t.Setenv("FLARE_JWT_SIGNING_KEY", "test-signing-key-with-at-least-32-bytes")
	t.Setenv("ConnectionStrings__Postgres", "postgresql://app:test@db/flare")
	t.Setenv("COOLIFY_BASE_URL", "")
	t.Setenv("COOLIFY_API_TOKEN", "")
	for _, test := range []struct {
		token     string
		accountID string
	}{
		{"", "0123456789abcdef0123456789abcdef"},
		{"token", "not-a-cloudflare-account"},
	} {
		t.Setenv("CLOUDFLARE_API_TOKEN", test.token)
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", test.accountID)
		if _, err := Load(); err == nil {
			t.Fatalf("Cloudflare config token=%t account=%q was accepted", test.token != "", test.accountID)
		}
	}
	t.Setenv("CLOUDFLARE_API_TOKEN", "token")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	if _, err := Load(); err != nil {
		t.Fatalf("DNS-only Cloudflare config was rejected: %v", err)
	}
}

func TestLoadPreservesQuotedNpgsqlPassword(t *testing.T) {
	t.Setenv("FLARE_JWT_SIGNING_KEY", "test-signing-key-with-at-least-32-bytes")
	t.Setenv("ConnectionStrings__Postgres", `Host=db;Database=flare;Username=app;Password="p;ass""word";SSL Mode=Disable`)
	t.Setenv("COOLIFY_BASE_URL", "")
	t.Setenv("COOLIFY_API_TOKEN", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	parsed, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("parse DatabaseURL: %v", err)
	}
	password, present := parsed.User.Password()
	if !present || password != `p;ass"word` {
		t.Fatalf("quoted password = %q, present = %v", password, present)
	}
}

func TestLoadRejectsWeakJWTKey(t *testing.T) {
	t.Setenv("ConnectionStrings__Postgres", "postgresql://app:test@db/flare")
	t.Setenv("FLARE_JWT_SIGNING_KEY", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("Load() should reject a weak JWT signing key")
	}
}
