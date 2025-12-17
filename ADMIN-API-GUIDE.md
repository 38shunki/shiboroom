# 管理画面API ガイド

**更新日**: 2025-12-17
**対象**: Phase 4.5 & Phase 6 実装

---

## 📋 概要

このドキュメントでは、管理者向けのAPIエンドポイントの使い方を説明します。

### ⚠️ 重要な注意事項

1. **本番環境では認証が必要**
   - 現在は認証なしでアクセス可能
   - 本番環境では Basic Auth または JWT を実装すること

2. **データ削除には細心の注意を**
   - 物理削除は dry-run モードで必ず確認
   - 初回実行は必ず `"dry_run": true`

3. **スクレイピングは慎重に**
   - レート制限を遵守
   - 403/429エラーが出たら1時間以上待機

---

## 🔧 管理画面API エンドポイント

### 1. 統計情報

#### システム統計を取得

```bash
GET /api/admin/stats
```

**レスポンス例**:
```json
{
  "properties": {
    "active": 1523,
    "removed": 45,
    "total": 1568
  },
  "recent_activity": {
    "fetched_last_24h": 234
  },
  "snapshots": {
    "total": 15234
  },
  "changes": {
    "last_7_days": 89
  },
  "deletions": {
    "total_deleted": 12,
    "by_reason": {
      "expired_90_days": 10,
      "manual_deletion": 2
    },
    "deleted_last_30_days": 5,
    "currently_removed": 45,
    "expired_ready_for_deletion": 8
  }
}
```

**使用例**:
```bash
curl http://localhost:8084/api/admin/stats
```

---

#### 最近の活動を取得

```bash
GET /api/admin/activity?limit=50
```

**パラメータ**:
- `limit`: 取得件数（デフォルト: 50）

**レスポンス例**:
```json
{
  "properties": [...],
  "count": 50
}
```

**使用例**:
```bash
curl "http://localhost:8084/api/admin/activity?limit=100"
```

---

#### エリア別統計を取得

```bash
GET /api/admin/area-stats
```

**レスポンス例**:
```json
{
  "area_stats": [
    {
      "station": "新宿駅",
      "count": 234
    },
    {
      "station": "渋谷駅",
      "count": 189
    }
  ],
  "count": 20
}
```

**使用例**:
```bash
curl http://localhost:8084/api/admin/area-stats
```

---

#### 家賃分布を取得

```bash
GET /api/admin/price-distribution
```

**レスポンス例**:
```json
{
  "price_distribution": [
    {
      "range_label": "〜5万円",
      "min_rent": 0,
      "max_rent": 50000,
      "count": 12
    },
    {
      "range_label": "5〜8万円",
      "min_rent": 50000,
      "max_rent": 80000,
      "count": 456
    }
  ]
}
```

**使用例**:
```bash
curl http://localhost:8084/api/admin/price-distribution
```

---

### 2. スクレイピング制御

#### スクレイピングを手動実行

```bash
POST /api/admin/scraping/trigger
```

**レスポンス例**:
```json
{
  "message": "Scraping job started",
  "status": "running"
}
```

**使用例**:
```bash
curl -X POST http://localhost:8084/api/admin/scraping/trigger
```

**注意事項**:
- 非同期で実行されます（即座に応答が返ります）
- 実行中かどうかは `/api/admin/scraping/status` で確認
- ログは `docker logs realestate-backend -f` で確認

---

#### スクレイピング状態を確認

```bash
GET /api/admin/scraping/status
```

**レスポンス例**:
```json
{
  "status": "idle",
  "message": "Status tracking not yet implemented"
}
```

**使用例**:
```bash
curl http://localhost:8084/api/admin/scraping/status
```

**TODO**: 実際の実行状態トラッキングを実装予定

---

### 3. クリーンアップ（物理削除）

#### 物理削除を実行（Dry-run推奨）

```bash
POST /api/admin/cleanup/run
```

**リクエストボディ**:
```json
{
  "retention_days": 90,       // 削除対象: 90日以前に removed になった物件
  "max_deletion_count": 10000, // 安全制限: 一度に削除できる最大数
  "dry_run": true             // true = 実削除しない（推奨）
}
```

**レスポンス例（Dry-run）**:
```json
{
  "target_count": 8,
  "deleted_count": 8,
  "skipped_count": 0,
  "error_count": 0,
  "dry_run": true,
  "executed_at": "2025-12-17T10:30:00Z",
  "deleted_properties": [
    "abc123...",
    "def456..."
  ]
}
```

**使用例（Dry-run）**:
```bash
# まず dry-run で確認（推奨）
curl -X POST http://localhost:8084/api/admin/cleanup/run \
  -H "Content-Type: application/json" \
  -d '{
    "retention_days": 90,
    "max_deletion_count": 10000,
    "dry_run": true
  }'

# 確認後、実際に削除（慎重に！）
curl -X POST http://localhost:8084/api/admin/cleanup/run \
  -H "Content-Type: application/json" \
  -d '{
    "retention_days": 90,
    "max_deletion_count": 10000,
    "dry_run": false
  }'
```

**⚠️ 重要**:
1. **初回は必ず dry-run で確認**
2. 削除対象が異常に多い場合は中止
3. `max_deletion_count` を超えるとエラーで停止
4. 削除されたデータは **delete_logs** テーブルに記録

---

#### 削除ログを取得

```bash
GET /api/admin/cleanup/logs?limit=100
```

**パラメータ**:
- `limit`: 取得件数（デフォルト: 100）

**レスポンス例**:
```json
{
  "logs": [
    {
      "id": 1,
      "property_id": "abc123...",
      "title": "渋谷区恵比寿 1K",
      "detail_url": "https://...",
      "removed_at": "2025-09-15T12:00:00Z",
      "deleted_at": "2025-12-17T10:30:00Z",
      "reason": "expired_90_days"
    }
  ],
  "count": 12
}
```

**使用例**:
```bash
curl "http://localhost:8084/api/admin/cleanup/logs?limit=50"
```

---

### 4. 物件履歴・変更履歴

#### 物件のスナップショット履歴を取得

```bash
GET /api/admin/properties/:id/history?limit=30
```

**パラメータ**:
- `id`: 物件ID
- `limit`: 取得件数（デフォルト: 30）

**レスポンス例**:
```json
{
  "property_id": "abc123...",
  "snapshots": [
    {
      "id": 1,
      "property_id": "abc123...",
      "snapshot_at": "2025-12-17",
      "rent": 85000,
      "floor_plan": "1K",
      "has_changed": true,
      "change_note": "2 changes detected"
    }
  ],
  "count": 30
}
```

**使用例**:
```bash
curl "http://localhost:8084/api/admin/properties/abc123.../history?limit=30"
```

---

#### 最近の変更を取得

```bash
GET /api/admin/changes/recent?limit=100
```

**パラメータ**:
- `limit`: 取得件数（デフォルト: 100）

**レスポンス例**:
```json
{
  "changes": [
    {
      "id": 1,
      "property_id": "abc123...",
      "change_type": "rent_changed",
      "old_value": "80000",
      "new_value": "85000",
      "change_magnitude": 5000,
      "detected_at": "2025-12-17T10:00:00Z"
    }
  ],
  "count": 45
}
```

**使用例**:
```bash
curl "http://localhost:8084/api/admin/changes/recent?limit=50"
```

---

## 🔒 セキュリティ

### 本番環境での実装推奨

#### 1. Basic認証の追加

```go
// middleware.go
func BasicAuth() gin.HandlerFunc {
    return gin.BasicAuth(gin.Accounts{
        "admin": "your-secure-password",
    })
}

// main.go
admin := r.Group("/api/admin")
admin.Use(BasicAuth())
{
    // routes...
}
```

#### 2. JWT認証の追加

```go
// より安全な方法
admin := r.Group("/api/admin")
admin.Use(JWTMiddleware())
{
    // routes...
}
```

#### 3. IP制限

```go
func IPWhitelist(allowedIPs []string) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        // Check if IP is allowed
    }
}
```

---

## 📊 運用シナリオ

### シナリオ1: 定期的な統計確認

```bash
# 毎朝、システム状態を確認
curl http://localhost:8084/api/admin/stats

# エリア別の物件数を確認
curl http://localhost:8084/api/admin/area-stats

# 価格分布を確認
curl http://localhost:8084/api/admin/price-distribution
```

---

### シナリオ2: 手動スクレイピング実行

```bash
# 1. 現在の状態を確認
curl http://localhost:8084/api/admin/stats

# 2. スクレイピングを実行
curl -X POST http://localhost:8084/api/admin/scraping/trigger

# 3. ログで進捗を確認
docker logs realestate-backend -f

# 4. 完了後、統計を再確認
curl http://localhost:8084/api/admin/stats
```

---

### シナリオ3: 古いデータのクリーンアップ

```bash
# 1. 現在の削除対象を確認（Dry-run）
curl -X POST http://localhost:8084/api/admin/cleanup/run \
  -H "Content-Type: application/json" \
  -d '{"retention_days": 90, "dry_run": true}'

# 2. 結果を確認して問題なければ実行
curl -X POST http://localhost:8084/api/admin/cleanup/run \
  -H "Content-Type: application/json" \
  -d '{"retention_days": 90, "dry_run": false}'

# 3. 削除ログを確認
curl http://localhost:8084/api/admin/cleanup/logs?limit=50
```

---

### シナリオ4: 物件の変更履歴を確認

```bash
# 1. 最近の変更を確認
curl http://localhost:8084/api/admin/changes/recent?limit=20

# 2. 特定物件の履歴を確認
curl http://localhost:8084/api/admin/properties/abc123.../history
```

---

## 🐛 トラブルシューティング

### 問題1: スクレイピングが開始しない

**症状**: `/api/admin/scraping/trigger` が `Scheduler not available` を返す

**原因**: MySQL/GORMが使用されていない

**解決**:
```bash
# docker-compose.yml で DB_TYPE=mysql を確認
# backend/config/scraper_config.yaml で type: mysql を確認
```

---

### 問題2: 物理削除で「safety check failed」

**症状**: 削除対象が多すぎてエラー

**原因**: `max_deletion_count` を超えている

**解決**:
```bash
# 削除対象を確認
curl -X POST http://localhost:8084/api/admin/cleanup/run \
  -H "Content-Type: application/json" \
  -d '{"retention_days": 90, "dry_run": true}'

# max_deletion_count を増やすか、retention_days を短くする
curl -X POST http://localhost:8084/api/admin/cleanup/run \
  -H "Content-Type: application/json" \
  -d '{"retention_days": 60, "max_deletion_count": 15000, "dry_run": false}'
```

---

### 問題3: 統計情報が更新されない

**症状**: `/api/admin/stats` の数値が変わらない

**原因**: スクレイピングが実行されていない

**解決**:
```bash
# スクレイピングを手動実行
curl -X POST http://localhost:8084/api/admin/scraping/trigger

# ログで確認
docker logs realestate-backend -f
```

---

## 📚 関連ドキュメント

- **IMPLEMENTATION-STATUS.md**: システム全体の実装状況
- **POC-TEST-GUIDE.md**: PoC検証ガイド
- **docs/TODO.md**: 開発タスク一覧

---

**最終更新**: 2025-12-17
**バージョン**: Phase 4.5 & Phase 6 Complete
