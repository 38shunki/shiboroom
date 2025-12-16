# ER図 - 不動産ポータル データベース設計

## 概要

毎日の差分更新を前提としたデータベース設計。
論理削除により検索結果から消滅物件を除外しつつ、過去データを保持する。

---

## テーブル一覧

### 1. properties（物件マスタ）
### 2. daily_property_snapshots（日次スナップショット）
### 3. scrape_areas（スクレイピング対象エリア）
### 4. scrape_logs（スクレイピング実行ログ）
### 5. delete_logs（物理削除ログ）※Phase 4.5で追加

---

## 1. properties（物件マスタ）

**目的**: 実際の物件データを保存。検索・表示の正。

```sql
CREATE TABLE properties (
  -- 基本情報
  id              VARCHAR(32) PRIMARY KEY,  -- MD5(正規化URL)
  detail_url      TEXT NOT NULL UNIQUE,     -- 正規化済みURL
  title           TEXT NOT NULL,
  image_url       TEXT,

  -- フィルタ用属性
  rent            INT,                      -- 賃料（円）
  floor_plan      VARCHAR(20),              -- 間取り（1K, 1LDKなど）
  area            DECIMAL(10, 2),           -- 面積（㎡）
  walk_time       INT,                      -- 駅徒歩（分）
  station         TEXT,                     -- 駅名
  address         TEXT,                     -- 住所
  building_age    INT,                      -- 築年数
  floor           INT,                      -- 階数

  -- ステータス管理
  status          ENUM('active', 'removed') NOT NULL DEFAULT 'active',
  removed_at      TIMESTAMP NULL,

  -- タイムスタンプ
  fetched_at      TIMESTAMP NOT NULL,       -- 最終取得日時
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_status (status),
  INDEX idx_rent (rent),
  INDEX idx_floor_plan (floor_plan),
  INDEX idx_walk_time (walk_time),
  INDEX idx_created_at (created_at)
);
```

**カラム説明**:

| カラム | 型 | NULL | 説明 |
|-------|---|------|-----|
| id | VARCHAR(32) | NO | MD5ハッシュ値。正規化URLから生成 |
| detail_url | TEXT | NO | 正規化済みのYahoo不動産URL（UNIQUE制約） |
| status | ENUM | NO | active: 検索対象 / removed: 消滅済み |
| removed_at | TIMESTAMP | YES | 消滅検出日時 |
| fetched_at | TIMESTAMP | NO | 最後にスクレイピングした日時 |
| created_at | TIMESTAMP | NO | 初回登録日時（固定） |

**論理削除の仕組み**:
- 物件が一覧から消えたら `status = 'removed'`
- 検索時は `WHERE status = 'active'` で絞る
- Meilisearchからも即座に削除

---

## 2. daily_property_snapshots（日次スナップショット）

**目的**: 毎日の「存在する物件ID一覧」を記録。差分検出に使用。

```sql
CREATE TABLE daily_property_snapshots (
  snapshot_date   DATE NOT NULL,            -- スナップショット取得日
  property_id     VARCHAR(32) NOT NULL,     -- 物件ID（properties.id）
  detail_url      TEXT NOT NULL,            -- 物件URL
  area_code       VARCHAR(20),              -- エリアコード

  PRIMARY KEY (snapshot_date, property_id),
  INDEX idx_snapshot_date (snapshot_date),
  INDEX idx_property_id (property_id)
);
```

**カラム説明**:

| カラム | 型 | NULL | 説明 |
|-------|---|------|-----|
| snapshot_date | DATE | NO | スナップショット取得日（例: 2025-12-16） |
| property_id | VARCHAR(32) | NO | 物件ID（properties.idと一致） |
| detail_url | TEXT | NO | 物件の詳細URL |
| area_code | VARCHAR(20) | YES | エリアコード（scrape_areas.area_code） |

**使用例**:

```sql
-- 昨日と今日を比較して新規物件を検出
SELECT t.property_id
FROM daily_property_snapshots t
LEFT JOIN daily_property_snapshots y
  ON t.property_id = y.property_id
  AND y.snapshot_date = '2025-12-15'
WHERE t.snapshot_date = '2025-12-16'
  AND y.property_id IS NULL;

-- 昨日と今日を比較して消滅物件を検出
SELECT y.property_id
FROM daily_property_snapshots y
LEFT JOIN daily_property_snapshots t
  ON y.property_id = t.property_id
  AND t.snapshot_date = '2025-12-16'
WHERE y.snapshot_date = '2025-12-15'
  AND t.property_id IS NULL;
```

**保持期間**: 最低7日分（差分検出・障害復旧用）

---

## 3. scrape_areas（スクレイピング対象エリア）

**目的**: エリア単位でスクレイピングを分割管理。

