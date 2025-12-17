-- Migration: Add source and source_property_id for stable property identification
-- This enables reliable differential scraping and multi-source support

-- Step 1: Add new columns (nullable initially for safe migration)
ALTER TABLE properties
ADD COLUMN IF NOT EXISTS source VARCHAR(20) DEFAULT 'yahoo',
ADD COLUMN IF NOT EXISTS source_property_id VARCHAR(100) DEFAULT NULL;

-- Step 2: Add indexes for performance (before backfill to help with queries)
ALTER TABLE properties ADD INDEX IF NOT EXISTS idx_source (source);

-- Step 3: Backfill source_property_id from existing detail_url
-- Extracts Yahoo property ID from URLs like:
-- https://realestate.yahoo.co.jp/rent/detail/000008250678c0a0c9accff94eab13c4c687966f0698
UPDATE properties
SET source_property_id = SUBSTRING_INDEX(SUBSTRING_INDEX(detail_url, '/detail/', -1), '?', 1)
WHERE source_property_id IS NULL
  AND detail_url LIKE '%realestate.yahoo.co.jp/rent/detail/%'
  AND CHAR_LENGTH(SUBSTRING_INDEX(SUBSTRING_INDEX(detail_url, '/detail/', -1), '?', 1)) = 48;

-- Step 4: For any URLs that couldn't be parsed (non-standard format),
-- use MD5 hash of detail_url as fallback (same as old behavior)
UPDATE properties
SET source_property_id = MD5(detail_url)
WHERE source_property_id IS NULL OR source_property_id = '';

-- Step 5: Now make source_property_id NOT NULL (all rows should have values)
ALTER TABLE properties
MODIFY COLUMN source VARCHAR(20) NOT NULL DEFAULT 'yahoo',
MODIFY COLUMN source_property_id VARCHAR(100) NOT NULL;

-- Step 6: Add UNIQUE constraint on (source, source_property_id)
-- This is the core uniqueness guarantee for differential scraping
ALTER TABLE properties
ADD UNIQUE INDEX IF NOT EXISTS idx_source_property (source, source_property_id);

-- Step 7: Remove old uniqueIndex on detail_url (keep as regular index for lookups)
-- Note: We keep detail_url indexed for debugging and URL-based lookups
ALTER TABLE properties DROP INDEX IF EXISTS detail_url;
ALTER TABLE properties ADD INDEX IF NOT EXISTS idx_detail_url (detail_url);

-- Step 8: Verify migration success
-- Count properties with valid Yahoo property IDs (48 hex characters)
SELECT
    'Migration Summary' AS status,
    COUNT(*) AS total_properties,
    SUM(CASE WHEN CHAR_LENGTH(source_property_id) = 48 THEN 1 ELSE 0 END) AS yahoo_id_extracted,
    SUM(CASE WHEN CHAR_LENGTH(source_property_id) = 32 THEN 1 ELSE 0 END) AS fallback_md5_hash,
    SUM(CASE WHEN source_property_id IS NULL THEN 1 ELSE 0 END) AS null_count
FROM properties;

-- Check for any duplicate (source, source_property_id) pairs before constraint
-- If this returns rows, manual cleanup is needed
SELECT source, source_property_id, COUNT(*) as duplicate_count
FROM properties
GROUP BY source, source_property_id
HAVING COUNT(*) > 1;

-- Identify fallback MD5 properties for future re-scraping
-- Yahoo IDs are 48 characters, MD5 hashes are 32 characters
SELECT
    'Fallback MD5 Properties' AS status,
    COUNT(*) AS total,
    MIN(created_at) AS oldest,
    MAX(created_at) AS newest
FROM properties
WHERE CHAR_LENGTH(source_property_id) = 32;

-- Show sample fallback properties
SELECT id, source, source_property_id, detail_url, created_at
FROM properties
WHERE CHAR_LENGTH(source_property_id) = 32
LIMIT 5;
