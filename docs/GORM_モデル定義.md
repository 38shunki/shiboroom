# GORM モデル定義 - MySQL 8.x対応

## 概要

PostgreSQLからMySQL 8.xへの移行に伴うGORMモデル定義。
論理削除、差分検出、エリア管理をサポート。

---

## 1. Property（物件マスタ）

```go
package models

import (
    "time"
    "gorm.io/gorm"
)

// Property は物件情報を表すモデル
type Property struct {
    // 基本情報
    ID        string    `gorm:"type:varchar(32);primaryKey" json:"id"`
    DetailURL string    `gorm:"type:text;not null;uniqueIndex" json:"detail_url"`
    Title     string    `gorm:"type:text;not null" json:"title"`
    ImageURL  string    `gorm:"type:text" json:"image_url,omitempty"`

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
    CreatedAt time.Time `gorm:"type:datetime;not null;autoCreateTime" json:"created_at"`
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

// BeforeCreate はID生成のフック
func (p *Property) BeforeCreate(tx *gorm.DB) error {
    // IDが未設定の場合、URL正規化してMD5生成
    if p.ID == "" {
        normalizedURL := normalizeURL(p.DetailURL)
        p.ID = generateMD5(normalizedURL)
    }
    return nil
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
```

---

## 2. DailyPropertySnapshot（日次スナップショット）

```go
package models

import (
    "time"
)

// DailyPropertySnapshot は日次の物件存在確認用スナップショット
type DailyPropertySnapshot struct {
    SnapshotDate string `gorm:"type:date;not null;primaryKey" json:"snapshot_date"`
    PropertyID   string `gorm:"type:varchar(32);not null;primaryKey" json:"property_id"`
    DetailURL    string `gorm:"type:text;not null" json:"detail_url"`
    AreaCode     string `gorm:"type:varchar(20);index" json:"area_code"`
}

// TableName はテーブル名を明示的に指定
func (DailyPropertySnapshot) TableName() string {
    return "daily_property_snapshots"
}

// Composite Primary Key の定義
func (DailyPropertySnapshot) CompositePrimaryKey() []string {
    return []string{"snapshot_date", "property_id"}
}
```

---

## 3. ScrapeArea（スクレイピング対象エリア）

```go
package models

import (
    "time"
)

// ScrapeArea はスクレイピング対象エリア
type ScrapeArea struct {
    AreaCode      string          `gorm:"type:varchar(20);primaryKey" json:"area_code"`
    Name          string          `gorm:"type:varchar(100);not null" json:"name"`
    Prefecture    string          `gorm:"type:varchar(50)" json:"prefecture"`
    City          string          `gorm:"type:varchar(50)" json:"city"`
    SearchURL     string          `gorm:"type:text" json:"search_url"`

    Enabled       bool            `gorm:"type:boolean;not null;default:true;index" json:"enabled"`
    LastScrapedAt *time.Time      `gorm:"type:datetime;index" json:"last_scraped_at,omitempty"`
    LastStatus    ScrapeStatus    `gorm:"type:varchar(20);default:'pending'" json:"last_status"`

    CreatedAt     time.Time       `gorm:"type:datetime;not null;autoCreateTime" json:"created_at"`
    UpdatedAt     time.Time       `gorm:"type:datetime;not null;autoUpdateTime" json:"updated_at"`
}

// ScrapeStatus はスクレイピング実行状態
type ScrapeStatus string

const (
    ScrapeStatusPending ScrapeStatus = "pending"
    ScrapeStatusSuccess ScrapeStatus = "success"
    ScrapeStatusFailed  ScrapeStatus = "failed"
)

// TableName はテーブル名を明示的に指定
func (ScrapeArea) TableName() string {
    return "scrape_areas"
}

// UpdateStatus はステータスを更新
func (s *ScrapeArea) UpdateStatus(status ScrapeStatus) {
    s.LastStatus = status
    now := time.Now()
    s.LastScrapedAt = &now
}
```

---

## 4. ScrapeLog（スクレイピング実行ログ）

