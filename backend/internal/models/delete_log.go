package models

import "time"

// DeleteLog represents a record of physically deleted properties
type DeleteLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PropertyID string    `gorm:"type:varchar(32);not null;index" json:"property_id"`
	Title      string    `gorm:"type:text" json:"title"`
	DetailURL  string    `gorm:"type:text" json:"detail_url"`
	RemovedAt  time.Time `gorm:"type:datetime" json:"removed_at"`
	DeletedAt  time.Time `gorm:"type:datetime;not null;autoCreateTime;index" json:"deleted_at"`
	Reason     string    `gorm:"type:varchar(50);not null" json:"reason"`
}

// TableName specifies the table name
func (DeleteLog) TableName() string {
	return "delete_logs"
}

// DeleteReason constants
const (
	DeleteReasonExpired    = "expired_90_days"
	DeleteReasonDuplicate  = "duplicate"
	DeleteReasonManual     = "manual_deletion"
	DeleteReasonDataClean  = "data_cleanup"
)
