-- Add last_seen_at column to properties table
ALTER TABLE properties ADD COLUMN IF NOT EXISTS last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE properties ADD INDEX IF NOT EXISTS idx_last_seen_at (last_seen_at);

-- Create scraping_state table for blocking management
CREATE TABLE IF NOT EXISTS scraping_state (
    id INT AUTO_INCREMENT PRIMARY KEY,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    blocked_until DATETIME NULL,
    blocked_reason TEXT NULL,
    last_attempt DATETIME NOT NULL,
    last_success DATETIME NULL,
    failure_count INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_blocked_until (blocked_until)
);

-- Insert initial scraping state
INSERT INTO scraping_state (id, last_attempt, is_blocked)
VALUES (1, CURRENT_TIMESTAMP, TRUE)
ON DUPLICATE KEY UPDATE id = id;

-- Update existing properties to set last_seen_at
UPDATE properties SET last_seen_at = updated_at WHERE last_seen_at IS NULL OR last_seen_at = '0000-00-00 00:00:00';
