-- Migration: Create detail_scrape_queue table
-- Purpose: Manage pending detail page scrapes with retry logic
-- Date: 2025-12-18

CREATE TABLE IF NOT EXISTS detail_scrape_queue (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    source VARCHAR(50) NOT NULL,
    source_property_id VARCHAR(255) NOT NULL,
    detail_url TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    priority INT DEFAULT 0,
    attempts INT DEFAULT 0,
    last_error TEXT,
    next_retry_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,

    INDEX idx_queue_lookup (source, source_property_id),
    INDEX idx_status (status),
    INDEX idx_priority (priority),
    INDEX idx_retry (next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Note: Unique constraint on (source, source_property_id, status) is enforced in application logic
-- MySQL 5.7 doesn't support partial indexes, so we use application-level checking instead