```go
package models

import (
    "time"
)

// ScrapeLog はスクレイピング実行ログ
type ScrapeLog struct {
    ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
    ExecutionDate string         `gorm:"type:date;not null;index" json:"execution_date"`
    AreaCode      *string        `gorm:"type:varchar(20);index" json:"area_code,omitempty"`
    Phase         ScrapePhase    `gorm:"type:varchar(20);not null" json:"phase"`

    Status        LogStatus      `gorm:"type:varchar(20);not null;index" json:"status"`
    TotalCount    *int           `gorm:"type:int" json:"total_count,omitempty"`
    SuccessCount  *int           `gorm:"type:int" json:"success_count,omitempty"`
    ErrorCount    *int           `gorm:"type:int" json:"error_count,omitempty"`

    ErrorMessage  string         `gorm:"type:text" json:"error_message,omitempty"`
    StartedAt     time.Time      `gorm:"type:datetime;not null" json:"started_at"`
    CompletedAt   *time.Time     `gorm:"type:datetime" json:"completed_at,omitempty"`
}

// ScrapePhase はスクレイピングのフェーズ
type ScrapePhase string

const (
    ScrapePhaseList    ScrapePhase = "list"      // 一覧取得
    ScrapePhaseDetail  ScrapePhase = "detail"    // 詳細取得
    ScrapePhaseDiff    ScrapePhase = "diff"      // 差分判定
    ScrapePhaseRemoval ScrapePhase = "removal"   // 削除処理
)

// LogStatus はログステータス
type LogStatus string

const (
    LogStatusStarted   LogStatus = "started"
    LogStatusCompleted LogStatus = "completed"
    LogStatusFailed    LogStatus = "failed"
    LogStatusPartial   LogStatus = "partial"  // 一部失敗
)

// TableName はテーブル名を明示的に指定
func (ScrapeLog) TableName() string {
    return "scrape_logs"
}

// Complete はログを完了状態に更新
func (s *ScrapeLog) Complete() {
    s.Status = LogStatusCompleted
    now := time.Now()
    s.CompletedAt = &now
}

// Fail はログを失敗状態に更新
func (s *ScrapeLog) Fail(err error) {
    s.Status = LogStatusFailed
    s.ErrorMessage = err.Error()
    now := time.Now()
    s.CompletedAt = &now
}
```

---

## マイグレーション

### AutoMigrate使用例

```go
package database

import (
    "gorm.io/gorm"
    "real-estate-portal/internal/models"
)

// MigrateAll はすべてのテーブルをマイグレーション
func MigrateAll(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.Property{},
        &models.DailyPropertySnapshot{},
        &models.ScrapeArea{},
        &models.ScrapeLog{},
    )
}
```

### golang-migrate使用例

```sql
-- migrations/000001_create_properties.up.sql
CREATE TABLE properties (
    id VARCHAR(32) PRIMARY KEY,
    detail_url TEXT NOT NULL,
    title TEXT NOT NULL,
    image_url TEXT,

    rent INT,
    floor_plan VARCHAR(20),
    area DECIMAL(10, 2),
    walk_time INT,
    station TEXT,
    address TEXT,
    building_age INT,
    floor INT,

    status VARCHAR(20) NOT NULL DEFAULT 'active',
    removed_at DATETIME,

    fetched_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_detail_url (detail_url(255)),
    INDEX idx_status (status),
    INDEX idx_rent (rent),
    INDEX idx_floor_plan (floor_plan),
    INDEX idx_walk_time (walk_time),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## リポジトリパターン実装例

```go
package repository

import (
    "context"
    "time"
    "gorm.io/gorm"
    "real-estate-portal/internal/models"
)

type PropertyRepository struct {
    db *gorm.DB
}

func NewPropertyRepository(db *gorm.DB) *PropertyRepository {
    return &PropertyRepository{db: db}
}

// GetActiveProperties はアクティブな物件一覧を取得
func (r *PropertyRepository) GetActiveProperties(ctx context.Context) ([]models.Property, error) {
    var properties []models.Property
    err := r.db.WithContext(ctx).
        Where("status = ?", models.PropertyStatusActive).
        Order("created_at DESC").
        Find(&properties).Error
    return properties, err
}

