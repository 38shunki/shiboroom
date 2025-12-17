package models

import "time"

type Property struct {
	// 基本情報
	ID               string `gorm:"type:varchar(32);primaryKey" json:"id"`
	Source           string `gorm:"type:varchar(20);not null;default:'yahoo';index:idx_source_property,priority:1" json:"source"`
	SourcePropertyID string `gorm:"type:varchar(100);not null;uniqueIndex:idx_source_property" json:"source_property_id"`
	DetailURL        string `gorm:"type:varchar(500);not null;index" json:"detail_url"`
	Title            string `gorm:"type:text;not null" json:"title"`
	ImageURL         string `gorm:"type:text" json:"image_url,omitempty"`

	// フィルタ用属性
	Rent        *int     `gorm:"type:int;index" json:"rent,omitempty"`
	FloorPlan   string   `gorm:"type:varchar(20);index" json:"floor_plan,omitempty"`
	Area        *float64 `gorm:"type:decimal(10,2)" json:"area,omitempty"`
	WalkTime    *int     `gorm:"type:int;index" json:"walk_time,omitempty"`
	Station     string   `gorm:"type:text" json:"station,omitempty"`
	Address     string   `gorm:"type:text" json:"address,omitempty"`
	BuildingAge *int     `gorm:"type:int" json:"building_age,omitempty"`
	Floor       *int     `gorm:"type:int" json:"floor,omitempty"`

	// ステータス管理（論理削除）
	Status     PropertyStatus `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	RemovedAt  *time.Time     `gorm:"type:datetime" json:"removed_at,omitempty"`
	LastSeenAt time.Time      `gorm:"type:datetime;not null;index" json:"last_seen_at"` // 最終確認日時

	// タイムスタンプ
	FetchedAt time.Time `gorm:"type:datetime;not null" json:"fetched_at"`
	CreatedAt time.Time `gorm:"type:datetime;not null;autoCreateTime;index:idx_created_at,sort:desc" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;not null;autoUpdateTime" json:"updated_at"`
}

// PropertyStatus は物件のステータス
type PropertyStatus string

const (
	PropertyStatusActive  PropertyStatus = "active"
	PropertyStatusRemoved PropertyStatus = "removed"
)

// TableName はテーブル名を明示的に指定
func (Property) TableName() string {
	return "properties"
}

// IsActive は物件がアクティブかどうか
func (p *Property) IsActive() bool {
	return p.Status == PropertyStatusActive
}

// DaysSinceLastSeen returns days since last seen
func (p *Property) DaysSinceLastSeen() int {
	return int(time.Since(p.LastSeenAt).Hours() / 24)
}

// IsLikelyExpired checks if property is likely expired (not seen for 7+ days)
func (p *Property) IsLikelyExpired() bool {
	return p.DaysSinceLastSeen() >= 7
}

// IsProbablyExpired checks if property is probably expired (not seen for 14+ days)
func (p *Property) IsProbablyExpired() bool {
	return p.DaysSinceLastSeen() >= 14
}

// UpdateLastSeen updates the last seen timestamp
func (p *Property) UpdateLastSeen() {
	p.LastSeenAt = time.Now()
}

// IsSourcePropertyIDExtracted checks if the source_property_id was extracted from URL (true)
// or is a fallback MD5 hash (false)
func (p *Property) IsSourcePropertyIDExtracted() bool {
	// Yahoo property IDs are 48 hex characters
	// MD5 hashes are 32 hex characters
	return len(p.SourcePropertyID) == 48
}

// NeedsPropertyIDRefresh checks if this property should be re-scraped to extract proper ID
func (p *Property) NeedsPropertyIDRefresh() bool {
	return !p.IsSourcePropertyIDExtracted() && p.Source == "yahoo"
}

// MarkAsRemoved は物件を論理削除
func (p *Property) MarkAsRemoved() {
	p.Status = PropertyStatusRemoved
	now := time.Now()
	p.RemovedAt = &now
}
