-- Revert: make org_id NOT NULL again.
ALTER TABLE
    audit_events
ALTER COLUMN
    org_id
SET
    NOT NULL;