// SaveProperty は物件を保存（Upsert）
func (r *PropertyRepository) SaveProperty(ctx context.Context, property *models.Property) error {
    return r.db.WithContext(ctx).
        Clauses(clause.OnConflict{
            Columns:   []clause.Column{{Name: "detail_url"}},
            DoUpdates: clause.AssignmentColumns([]string{
                "title", "image_url", "rent", "floor_plan",
                "area", "walk_time", "station", "address",
                "building_age", "floor", "fetched_at", "updated_at",
            }),
        }).
        Create(property).Error
}

// MarkAsRemoved は物件を論理削除
func (r *PropertyRepository) MarkAsRemoved(ctx context.Context, propertyIDs []string) error {
    now := time.Now()
    return r.db.WithContext(ctx).
        Model(&models.Property{}).
        Where("id IN ?", propertyIDs).
        Updates(map[string]interface{}{
            "status":     models.PropertyStatusRemoved,
            "removed_at": now,
        }).Error
}
```

---

## スナップショットリポジトリ

```go
package repository

import (
    "context"
    "gorm.io/gorm"
    "real-estate-portal/internal/models"
)

type SnapshotRepository struct {
    db *gorm.DB
}

func NewSnapshotRepository(db *gorm.DB) *SnapshotRepository {
    return &SnapshotRepository{db: db}
}

// SaveSnapshots はスナップショットを一括保存
func (r *SnapshotRepository) SaveSnapshots(ctx context.Context, snapshots []models.DailyPropertySnapshot) error {
    return r.db.WithContext(ctx).
        CreateInBatches(snapshots, 1000).Error
}

// GetSnapshotIDs は指定日のスナップショットIDを取得
func (r *SnapshotRepository) GetSnapshotIDs(ctx context.Context, date string) ([]string, error) {
    var ids []string
    err := r.db.WithContext(ctx).
        Model(&models.DailyPropertySnapshot{}).
        Where("snapshot_date = ?", date).
        Pluck("property_id", &ids).Error
    return ids, err
}

// GetDifference は昨日と今日の差分を取得
func (r *SnapshotRepository) GetDifference(ctx context.Context, today, yesterday string) (new, removed []string, err error) {
    // 新規物件
    err = r.db.WithContext(ctx).Raw(`
        SELECT t.property_id
        FROM daily_property_snapshots t
        LEFT JOIN daily_property_snapshots y
          ON t.property_id = y.property_id
          AND y.snapshot_date = ?
        WHERE t.snapshot_date = ?
          AND y.property_id IS NULL
    `, yesterday, today).Pluck("property_id", &new).Error

    if err != nil {
        return nil, nil, err
    }

    // 消滅物件
    err = r.db.WithContext(ctx).Raw(`
        SELECT y.property_id
        FROM daily_property_snapshots y
        LEFT JOIN daily_property_snapshots t
          ON y.property_id = t.property_id
          AND t.snapshot_date = ?
        WHERE y.snapshot_date = ?
          AND t.property_id IS NULL
    `, today, yesterday).Pluck("property_id", &removed).Error

    return new, removed, err
}
```

---

## 設定ファイル例

```go
package config

import (
    "fmt"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

type DBConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    Database string
}

func NewDB(config *DBConfig) (*gorm.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        config.User,
        config.Password,
        config.Host,
        config.Port,
        config.Database,
    )

    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    })

    if err != nil {
        return nil, fmt.Errorf("DB接続エラー: %w", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    // コネクションプール設定
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)

    return db, nil
}
```

---

## 注意事項

### ENUM vs VARCHAR

GORMでのENUMは扱いづらいため、VARCHARで実装しています。
アプリケーション層でバリデーションを行います。

```go
// バリデーション例
func (s PropertyStatus) IsValid() bool {
    switch s {
    case PropertyStatusActive, PropertyStatusRemoved:
        return true
    }
    return false
}
```

### TEXT型のインデックス

MySQLではTEXT型に直接インデックスを張れないため、
`detail_url` の uniqueIndexは最初の255文字のみを対象にしています。

```go
gorm:"uniqueIndex:idx_detail_url,length:255"
```

### 日付型

`snapshot_date` はDATE型ですが、Go側では `string` で扱います（"2006-01-02"形式）。

---

## 次のステップ

1. **マイグレーションファイル作成**
2. **リポジトリ層の完全実装**
3. **サービス層の実装**（ビジネスロジック）
4. **統合テスト**

---

**最終更新**: 2025-12-16
