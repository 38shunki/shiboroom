-- Separate マンション and アパート into distinct building types
-- Previously they were both normalized to 'apartment', now mansion = 'mansion', apartment = 'apartment'

-- First, let's see what we have (for logging)
-- This migration will separate them based on the original scraping data
-- Since we don't have the original Japanese data stored, we need to re-scrape
-- For now, let's just document that existing 'apartment' entries could be either mansion or apartment
-- and will be properly classified on next scrape

-- Note: This migration doesn't change existing data because we don't know which 'apartment'
-- entries were originally マンション vs アパート. The normalization function change will
-- ensure all NEW scrapes are properly classified.

-- Future scrapes will use:
-- マンション → mansion
-- アパート → apartment
