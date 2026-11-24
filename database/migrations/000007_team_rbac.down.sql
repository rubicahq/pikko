-- Reverse Agent 03 team RBAC. Fails if non-owner memberships exist (intentional).

DROP TABLE IF EXISTS team_invites;

-- Only restore owner-only CHECK when no other roles remain.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM store_memberships WHERE role <> 'owner'
    ) THEN
        RAISE EXCEPTION 'cannot down-migrate 000007: non-owner memberships exist';
    END IF;
END $$;

ALTER TABLE store_memberships
    DROP CONSTRAINT IF EXISTS store_memberships_role_check;

ALTER TABLE store_memberships
    ADD CONSTRAINT store_memberships_role_check
    CHECK (role IN ('owner'));
