CREATE TABLE IF NOT EXISTS "Alerts" (
    "Id" uuid PRIMARY KEY,
    "Fingerprint" varchar(256) NOT NULL,
    "Kind" varchar(100) NOT NULL,
    "Severity" varchar(16) NOT NULL CHECK ("Severity" IN ('info', 'warning', 'critical')),
    "Title" varchar(200) NOT NULL,
    "Message" varchar(2000) NOT NULL,
    "Source" varchar(100) NOT NULL,
    "ResourceType" varchar(100) NULL,
    "ResourceId" varchar(256) NULL,
    "Status" varchar(16) NOT NULL CHECK ("Status" IN ('active', 'recovered')),
    "FirstSeenAt" timestamptz NOT NULL,
    "LastSeenAt" timestamptz NOT NULL,
    "RecoveredAt" timestamptz NULL,
    "OccurrenceCount" integer NOT NULL DEFAULT 1 CHECK ("OccurrenceCount" > 0)
);

CREATE TABLE IF NOT EXISTS "AlertReads" (
    "AlertId" uuid NOT NULL REFERENCES "Alerts" ("Id") ON DELETE CASCADE,
    "UserId" uuid NOT NULL REFERENCES "Users" ("Id") ON DELETE CASCADE,
    "ReadAt" timestamptz NOT NULL,
    PRIMARY KEY ("AlertId", "UserId")
);

CREATE UNIQUE INDEX IF NOT EXISTS "IX_Alerts_ActiveFingerprint"
    ON "Alerts" ("Fingerprint") WHERE "Status" = 'active';
CREATE INDEX IF NOT EXISTS "IX_Alerts_LastSeenAt" ON "Alerts" ("LastSeenAt" DESC);
CREATE INDEX IF NOT EXISTS "IX_AlertReads_UserId_ReadAt" ON "AlertReads" ("UserId", "ReadAt");
