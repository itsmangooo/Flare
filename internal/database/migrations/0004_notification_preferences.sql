CREATE TABLE IF NOT EXISTS "NotificationPreferences" (
    "Id" smallint PRIMARY KEY CHECK ("Id" = 1),
    "Enabled" boolean NOT NULL,
    "MinimumSeverity" varchar(16) NOT NULL CHECK ("MinimumSeverity" IN ('info', 'warning', 'critical')),
    "RecoveryEnabled" boolean NOT NULL,
    "DockerEnabled" boolean NOT NULL,
    "CoolifyEnabled" boolean NOT NULL,
    "CloudflareEnabled" boolean NOT NULL,
    "HostEnabled" boolean NOT NULL,
    "UpdatedAt" timestamptz NOT NULL,
    "UpdatedBy" uuid NULL REFERENCES "Users" ("Id") ON DELETE SET NULL
);
