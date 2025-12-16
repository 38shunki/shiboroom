package models

import "time"

type Property struct {
	// 基本情報
	ID        string `gorm:"type:varchar(32);primaryKey" json:"id"`
	DetailURL string `gorm:"type:varchar(500);not null;uniqueIndex" json:"detail_url"`
	Title     string `gorm:"type:text;not null" json:"title"`
	ImageURL  string `gorm:"type:text" json:"image_url,omitempty"`

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
	Status    PropertyStatus `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	RemovedAt *time.Time     `gorm:"type:datetime" json:"removed_at,omitempty"`

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

// MarkAsRemoved は物件を論理削除
func (p *Property) MarkAsRemoved() {
	p.Status = PropertyStatusRemoved
	now := time.Now()
	p.RemovedAt = &now
}
