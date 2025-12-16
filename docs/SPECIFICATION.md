# 不動産ポータルサイト PoC - 仕様書

**プロジェクト名**: Real Estate Portal PoC
**作成日**: 2025-12-16
**バージョン**: 1.0

## 📋 目次

1. [プロジェクト概要](#プロジェクト概要)
2. [技術スタック](#技術スタック)
3. [システム構成](#システム構成)
4. [機能仕様](#機能仕様)
5. [データモデル](#データモデル)
6. [API仕様](#api仕様)
7. [フロントエンド仕様](#フロントエンド仕様)
8. [スクレイピング仕様](#スクレイピング仕様)
9. [検索・フィルタ仕様](#検索フィルタ仕様)
10. [デプロイ・運用](#デプロイ運用)

---

## プロジェクト概要

### 目的
Yahoo不動産から物件情報をスクレイピングし、検索・フィルタ機能を持つ不動産ポータルサイトの技術的実現可能性を検証する。

### スコープ
- ✅ 物件詳細ページからの基本情報抽出
- ✅ 画像外部URL参照での表示
- ✅ 全文検索機能（Meilisearch）
- ✅ 多条件フィルタ機能
- ✅ 元サイトへのリンク機能

### 制約事項
- **技術検証目的のみ**（商用利用には提携・許諾が必要）
- スクレイピング間隔: 1秒以上（サーバー負荷軽減）
- 画像: 外部URL参照（ホットリンク）
- データ更新: 手動スクレイピング

---

## 技術スタック

### バックエンド
| 技術 | バージョン | 用途 |
|------|-----------|------|
| Go | 1.23 | メインAPI・スクレイピング |
| Gin | 1.9.1 | Webフレームワーク |
| goquery | 1.8.1 | HTMLパース |
| PostgreSQL | 15-alpine | データベース |
| Meilisearch | v1.5 | 全文検索エンジン |

### フロントエンド
| 技術 | バージョン | 用途 |
|------|-----------|------|
| React | 18.2.0 | UIフレームワーク |
| Vite | 5.0.8 | ビルドツール |

### インフラ
- Docker / Docker Compose
- ポート構成:
  - Frontend: 5176
  - Backend: 8084
  - PostgreSQL: 5433
  - Meilisearch: 7700

---

## システム構成

```
┌─────────────────────────────────────────────────────────┐
│                    User (Browser)                       │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│            React Frontend (Port 5176)                   │
│  - 検索UI                                                │
│  - フィルタUI                                            │
│  - 物件一覧表示                                          │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP
                      ▼
┌─────────────────────────────────────────────────────────┐
│         Go Backend API (Port 8084)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Scraper     │  │   Database   │  │    Search    │  │
│  │   Engine     │  │   Handler    │  │    Client    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└────────┬──────────────────┬─────────────────┬──────────┘
         │                  │                 │
         │                  ▼                 ▼
         │         ┌──────────────┐  ┌──────────────┐
         │         │ PostgreSQL   │  │ Meilisearch  │
         │         │ (Port 5433)  │  │ (Port 7700)  │
         │         └──────────────┘  └──────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│              Yahoo不動産                                │
│         (スクレイピング対象)                            │
└─────────────────────────────────────────────────────────┘
```

---

## 機能仕様

### 1. 物件スクレイピング機能

#### 1.1 単一URLスクレイピング
- **エンドポイント**: `POST /api/scrape`
- **入力**: Yahoo不動産の物件詳細URL
- **処理**:
  1. HTMLを取得
  2. メタデータ・詳細情報を抽出
  3. PostgreSQLに保存
  4. Meilisearchにインデックス
- **出力**: 抽出した物件情報（JSON）

#### 1.2 一括スクレイピング
- **エンドポイント**: `POST /api/scrape/batch`
- **入力**: URL配列
- **処理**: 各URLを1秒間隔で順次スクレイピング
- **出力**: 成功・失敗件数、エラー詳細

### 2. 検索機能

#### 2.1 キーワード検索
- **エンドポイント**: `GET /api/search?q={keyword}`
- **検索対象**:
  - タイトル
  - 駅名
  - 住所
  - 間取り

#### 2.2 全件取得
- **エンドポイント**: `GET /api/properties`
- **ソート**: 登録日時の降順

### 3. フィルタ機能

#### 3.1 賃料範囲フィルタ
- **パラメータ**: `min_rent`, `max_rent`
- **単位**: 円
- **例**: `min_rent=80000&max_rent=120000`

#### 3.2 間取りフィルタ
- **パラメータ**: `floor_plan`（複数指定可）
- **値**: 1K, 1DK, 1LDK, 2K, 2DK, 2LDK, 3LDK等
- **例**: `floor_plan=1K&floor_plan=1DK`

#### 3.3 駅徒歩時間フィルタ
- **パラメータ**: `max_walk_time`
- **単位**: 分
- **例**: `max_walk_time=10`

#### 3.4 複合フィルタ
- **エンドポイント**: `GET /api/filter`
- **組み合わせ例**:
  ```
  /api/filter?q=新宿&min_rent=80000&max_rent=120000&floor_plan=1K&max_walk_time=10
  ```

### 4. 画像表示機能
- **方式**: 外部URL参照（ホットリンク）
- **取得元**: Yahoo画像サーバー (yimg.jp)
- **フォールバック**: 画像読み込み失敗時は「画像なし」プレースホルダー表示

---

## データモデル

### Property（物件）テーブル

| カラム名 | 型 | NULL | 説明 |
|---------|-----|------|------|
| id | VARCHAR(32) | NOT NULL | MD5ハッシュ（URLベース） |
| detail_url | TEXT | NOT NULL | 物件詳細URL（UNIQUE） |
| title | TEXT | NOT NULL | 物件タイトル |
| image_url | TEXT | NULL | 物件画像URL |
| rent | INTEGER | NULL | 賃料（円） |
| floor_plan | VARCHAR(20) | NULL | 間取り |
| area | DECIMAL(10,2) | NULL | 面積（㎡） |
| walk_time | INTEGER | NULL | 駅徒歩（分） |
| station | TEXT | NULL | 最寄り駅 |
| address | TEXT | NULL | 住所 |
| building_age | INTEGER | NULL | 築年数 |
| floor | INTEGER | NULL | 階数 |
| fetched_at | TIMESTAMP | NOT NULL | スクレイピング日時 |
| created_at | TIMESTAMP | NOT NULL | 登録日時 |

#### インデックス
- PRIMARY KEY: `id`
- UNIQUE: `detail_url`
- INDEX: `created_at` (DESC)
- INDEX: `rent`
- INDEX: `floor_plan`
- INDEX: `walk_time`

---

## API仕様

### Base URL
```
http://localhost:8084
```

### エンドポイント一覧

#### 1. ヘルスチェック
```http
GET /health
```

**レスポンス例**:
```json
{
  "status": "ok",
  "time": "2025-12-16T01:00:00Z"
}
```

---

#### 2. 物件一覧取得
```http
GET /api/properties
```

**レスポンス例**:
```json
[
  {
    "id": "123b204cdc2754601840243f5a97dc99",
    "detail_url": "https://realestate.yahoo.co.jp/rent/detail/...",
    "title": "【Yahoo!不動産】新宿夏目坂コート (2階/1K)の賃貸物件詳細",
    "image_url": "https://realestate-pctr.c.yimg.jp/...",
    "rent": 105000,
    "floor_plan": "1K",
    "area": 25.68,
    "walk_time": 6,
    "station": "...",
    "address": "東京都新宿区喜久井町",
    "building_age": 6,
    "floor": 2,
    "fetched_at": "2025-12-16T01:00:00Z",
    "created_at": "2025-12-16T01:00:00Z"
  }
]
```

---

#### 3. 物件詳細取得
```http
GET /api/properties/:id
```

**パラメータ**:
- `id`: 物件ID

---

#### 4. スクレイピング（単一）
```http
POST /api/scrape
Content-Type: application/json

{
  "url": "https://realestate.yahoo.co.jp/rent/detail/..."
}
```

**レスポンス**: 抽出した物件情報

---

#### 5. スクレイピング（一括）
```http
POST /api/scrape/batch
Content-Type: application/json

{
  "urls": [
    "https://realestate.yahoo.co.jp/rent/detail/...",
    "https://realestate.yahoo.co.jp/rent/detail/..."
  ]
}
```

**レスポンス例**:
```json
{
  "success": 2,
  "failed": 0,
  "errors": [],
  "properties": [...]
}
```

---

#### 6. キーワード検索
```http
GET /api/search?q={keyword}&limit={limit}
```

**パラメータ**:
- `q`: 検索キーワード
- `limit`: 取得件数（デフォルト: 20）

---

#### 7. フィルタ検索
```http
GET /api/filter?min_rent={min}&max_rent={max}&floor_plan={plan}&max_walk_time={time}&q={keyword}&limit={limit}
```

**パラメータ**:
- `min_rent`: 最低賃料（円）
- `max_rent`: 最高賃料（円）
- `floor_plan`: 間取り（複数指定可）
- `max_walk_time`: 最大駅徒歩時間（分）
- `q`: キーワード
- `limit`: 取得件数（デフォルト: 20）

**例**:
```
/api/filter?min_rent=80000&max_rent=120000&floor_plan=1K&floor_plan=1DK&max_walk_time=10&q=新宿
```

---

## フロントエンド仕様

### ページ構成

```
┌─────────────────────────────────────────────┐
│              Header                         │
│  不動産ポータル - PoC                       │
└─────────────────────────────────────────────┘
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │ 物件URLをスクレイピング              │   │
│  │ [URL入力欄] [スクレイピング]        │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │ 物件検索・フィルタ                  │   │
│  │ [検索欄] [▶フィルタ] [検索] [リセット] │
│  │                                     │   │
│  │ ┌─ フィルタパネル ─────────────┐   │   │
│  │ │ 賃料: [最小] 〜 [最大]        │   │   │
│  │ │ 間取り: □1K □1DK □1LDK...    │   │   │
│  │ │ 駅徒歩: [セレクト]            │   │   │
│  │ └───────────────────────────┘   │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │ 物件一覧 (5件)                      │   │
│  │                                     │   │
│  │ ┌──────┐ ┌──────┐ ┌──────┐        │   │
│  │ │画像  │ │画像  │ │画像  │        │   │
│  │ │タイトル│ タイトル│ タイトル│       │   │
│  │ │賃料  │ │賃料  │ │賃料  │        │   │
│  │ │間取り│ │間取り│ │間取り│        │   │
│  │ │詳細→│ │詳細→│ │詳細→│        │   │
│  │ └──────┘ └──────┘ └──────┘        │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

### コンポーネント構成

```
App
├── Header
├── ScrapeSection
│   └── ScrapeForm
├── SearchSection
│   ├── SearchForm
│   └── FilterPanel
│       ├── RentFilter
│       ├── FloorPlanFilter
│       └── WalkTimeFilter
└── ResultsSection
    └── PropertyGrid
        └── PropertyCard[]
            ├── PropertyImage
            ├── PropertyTitle
            ├── PropertyDetails
            └── PropertyLink
```

---

## スクレイピング仕様

### 対象サイト
- Yahoo不動産 物件詳細ページ
- URL形式: `https://realestate.yahoo.co.jp/rent/detail/{id}/`

### 抽出方法

#### 1. メタデータ（優先）
| データ | 抽出元 |
|--------|--------|
| タイトル | `<meta property="og:title">` |
| 画像URL | `<meta property="og:image">` |

#### 2. 本文からの抽出（正規表現）

##### 賃料
```regex
([0-9]+\.?[0-9]*)万円
賃料[：:]\s*([0-9,]+)円
```

##### 間取り
```regex
([0-9]?[SLDK]+)\b
```

##### 面積
```regex
([0-9]+\.?[0-9]*)[㎡m²]
```

##### 駅徒歩時間
```regex
[徒歩]+([0-9]+)分
```

##### 築年数
```regex
築[年数]*([0-9]+)年
```

##### 階数
```regex
([0-9]+)[階F]
```

##### 住所
```regex
(東京都|大阪府|神奈川県|千葉県|埼玉県)[^\n]+
```

### 注意事項
- User-Agent: `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36`
- タイムアウト: 30秒
- リトライ: なし（エラーは返却）
- 文字エンコーディング: UTF-8（rune単位で安全に切断）

---

## 検索・フィルタ仕様

### Meilisearch設定

#### 検索対象属性（Searchable）
- `title`
- `detail_url`
- `station`
- `address`
- `floor_plan`

#### フィルタ可能属性（Filterable）
- `id`
- `rent`
- `floor_plan`
- `walk_time`
- `area`
- `building_age`
- `floor`
- `station`

#### ソート可能属性（Sortable）
- `rent`
- `area`
- `walk_time`
- `building_age`
- `created_at`

### フィルタクエリ生成例

```go
// 賃料範囲: 80,000円〜120,000円
filters = "rent >= 80000 AND rent <= 120000"

// 間取り: 1K または 1DK
filters = "(floor_plan = '1K' OR floor_plan = '1DK')"

// 駅徒歩: 10分以内
filters = "walk_time <= 10"

// 複合条件
filters = "rent >= 80000 AND rent <= 120000 AND (floor_plan = '1K' OR floor_plan = '1DK') AND walk_time <= 10"
```

---

## デプロイ・運用

### 起動方法

```bash
# プロジェクトディレクトリへ移動
cd /Users/shu/Documents/dev/real-estate-portal

# 全サービス起動
docker-compose up -d

# ビルドから起動
docker-compose up -d --build
```

### 停止方法

```bash
# 停止（データ保持）
docker-compose stop

# コンテナ削除（データ保持）
docker-compose down

# ⚠️ データも削除（危険）
docker-compose down -v
```

### ログ確認

```bash
# 全サービス
docker-compose logs -f

# 特定サービス
docker-compose logs -f backend
docker-compose logs -f frontend
```

### データベース操作

```bash
# PostgreSQL接続
docker-compose exec db psql -U realestate_user -d realestate_db

# バックアップ
docker-compose exec db pg_dump -U realestate_user realestate_db > backup_$(date +%Y%m%d).sql

# リストア
docker-compose exec -T db psql -U realestate_user -d realestate_db < backup_20251216.sql
```

### Meilisearch操作

```bash
# ヘルスチェック
curl http://localhost:7700/health

# インデックス確認
curl -H "Authorization: Bearer masterKey123" http://localhost:7700/indexes

# 統計確認
curl -H "Authorization: Bearer masterKey123" http://localhost:7700/stats
```

---

## パフォーマンス

### 推奨スペック
- CPU: 2コア以上
- メモリ: 4GB以上
- ストレージ: 10GB以上

### レスポンスタイム（目安）
- スクレイピング（単一）: 2〜5秒
- 検索: 50〜200ms
- フィルタ検索: 100〜300ms

### スケーリング
- PostgreSQL: 接続プール設定で対応
- Meilisearch: 単一インスタンスで10万件程度まで対応可能
- Go Backend: 水平スケーリング可能（ステートレス）

---

## セキュリティ

### 実装済み
- CORS設定（フロントエンド origin のみ許可）
- SQL injection対策（プリペアドステートメント）
- XSS対策（React の自動エスケープ）

### 未実装（本番化時に必要）
- 認証・認可
- レート制限
- HTTPS/TLS
- CSRFトークン
- セッション管理

---

## 今後の拡張

### 優先度: 高
1. **自動スクレイピング**
   - クーロンジョブによる定期実行
   - 差分更新機能

2. **詳細フィルタ**
   - エリア選択（区・市単位）
   - 設備条件（バストイレ別、オートロック等）
   - 築年数フィルタ

3. **ソート機能**
   - 賃料順
   - 新着順
   - 面積順

### 優先度: 中
1. **お気に入り機能**
   - ユーザー登録
   - 物件保存

2. **比較機能**
   - 複数物件の並列表示

3. **通知機能**
   - 新着物件メール通知
   - 条件マッチング通知

### 優先度: 低
1. **地図表示**
   - Google Maps API統合
   - 物件位置表示

2. **画像管理**
   - 画像ダウンロード・保存
   - サムネイル生成

3. **管理画面**
   - 物件一覧・編集
   - スクレイピング設定

---

## トラブルシューティング

### 1. コンテナが起動しない

```bash
# ログ確認
docker-compose logs backend

# 再ビルド
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### 2. データベース接続エラー

```bash
# データベース状態確認
docker-compose exec db pg_isready -U realestate_user

# テーブル確認
docker-compose exec db psql -U realestate_user -d realestate_db -c "\dt"
```

### 3. Meilisearch接続エラー

```bash
# Meilisearch状態確認
curl http://localhost:7700/health

# コンテナ再起動
docker-compose restart meilisearch
```

### 4. フロントエンド表示エラー

```bash
# フロントエンドログ確認
docker-compose logs -f frontend

# ブラウザコンソール確認
# Chrome DevTools → Console
```

---

## ライセンス・法的事項

### 重要な注意事項

⚠️ **このプロジェクトは技術検証目的（PoC）です**

1. **商用利用禁止**
   - Yahoo不動産の利用規約により、商用利用には提携が必要

2. **スクレイピング制限**
   - robots.txt の遵守
   - 過度なアクセス禁止（現在は1秒間隔）
   - User-Agentの適切な設定

3. **画像利用**
   - 画像URLは外部参照のみ
   - 画像ダウンロード・再配布は許可が必要

4. **本番化に向けて**
   - Yahoo!との提携交渉
   - アフィリエイトプログラムへの参加
   - データ利用契約の締結

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2025-12-16 | 1.0 | 初版作成 |

---

## 連絡先・サポート

プロジェクトパス: `/Users/shu/Documents/dev/real-estate-portal`

Docker管理ドキュメント: `/Users/shu/Documents/dev/Docker.md`

---

**End of Specification**
