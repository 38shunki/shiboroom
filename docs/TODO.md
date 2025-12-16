# TODO リスト - 不動産ポータル開発

## 🎯 優先順位

### Phase 0: PoC検証（最優先）

#### ✅ PoC合格ラインチェック項目

- [ ] **1) スクレイピング安定性**
  - [ ] 同じURLで3回連続成功（タイトル/画像URL/詳細URLが取れる）
  - [ ] 取得失敗時に空データで保存しない（DB/検索の汚染防止）
  - [ ] 実施URL: `https://realestate.yahoo.co.jp/rent/...` (具体的なURL)

- [ ] **2) 検索が目的通り機能**
  - [ ] `q=新宿` のような検索がMeilisearchで返る
  - [ ] フィルタ（rent / floor_plan / walk_time）がMeilisearch側で絞れている
  - [ ] DB側で後から絞っていない（クエリログで確認）

- [ ] **3) 画像外部参照が成立**
  - [ ] `<img src="yimg.jp...">` がフロントで表示できる
  - [ ] 表示できない場合はプレースホルダへフォールバック（実装済み）
  - [ ] 短時間で画像URLが変わらないかを確認（翌日も同じ物件が表示できるか）

- [ ] **4) Yahoo不動産へのリンクが成立**
  - [ ] `detail_url` が常に正しい
  - [ ] `target="_blank" rel="noreferrer"` で遷移OK

**✅ PoC合格基準**: 上記4項目すべてクリアで「技術的にいける」= YES

#### 🚫 PoC中止ライン（いずれか該当したら次に進まない）

- [ ] 同一URLで3回中2回以上スクレイピング失敗
- [ ] 画像URLが24時間以内に頻繁に失効
- [ ] 一覧ページ取得で明確なブロック挙動
- [ ] Meilisearchのインデックスが再構築不能

**⚠️ 重要**: 上記に該当する場合は技術的に継続不可と判断し、代替手段（提携・API利用）を検討

---

### Phase 1: 技術スタック修正（必須）

#### 🔧 現在実装との乖離修正

- [ ] **Database: PostgreSQL → MySQL 8.x**
  - [ ] MySQL 8.x コンテナに変更
  - [ ] GORMへ移行（現在は直接SQL）
  - [ ] golang-migrateでマイグレーション管理
  - [ ] 既存データのマイグレーション（DELETE禁止）

- [ ] **Frontend: React+Vite → Next.js 14**
  - [ ] Next.js 14プロジェクト作成（App Router）
  - [ ] 既存のReactコンポーネントを移行
  - [ ] Tailwind CSS + Radix UI導入
  - [ ] React Query（Server State）導入
  - [ ] Zustand（UI State）導入
  - [ ] SSG/SSR設計（LP・OGP部分）

- [ ] **ORM導入: 直接SQL → GORM**
  - [ ] models定義をGORM形式に変更
  - [ ] database層をGORMに書き換え
  - [ ] トランザクション処理の統一

---

### Phase 2: PoCの安定化・改善

#### 🛡️ データ品質改善

- [ ] **URL正規化機能**
  - [ ] クエリ文字列削除（`?xxx`）
  - [ ] 末尾スラッシュ統一
  - [ ] `<link rel="canonical">` 優先採用
  - [ ] ID生成をMD5(正規化URL)に変更

- [ ] **画像URL生存確認**
  - [ ] HTTP HEADでステータスコード確認
  - [ ] 200以外は `image_url = NULL` に設定
  - [ ] フロントはプレースホルダにフォールバック

- [ ] **正規表現抽出の誤爆対策**
  - [ ] 階数抽出の改善（家賃の数値を誤爆しない）
  - [ ] 怪しい値はNULL化
  - [ ] パース失敗時の処理改善
  - [ ] バリデーション追加（rent: 1万円〜100万円など）

- [ ] **再スクレイプ時の上書き（Upsert）**
  - [ ] `detail_url` UNIQUEを利用
  - [ ] `fetched_at` 更新
  - [ ] `created_at` は初回固定

---

### Phase 3: 差分検出・毎日更新機能

#### 📊 テーブル設計

- [ ] **daily_property_snapshots テーブル**
  ```sql
  CREATE TABLE daily_property_snapshots (
    snapshot_date DATE NOT NULL,
    property_id VARCHAR(32) NOT NULL,
    detail_url TEXT NOT NULL,
    area_code VARCHAR(20),
    PRIMARY KEY (snapshot_date, property_id)
  );
  ```

- [ ] **scrape_areas テーブル**
  ```sql
  CREATE TABLE scrape_areas (
    area_code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    prefecture VARCHAR(50),
    city VARCHAR(50),
    last_scraped_at TIMESTAMP,
    enabled BOOLEAN DEFAULT true
  );
  ```

- [ ] **properties テーブルに論理削除追加**
  ```sql
  ALTER TABLE properties
    ADD COLUMN status ENUM('active', 'removed') DEFAULT 'active',
    ADD COLUMN removed_at TIMESTAMP NULL;
  ```

#### 🔄 差分検出ロジック

- [ ] **一覧取得機能**
  - [ ] Yahoo不動産検索結果ページのスクレイピング
  - [ ] エリア単位で分割取得（都道府県→市区町村）
  - [ ] 物件ID + detail_url のみ取得
  - [ ] `daily_property_snapshots` に保存

- [ ] **一覧取得の完全性チェック（重要！）**
  - [ ] 全エリアの snapshot が成功した場合のみ差分判定を実行
  - [ ] 一部エリア失敗時は当日の削除処理をスキップ
  - [ ] skip 時は scrape_logs に理由を記録
  - [ ] ⚠️ **途中失敗時に消滅判定すると誤削除が発生するため必須**

- [ ] **差分判定ロジック**
  - [ ] 昨日と今日のsnapshotを比較
  - [ ] 新規: 今日にあって昨日にない
  - [ ] 継続: 両方にある
  - [ ] 消滅: 昨日にあって今日にない
  - [ ] 完全性チェックがOKの場合のみ実行

- [ ] **消滅物件の処理（慎重に）**
  - [ ] `status = 'removed'` に更新
  - [ ] `removed_at` にタイムスタンプ設定
  - [ ] Meilisearchから削除
  - [ ] 検索結果に表示しない
  - [ ] ⚠️ 物理削除は絶対にしない（論理削除のみ）

#### ⏰ スケジューラー実装

- [ ] **cron / asynq 選定**
- [ ] **実行順序の実装**
  1. 一覧取得（全エリア）
  2. 差分判定
  3. 新規のみ詳細取得
  4. 消滅を検索から削除

- [ ] **実行時間設定**
  - [ ] 1日1回のみ
  - [ ] 深夜〜早朝（例: 2:00 AM）
  - [ ] 固定時刻

- [ ] **エラー処理**
  - [ ] 403/429/5xx → 即停止
  - [ ] 再実行は翌日まで禁止
  - [ ] 中途半端な状態を公開しない

---

### Phase 4: 負荷最小化・安全運用設計

#### ⚙️ 設定外部化

- [ ] **scraper_config.yaml 作成**
  ```yaml
  scraper:
    interval_seconds: 2
    max_requests_per_day: 5000
    stop_on_error: true
    retry_count: 0
    timeout_seconds: 30
  ```

- [ ] **レート制限実装**
  - [ ] 同時実行数 = 1（並列禁止）
  - [ ] リクエスト間sleep固定
  - [ ] 1日の最大リクエスト数制限
  - [ ] エラー時の即停止

---

### Phase 4.5: 物理削除バッチ（中長期対応）

#### 🗑️ データ量最適化

- [ ] **delete_logs テーブル作成**
  ```sql
  CREATE TABLE delete_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    property_id VARCHAR(32) NOT NULL,
    title TEXT,
    detail_url TEXT,
    removed_at DATETIME,
    deleted_at DATETIME NOT NULL,
    reason VARCHAR(50) NOT NULL
  );
  ```

- [ ] **物理削除バッチ実装**
  - [ ] 削除対象の特定（90日経過した removed 物件）
  - [ ] 事前チェック（当日の一覧取得が完全成功）
  - [ ] 削除件数の異常値チェック（10,000件超で中止）
  - [ ] delete_logsへの記録
  - [ ] Meilisearchから削除（念のため）
  - [ ] DB物理削除実行

- [ ] **dry-runモード実装**
  - [ ] 削除対象の確認のみ（実削除なし）
  - [ ] ログ出力で件数・対象を確認
  - [ ] 初回実行は必ずdry-run

- [ ] **スケジューラー設定**
  - [ ] 週1回実行（日曜深夜3:00 AM推奨）
  - [ ] 保持期間設定（初期: 90日）
  - [ ] アラート設定（異常削除件数時）

- [ ] **監視・アラート**
  - [ ] 削除済み物件の割合監視
  - [ ] 物理削除件数の監視
  - [ ] データ量の定期確認

**⚠️ 重要**: 初期は論理削除のみ。データ量が10GB超えたら物理削除を検討

---

### Phase 5: アーキテクチャ改善

#### 🏗️ データ取得レイヤの抽象化

- [ ] **Fetcher Interface 設計**
  ```go
  type PropertyFetcher interface {
      FetchList(area AreaCode) ([]PropertyID, error)
      FetchDetail(id PropertyID) (*PropertyInput, error)
  }

  type PropertyInput struct {
      ID          string
      Title       string
      Rent        *int
      FloorPlan   string
      Area        *float64
      WalkTime    *int
      Station     string
      Address     string
      ImageURL    string
      DetailURL   string
  }
  ```

- [ ] **実装クラス**
  - [ ] `YahooScraper` (現在)
  - [ ] `AffiliateFeed` (将来)
  - [ ] `ManualUpload` (将来)

- [ ] **DI構造**
  - [ ] コンストラクタDI
  - [ ] 必要に応じてwire導入

---

### Phase 6: 管理機能

#### 📈 ステータス可視化

- [ ] **管理画面API**
  - [ ] 今日の新規件数
  - [ ] 今日の消滅件数
  - [ ] エリア別取得成功率
  - [ ] 最終実行時刻
  - [ ] エラーログ

- [ ] **管理画面UI**
  - [ ] Next.js管理ページ
  - [ ] 認証（Basic Auth or JWT）
  - [ ] スクレイピング手動実行
  - [ ] エリア有効/無効切り替え

---

## 📋 実装メモ

### 現在の状態
- ✅ 基本的なスクレイピング機能
- ✅ PostgreSQL + Meilisearch
- ✅ React + Vite フロントエンド
- ✅ フィルタ機能（賃料、間取り、徒歩時間）
- ✅ 外部画像参照
- ✅ Yahoo不動産へのリンク

### 技術的負債
- ❌ PostgreSQLを使用（MySQL 8.xに変更必要）
- ❌ ORM未使用（GORM導入必要）
- ❌ React+Vite使用（Next.js 14に移行必要）
- ❌ マイグレーション管理なし（golang-migrate導入必要）

### 注意事項
- ⚠️ **データ削除禁止**: 既存データは必ず保持
- ⚠️ **Docker破壊禁止**: 既存コンテナの削除禁止
- ⚠️ **マイグレーション慎重**: リフレッシュ・既存データ消去禁止
- ⚠️ **BAN対策必須**: 同時実行数1、sleep固定、エラー即停止

---

## 🎯 次のアクション

1. **Phase 0: PoC検証** を実施（最優先）
2. PoC合格確認後、Phase 1の技術スタック修正を計画
3. 技術スタック修正完了後、Phase 2以降を順次実装

---

**最終更新**: 2025-12-16
