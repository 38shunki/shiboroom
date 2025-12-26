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
	Rent              *int     `gorm:"type:int;index" json:"rent,omitempty"`
	FloorPlan         string   `gorm:"type:varchar(20);index" json:"floor_plan,omitempty"`
	Area              *float64 `gorm:"type:decimal(10,2)" json:"area,omitempty"`
	WalkTime          *int     `gorm:"type:int;index" json:"walk_time,omitempty"`
	Station           string   `gorm:"type:text" json:"station,omitempty"`
	Address           string   `gorm:"type:text" json:"address,omitempty"`
	BuildingAge       *int     `gorm:"type:int" json:"building_age,omitempty"`
	Floor             *int     `gorm:"type:int" json:"floor,omitempty"`
	BuildingType      string   `gorm:"type:varchar(50);index" json:"building_type"`                // マンション/アパート/一戸建て
	Structure         string   `gorm:"type:varchar(50)" json:"structure"`                          // 鉄筋コンクリート/軽量鉄骨等
	Facilities        string   `gorm:"type:text" json:"facilities"`                                // こだわり条件(JSON配列形式)
	Features          string   `gorm:"type:text" json:"features"`                                  // ピックアウト特徴(JSON配列形式)
	BuildingName      string   `gorm:"type:varchar(255)" json:"building_name,omitempty"`           // 建物名
	Direction         string   `gorm:"type:varchar(50)" json:"direction,omitempty"`                // 方位
	FloorPlanDetails  string   `gorm:"type:text" json:"floor_plan_details"`                        // 間取り内訳
	FloorLabel        string   `gorm:"type:varchar(100)" json:"floor_label"`             // 階数情報（例: 地上3階建て/2階部分）
	Parking           string   `gorm:"type:varchar(255)" json:"parking"`                           // 駐車場
	ContractPeriod    string   `gorm:"type:varchar(50)" json:"contract_period"`                    // 契約期間
	Insurance         string   `gorm:"type:varchar(255)" json:"insurance"`                         // 保険
	RoomLayoutImageURL string  `gorm:"type:text" json:"room_layout_image_url,omitempty"`          // 間取り図URL

	// 契約・費用情報
	ManagementFee     string `gorm:"type:varchar(100)" json:"management_fee,omitempty"`      // 管理費・共益費
	Deposit           string `gorm:"type:varchar(100)" json:"deposit,omitempty"`             // 敷金
	KeyMoney          string `gorm:"type:varchar(100)" json:"key_money,omitempty"`           // 礼金
	GuarantorDeposit  string `gorm:"type:varchar(100)" json:"guarantor_deposit,omitempty"`   // 保証金
	SecurityDeposit   string `gorm:"type:varchar(100)" json:"security_deposit,omitempty"`    // 敷引
	MoveInDate        string `gorm:"type:varchar(100)" json:"move_in_date,omitempty"`        // 入居可能時期
	Conditions        string `gorm:"type:varchar(255)" json:"conditions,omitempty"`          // 条件等
	Notes             string `gorm:"type:text" json:"notes,omitempty"`                       // 備考（初期費用詳細など）

	// ステータス管理（論理削除）
	Status     PropertyStatus `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	RemovedAt  *time.Time     `gorm:"type:datetime" json:"removed_at,omitempty"`
	LastSeenAt *time.Time     `gorm:"type:datetime;index" json:"last_seen_at,omitempty"` // 最終確認日時

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
	if p.LastSeenAt == nil {
		return 9999 // Never seen
	}
	return int(time.Since(*p.LastSeenAt).Hours() / 24)
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
	now := time.Now()
	p.LastSeenAt = &now
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
