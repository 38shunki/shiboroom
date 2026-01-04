package models

import "time"

// PropertyImage represents an image associated with a property
type PropertyImage struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PropertyID string    `gorm:"type:varchar(64);not null;index" json:"property_id"`
	ImageURL   string    `gorm:"type:text;not null" json:"image_url"`
	SortOrder  int       `gorm:"not null;default:0;index" json:"sort_order"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for PropertyImage
func (PropertyImage) TableName() string {
	return "property_images"
}