```sql
CREATE TABLE scrape_areas (
  area_code       VARCHAR(20) PRIMARY KEY,  -- エリアコード（例: tokyo_shinjuku）
  name            VARCHAR(100) NOT NULL,    -- 表示名（例: 東京都新宿区）
  prefecture      VARCHAR(50),              -- 都道府県
  city            VARCHAR(50),              -- 市区町村
  search_url      TEXT,                     -- Yahoo不動産検索URL

  enabled         BOOLEAN NOT NULL DEFAULT true,  -- 有効/無効
  last_scraped_at TIMESTAMP NULL,           -- 最終実行日時
  last_status     ENUM('success', 'failed', 'pending') DEFAULT 'pending',

  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_enabled (enabled),
  INDEX idx_last_scraped_at (last_scraped_at)
);
```

**カラム説明**:

| カラム | 型 | NULL | 説明 |
|-------|---|------|-----|
| area_code | VARCHAR(20) | NO | 一意なエリアコード |
| name | VARCHAR(100) | NO | 表示名（例: 東京都新宿区） |
| search_url | TEXT | YES | Yahoo不動産の検索URL |
| enabled | BOOLEAN | NO | false にするとスキップ |
| last_scraped_at | TIMESTAMP | YES | 最終実行日時 |
| last_status | ENUM | YES | 最後の実行結果 |

**エリア例**:

```sql
INSERT INTO scrape_areas (area_code, name, prefecture, city, search_url) VALUES
('tokyo_shinjuku', '東京都新宿区', '東京都', '新宿区', 'https://realestate.yahoo.co.jp/rent/search/...'),
('tokyo_shibuya', '東京都渋谷区', '東京都', '渋谷区', 'https://realestate.yahoo.co.jp/rent/search/...'),
('osaka_chuo', '大阪府中央区', '大阪府', '中央区', 'https://realestate.yahoo.co.jp/rent/search/...');
```

---

## 4. scrape_logs（スクレイピング実行ログ）

**目的**: スクレイピングの実行履歴を記録。

```sql
CREATE TABLE scrape_logs (
  id              BIGINT AUTO_INCREMENT PRIMARY KEY,
  execution_date  DATE NOT NULL,            -- 実行日
  area_code       VARCHAR(20),              -- エリアコード（NULL = 全体）
  phase           ENUM('list', 'detail', 'diff') NOT NULL,  -- フェーズ

  status          ENUM('started', 'completed', 'failed') NOT NULL,
  total_count     INT,                      -- 処理件数
  success_count   INT,                      -- 成功件数
  error_count     INT,                      -- エラー件数

  error_message   TEXT,                     -- エラーメッセージ
  started_at      TIMESTAMP NOT NULL,
  completed_at    TIMESTAMP NULL,

  INDEX idx_execution_date (execution_date),
  INDEX idx_area_code (area_code),
  INDEX idx_status (status)
);
```

**カラム説明**:

| カラム | 型 | NULL | 説明 |
|-------|---|------|-----|
| execution_date | DATE | NO | 実行日（例: 2025-12-16） |
| area_code | VARCHAR(20) | YES | NULL = 全体処理 |
| phase | ENUM | NO | list: 一覧取得 / detail: 詳細取得 / diff: 差分判定 |
| status | ENUM | NO | started / completed / failed |
| total_count | INT | YES | 処理対象件数 |
| success_count | INT | YES | 成功件数 |
| error_count | INT | YES | エラー件数 |

---

## 5. delete_logs（物理削除ログ）※Phase 4.5

**目的**: 物理削除した物件の履歴を記録。監査・分析用。

```sql
CREATE TABLE delete_logs (
  id              BIGINT AUTO_INCREMENT PRIMARY KEY,
  property_id     VARCHAR(32) NOT NULL,     -- 削除した物件ID
  title           TEXT,                     -- 削除時のタイトル
  detail_url      TEXT,                     -- 削除時のURL

  removed_at      DATETIME,                 -- 論理削除日時
  deleted_at      DATETIME NOT NULL,        -- 物理削除日時
  reason          VARCHAR(50) NOT NULL,     -- 削除理由

  INDEX idx_property_id (property_id),
  INDEX idx_deleted_at (deleted_at),
  INDEX idx_reason (reason)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**カラム説明**:

| カラム | 型 | NULL | 説明 |
|-------|---|------|-----|
| property_id | VARCHAR(32) | NO | 削除した物件ID |
| title | TEXT | YES | 削除時のタイトル（分析用） |
| detail_url | TEXT | YES | 削除時のURL（追跡用） |
| removed_at | DATETIME | YES | 論理削除された日時 |
| deleted_at | DATETIME | NO | 物理削除を実行した日時 |
| reason | VARCHAR(50) | NO | 削除理由（expired_90_days等） |

**削除理由の種類**:

- `expired_90_days` - 90日経過による自動削除
- `expired_30_days` - 30日経過による自動削除
- `manual_cleanup` - 管理者による手動削除
- `data_migration` - データ移行時の削除

**使用例**:

```sql
-- 最近30日間の物理削除件数
SELECT COUNT(*)
FROM delete_logs
WHERE deleted_at >= NOW() - INTERVAL 30 DAY;

