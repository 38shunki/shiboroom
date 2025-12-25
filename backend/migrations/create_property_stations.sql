-- Create property_stations table for storing multiple station access information
CREATE TABLE IF NOT EXISTS property_stations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    property_id VARCHAR(32) NOT NULL,
    station_name VARCHAR(255) NOT NULL,
    line_name VARCHAR(255) NOT NULL,
    walk_minutes INT NOT NULL,
    sort_order INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- Foreign key constraint
    FOREIGN KEY (property_id) REFERENCES properties(id) ON DELETE CASCADE,

    -- Index for common queries
    INDEX idx_property_id (property_id),
    INDEX idx_station_name (station_name),
    INDEX idx_line_name (line_name),
    INDEX idx_walk_minutes (walk_minutes),
    INDEX idx_sort_order (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
