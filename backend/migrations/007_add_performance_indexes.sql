-- Add performance indexes for filtering and sorting
-- Migration 007: Performance optimization indexes

-- Single column indexes for range filters
CREATE INDEX IF NOT EXISTS idx_properties_area ON properties(area);
CREATE INDEX IF NOT EXISTS idx_properties_building_age ON properties(building_age);
CREATE INDEX IF NOT EXISTS idx_properties_floor ON properties(floor);

-- Composite index for common query pattern: active properties sorted by creation date
-- (status, created_at DESC, id) for efficient pagination with cursor
CREATE INDEX IF NOT EXISTS idx_properties_status_created_id ON properties(status, created_at DESC, id);

-- Composite index for rent-based filtering (common use case)
-- (status, rent, id) for rent range queries with cursor
CREATE INDEX IF NOT EXISTS idx_properties_status_rent_id ON properties(status, rent, id);

-- property_stations indexes for efficient station/line filtering
-- Covers EXISTS queries from properties
CREATE INDEX IF NOT EXISTS idx_property_stations_property_id ON property_stations(property_id);
CREATE INDEX IF NOT EXISTS idx_property_stations_station_property ON property_stations(station_name, property_id);
CREATE INDEX IF NOT EXISTS idx_property_stations_line_property ON property_stations(line_name, property_id);
CREATE INDEX IF NOT EXISTS idx_property_stations_walk_property ON property_stations(walk_minutes, property_id);

-- Composite index for station filtering with walk time
CREATE INDEX IF NOT EXISTS idx_property_stations_composite ON property_stations(station_name, line_name, walk_minutes, property_id);
