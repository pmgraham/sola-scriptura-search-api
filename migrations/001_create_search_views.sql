-- Migration: Create search-specific materialized views
-- Created: 2026-01-26
-- Purpose: Pre-join data for vector search operations

--------------------------------------------------------------------------------
-- MV: mv_verses_search
-- Verses with embeddings and book info for vector similarity search
--------------------------------------------------------------------------------
CREATE MATERIALIZED VIEW api_views.mv_verses_search AS
SELECT
    v.osis_verse_id AS verse_id,
    b.osis_id AS book,
    b.book_order,
    v.chapter,
    v.verse,
    v.text,
    v.embedding
FROM api.verses v
JOIN api.books b ON v.book_id = b.id
WHERE v.embedding IS NOT NULL;

-- Create vector index for similarity search
CREATE INDEX idx_mv_verses_search_embedding ON api_views.mv_verses_search
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_mv_verses_search_verse_id ON api_views.mv_verses_search (verse_id);

--------------------------------------------------------------------------------
-- Refresh notes:
-- REFRESH MATERIALIZED VIEW CONCURRENTLY api_views.mv_verses_search;
-- Note: CONCURRENTLY requires a unique index, add if needed:
-- CREATE UNIQUE INDEX idx_mv_verses_search_verse_id_unique ON api_views.mv_verses_search (verse_id);
