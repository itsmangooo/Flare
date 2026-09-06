ALTER TABLE "Alerts"
    ADD COLUMN IF NOT EXISTS "LastNotifiedAt" timestamptz NULL;
