-- Migration: Add importance_tier to topic_verses
-- Created: 2026-01-27
-- Purpose: Allow prioritization of verses within topics (1=essential, 2=important, 3=supporting)

--------------------------------------------------------------------------------
-- Add importance_tier column to topic_verses
--------------------------------------------------------------------------------
ALTER TABLE api.topic_verses
ADD COLUMN IF NOT EXISTS importance_tier SMALLINT DEFAULT 3;

-- Add comment explaining the column
COMMENT ON COLUMN api.topic_verses.importance_tier IS
    'Verse importance within topic: 1=essential, 2=important, 3=supporting';

-- Create index for efficient ordering by importance
CREATE INDEX IF NOT EXISTS idx_topic_verses_importance
    ON api.topic_verses (topic_id, importance_tier);

--------------------------------------------------------------------------------
-- Usage notes:
-- Queries should ORDER BY importance_tier, then book_order/chapter/verse
-- Example:
--   SELECT v.osis_verse_id, tv.importance_tier
--   FROM api.topic_verses tv
--   JOIN api.verses v ON tv.verse_id = v.id
--   JOIN api.books b ON v.book_id = b.id
--   WHERE tv.topic_id = $1
--   ORDER BY tv.importance_tier, b.book_order, v.chapter, v.verse
--------------------------------------------------------------------------------
