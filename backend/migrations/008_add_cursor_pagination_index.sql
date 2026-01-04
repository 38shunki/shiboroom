-- Add composite index for cursor pagination on fetched_at + id
-- This index supports efficient cursor-based pagination for sort=newest

-- Create composite index for cursor pagination (fetched_at DESC, id DESC)
CREATE INDEX idx_cursor_newest ON properties(status, fetched_at DESC, id DESC);

-- Note: Including status first allows efficient filtering by status='active' before cursor navigation
