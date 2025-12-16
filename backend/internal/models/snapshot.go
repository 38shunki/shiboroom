package models

import "time"

// PropertySnapshot represents a daily snapshot of a property's state
type PropertySnapshot struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PropertyID string    `gorm:"type:varchar(32);not null;index:idx_property_date" json:"property_id"`
	SnapshotAt time.Time `gorm:"type:date;not null;index:idx_property_date,priority:2;index:idx_snapshot_date" json:"snapshot_at"`

	// Property state at snapshot time
	Rent        *int     `gorm:"type:int" json:"rent,omitempty"`
	FloorPlan   string   `gorm:"type:varchar(20)" json:"floor_plan,omitempty"`
	Area        *float64 `gorm:"type:decimal(10,2)" json:"area,omitempty"`
	WalkTime    *int     `gorm:"type:int" json:"walk_time,omitempty"`
	Station     string   `gorm:"type:text" json:"station,omitempty"`
	Address     string   `gorm:"type:text" json:"address,omitempty"`
	BuildingAge *int     `gorm:"type:int" json:"building_age,omitempty"`
	Floor       *int     `gorm:"type:int" json:"floor,omitempty"`
	ImageURL    string   `gorm:"type:text" json:"image_url,omitempty"`
	Status      string   `gorm:"type:varchar(20);not null" json:"status"`

	// Change detection
	HasChanged bool   `gorm:"type:boolean;default:false" json:"has_changed"`
	ChangeNote string `gorm:"type:text" json:"change_note,omitempty"`

	CreatedAt time.Time `gorm:"type:datetime;not null;autoCreateTime" json:"created_at"`
}

// TableName specifies the table name
func (PropertySnapshot) TableName() string {
	return "property_snapshots"
}

// PropertyChange represents detected changes between snapshots
type PropertyChange struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PropertyID     string    `gorm:"type:varchar(32);not null;index" json:"property_id"`
	SnapshotID     uint      `gorm:"type:bigint;not null" json:"snapshot_id"`
	ChangeType     string    `gorm:"type:varchar(50);not null" json:"change_type"` // rent_changed, status_changed, etc.
	OldValue       string    `gorm:"type:text" json:"old_value,omitempty"`
	NewValue       string    `gorm:"type:text" json:"new_value,omitempty"`
	ChangeMagnitude *float64 `gorm:"type:decimal(10,2)" json:"change_magnitude,omitempty"` // For numerical changes
	DetectedAt     time.Time `gorm:"type:datetime;not null;autoCreateTime;index" json:"detected_at"`
}

// TableName specifies the table name
func (PropertyChange) TableName() string {
	return "property_changes"
}

// ChangeType constants
const (
	ChangeTypeRent        = "rent_changed"
	ChangeTypeStatus      = "status_changed"
	ChangeTypeArea        = "area_changed"
	ChangeTypeFloorPlan   = "floor_plan_changed"
	ChangeTypeBuildingAge = "building_age_changed"
	ChangeTypeImage       = "image_changed"
	ChangeTypeNew         = "new_property"
	ChangeTypeRemoved     = "property_removed"
)
