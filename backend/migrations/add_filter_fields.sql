-- Add new filter fields to properties table
-- These fields are all optional and will not affect existing data

ALTER TABLE properties 
ADD COLUMN IF NOT EXISTS building_type VARCHAR(50) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS structure VARCHAR(50) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS facilities TEXT DEFAULT NULL,
ADD COLUMN IF NOT EXISTS features TEXT DEFAULT NULL;

-- Add index on building_type for faster filtering
CREATE INDEX IF NOT EXISTS idx_properties_building_type ON properties(building_type);
