-- Migration 003: Change discussion_messages.agent_id and anon_id_mappings.agent_id from UUID to TEXT
-- Reason: seed agents during cold-start use non-UUID identifiers; TEXT is safer for these columns
-- which have no meaningful FK integrity (agents are sometimes synthetic/seed).

-- discussion_messages: drop FK referencing agents.id, then widen type
ALTER TABLE discussion_messages DROP CONSTRAINT IF EXISTS discussion_messages_agent_id_fkey;
ALTER TABLE discussion_messages ALTER COLUMN agent_id TYPE text USING agent_id::text;

-- anon_id_mappings: same treatment
ALTER TABLE anon.anon_id_mappings ALTER COLUMN agent_id TYPE text USING agent_id::text;
