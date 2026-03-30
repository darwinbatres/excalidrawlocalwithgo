-- Allow audit_events without an organization context (e.g. login, registration).
ALTER TABLE
    audit_events
ALTER COLUMN
    org_id DROP NOT NULL;

ALTER TABLE
    audit_events DROP CONSTRAINT audit_events_org_id_fkey;

ALTER TABLE
    audit_events
ADD
    CONSTRAINT audit_events_org_id_fkey FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE;