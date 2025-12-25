package models

import "time"

// PropertyStation represents a station access point for a property
type PropertyStation struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PropertyID   string    `gorm:"type:varchar(32);not null;index:idx_property_id" json:"property_id"`
	StationName  string    `gorm:"type:varchar(255);not null;index:idx_station_name" json:"station_name"`
	LineName     string    `gorm:"type:varchar(255);not null;index:idx_line_name" json:"line_name"`
	WalkMinutes  int       `gorm:"type:int;not null;index:idx_walk_minutes" json:"walk_minutes"`
	SortOrder    int       `gorm:"type:int;not null;default:1;index:idx_sort_order" json:"sort_order"` // 1=最寄り, 2=2番目, etc.
	CreatedAt    time.Time `gorm:"type:timestamp;not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:timestamp;not null;autoUpdateTime" json:"updated_at"`

	// Relationship
	Property Property `gorm:"foreignKey:PropertyID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name
func (PropertyStation) TableName() string {
	return "property_stations"
}

// IsPrimary returns true if this is the primary (nearest) station
func (ps *PropertyStation) IsPrimary() bool {
	return ps.SortOrder == 1
}
