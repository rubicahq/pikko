-- Agent 03: Team RBAC + invites (PRD §5.3 STORE-06, §24 RBAC-01–04)
-- Additive only: widen membership roles; add team_invites. Existing owners unchanged.

ALTER TABLE store_memberships
    DROP CONSTRAINT IF EXISTS store_memberships_role_check;

ALTER TABLE store_memberships
    ADD CONSTRAINT store_memberships_role_check
    CHECK (role IN ('owner', 'admin', 'cs', 'catalog', 'purchasing', 'packing', 'finance'));

CREATE TABLE team_invites (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    email               TEXT NOT NULL,
    email_normalized    TEXT NOT NULL,
    role                TEXT NOT NULL
        CHECK (role IN ('admin', 'cs', 'catalog', 'purchasing', 'packing', 'finance')),
    invited_by_user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash          BYTEA NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'cancelled', 'expired')),
    expires_at          TIMESTAMPTZ NOT NULL,
    accepted_at         TIMESTAMPTZ,
    accepted_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    cancelled_at        TIMESTAMPTZ,
    last_sent_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT team_invites_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX team_invites_store_id_idx ON team_invites (store_id);
CREATE INDEX team_invites_email_status_idx ON team_invites (email_normalized, status);
CREATE INDEX team_invites_expires_idx ON team_invites (expires_at)
    WHERE status = 'pending';

-- At most one pending invite per store + email
CREATE UNIQUE INDEX team_invites_store_email_pending_unique
    ON team_invites (store_id, email_normalized)
    WHERE status = 'pending';