-- 削除理由別の集計
SELECT reason, COUNT(*) as count
FROM delete_logs
GROUP BY reason;
```

**保持期間**: **無期限** （容量が小さいため削除不要）

---

## ER図（テキスト版）

```
┌─────────────────────────────┐
│      properties             │  ← 物件マスタ（検索・表示の正）
├─────────────────────────────┤
│ PK: id                      │
│     detail_url (UNIQUE)     │
│     title                   │
│     image_url               │
│     rent                    │
│     floor_plan              │
│     status (active/removed) │
│     removed_at              │
│     fetched_at              │
│     created_at              │
└─────────────────────────────┘
         ↑
         │ (差分検出で更新)
         │
┌─────────────────────────────┐
│ daily_property_snapshots    │  ← 日次スナップショット
├─────────────────────────────┤
│ PK: (snapshot_date,         │
│      property_id)           │
│     detail_url              │
│     area_code               │
└─────────────────────────────┘
         ↑
         │ (エリア別に取得)
         │
┌─────────────────────────────┐
│      scrape_areas           │  ← エリアマスタ
├─────────────────────────────┤
│ PK: area_code               │
│     name                    │
│     prefecture              │
│     city                    │
│     search_url              │
│     enabled                 │
│     last_scraped_at         │
│     last_status             │
└─────────────────────────────┘
         ↑
         │ (実行履歴)
         │
┌─────────────────────────────┐
│      scrape_logs            │  ← 実行ログ
├─────────────────────────────┤
│ PK: id                      │
│     execution_date          │
│     area_code               │
│     phase                   │
│     status                  │
│     total_count             │
│     success_count           │
│     error_count             │
└─────────────────────────────┘
         ↓
         │ (物理削除記録)
         │
┌─────────────────────────────┐
│      delete_logs            │  ← 物理削除ログ (Phase 4.5)
├─────────────────────────────┤
│ PK: id                      │
│     property_id             │
│     title                   │
│     detail_url              │
│     removed_at              │
│     deleted_at              │
│     reason                  │
└─────────────────────────────┘
```

---

## データフロー

### 毎日の処理フロー

```
1. 一覧取得
   ↓
   scrape_areas から enabled = true のエリアを取得
   ↓
   各エリアの search_url をスクレイピング
   ↓
   物件ID一覧を取得
   ↓
   daily_property_snapshots に保存（今日の日付）

2. 差分判定
   ↓
   昨日と今日の snapshots を比較
   ↓
   新規: 今日にあって昨日にない → detail取得対象
   継続: 両方にある → 何もしない
   消滅: 昨日にあって今日にない → status = 'removed'

3. 詳細取得（新規のみ）
   ↓
   新規IDのみ詳細ページをスクレイピング
   ↓
   properties にINSERT or UPDATE

4. 消滅処理
   ↓
   properties.status = 'removed', removed_at = NOW()
   ↓
   Meilisearch から DELETE
```

---

## マイグレーション戦略

### 現在（PostgreSQL）から MySQL への移行

**手順**:

1. MySQL 8.x コンテナを追加
2. 新しいスキーマを作成
3. 既存データをエクスポート
4. MySQL にインポート
5. GORM で接続切り替え
6. PostgreSQL コンテナを停止（削除はしない）

**注意**: データ削除禁止、既存コンテナ保持

---

## インデックス戦略

### 検索性能向上のためのインデックス

```sql
-- properties
CREATE INDEX idx_properties_status ON properties(status);
CREATE INDEX idx_properties_rent ON properties(rent);
CREATE INDEX idx_properties_floor_plan ON properties(floor_plan);
CREATE INDEX idx_properties_walk_time ON properties(walk_time);
CREATE INDEX idx_properties_created_at ON properties(created_at);

-- daily_property_snapshots
CREATE INDEX idx_snapshots_date ON daily_property_snapshots(snapshot_date);
CREATE INDEX idx_snapshots_property_id ON daily_property_snapshots(property_id);

-- scrape_areas
CREATE INDEX idx_areas_enabled ON scrape_areas(enabled);
CREATE INDEX idx_areas_last_scraped ON scrape_areas(last_scraped_at);

-- scrape_logs
CREATE INDEX idx_logs_execution_date ON scrape_logs(execution_date);
CREATE INDEX idx_logs_area_code ON scrape_logs(area_code);
CREATE INDEX idx_logs_status ON scrape_logs(status);
```

---

## 将来の拡張

### ユーザー関連（Phase 4+）

```sql
-- users（ユーザーマスタ）
-- user_favorites（お気に入り）
-- user_hidden_properties（非表示設定）
-- user_notifications（通知設定）
-- discrepancy_reports（相違報告）
```

### アフィリエイト連携（Phase 5+）

```sql
-- affiliate_feeds（提携データソース）
-- feed_properties（フィード由来物件）
```

---

**最終更新**: 2025-12-16
