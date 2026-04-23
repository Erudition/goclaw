-- Add chat_id to vault_documents for cross-chat isolation within isolated teams.
-- NULL = team-wide doc (shared mode or legacy); non-NULL = scoped to specific chat.
ALTER TABLE vault_documents ADD COLUMN IF NOT EXISTS chat_id TEXT;

-- Composite index for team + chat filtering (primary query pattern for isolated teams).
CREATE INDEX IF NOT EXISTS idx_vault_docs_team_chat
    ON vault_documents(team_id, chat_id)
    WHERE team_id IS NOT NULL;

-- -----------------------------------------------------------------------------
-- Backfill 1: team-scoped docs (scope='team', team_id set).
-- Two path layouts:
--   master tenant:     teams/<team_uuid>/<chat>/...
--   non-master tenant: tenants/<slug>/teams/<team_uuid>/<chat>/...
-- Chat segments starting with '.' (e.g. '.goclaw') are config dirs, not real chats — skip.
-- -----------------------------------------------------------------------------
UPDATE vault_documents vd
SET chat_id = (regexp_match(vd.path, '^(?:tenants/[^/]+/)?teams/[^/]+/([^/]+)/'))[1]
FROM agent_teams t
WHERE vd.team_id = t.id
  AND (t.settings->>'workspace_scope' IS NULL OR t.settings->>'workspace_scope' != 'shared')
  AND vd.path ~ '^(?:tenants/[^/]+/)?teams/[^/]+/[^.][^/]*/';

-- -----------------------------------------------------------------------------
-- Backfill 2: legacy docs from before team scope (team_id IS NULL) with chat
-- identifiers embedded in their path. Without chat_id these leak across chats
-- in isolated-team search because the `searchChatFilter` predicate cannot
-- distinguish them.
--
-- Path layouts handled (ordered most-specific → most-general in COALESCE):
--   telegram/group_telegram_<chat>/...           (nested legacy)
--   <agent_key>/telegram/group_telegram_<chat>/. (agent-owned nested)
--   group_telegram_<chat>/...                    (bare legacy)
--   /telegram/<chat>/... or leading telegram/<chat>/...
--   tenants/<slug>/ws/<chat>/...                 (non-master tenant WS)
--   ws/<chat>/... or <agent_key>/ws/<chat>/...   (WS direct)
--   <agent_key>/delegate/<chat>/...              (delegated task)
--   <agent_key>/<botname>/group_<botname>_<chat>/... (legacy bot channel)
--   <agent_key>/<botname>/<chat>/...             (bot + numeric/ws chat)
--
-- Chat IDs can be numeric (Telegram), `system`, user handles, etc.
-- Only populate when chat_id IS NULL so interceptor-stamped values are preserved.
-- -----------------------------------------------------------------------------
UPDATE vault_documents
SET chat_id = COALESCE(
    (regexp_match(path, '^telegram/group_telegram_(-?[0-9]+)/'))[1],
    (regexp_match(path, '/telegram/group_telegram_(-?[0-9]+)/'))[1],
    (regexp_match(path, '^group_telegram_(-?[0-9]+)/'))[1],
    (regexp_match(path, '/group_telegram_(-?[0-9]+)/'))[1],
    (regexp_match(path, '^telegram/(-?[0-9a-zA-Z_-]+)/'))[1],
    (regexp_match(path, '/telegram/([a-zA-Z0-9_-]+)/'))[1],
    (regexp_match(path, '^tenants/[^/]+/ws/([^/]+)/'))[1],
    (regexp_match(path, '^ws/([^/]+)/'))[1],
    (regexp_match(path, '^[^/]+/ws/([^/]+)/'))[1],
    (regexp_match(path, '^[^/]+/delegate/([^/]+)/'))[1],
    (regexp_match(path, '^[^/]+/[^/]+/group_[^/]+_(-?[0-9]+)/'))[1],
    (regexp_match(path, '^[^/]+/[^/]+/([a-zA-Z0-9_-]+)/'))[1]
)
WHERE chat_id IS NULL
  AND team_id IS NULL
  AND (
    path ~ '(^|/)(group_telegram_|telegram/)'
    OR path ~ '^(([^/]+/)?(tenants/[^/]+/)?)?ws/[^/]+/'
    OR path ~ '^[^/]+/delegate/[^/]+/'
    OR path ~ '^[^/]+/[^/]+/(group_[^/]+_-?[0-9]+|[0-9]+)/'
  );
