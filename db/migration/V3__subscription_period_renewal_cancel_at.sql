-- Subscription billing period (renewed_at + 30 days → ends_at) and self-service cancel at period end (cancelled_at).
ALTER TABLE entitlements ADD COLUMN IF NOT EXISTS renewed_at TIMESTAMPTZ;
ALTER TABLE entitlements ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

UPDATE entitlements
SET renewed_at = created_at
WHERE type = 'SUBSCRIPTION' AND renewed_at IS NULL;

UPDATE entitlements
SET ends_at = renewed_at + INTERVAL '30 days'
WHERE type = 'SUBSCRIPTION'
  AND status = 'ACTIVE'
  AND ends_at IS NULL
  AND renewed_at IS NOT NULL;
