# Phase 3: 差分検出・毎日更新 - 実装設計

## 概要

毎日の一覧取得 → 差分判定 → 新規詳細取得 → 消滅処理を安全に実行する設計。

**最重要原則**: **一覧取得が不完全な場合は削除処理をスキップする**

---

## 処理フロー全体図

```
┌─────────────────────────────────────┐
│  1. 一覧取得（全エリア）             │
│     - エリアごとに物件ID一覧を取得   │
│     - daily_property_snapshots保存  │
│     - 完全性フラグを記録             │
└─────────┬───────────────────────────┘
          │
          ▼
┌─────────────────────────────────────┐
│  2. 完全性チェック                   │
│     - 全エリア成功？                 │
│     - YES → 差分判定へ               │
│     - NO  → 削除スキップ、新規のみ   │
└─────────┬───────────────────────────┘
          │
          ▼
┌─────────────────────────────────────┐
│  3. 差分判定                         │
│     - 新規物件: 詳細取得対象         │
│     - 継続物件: 何もしない           │
│     - 消滅物件: 削除対象（完全時のみ）│
└─────────┬───────────────────────────┘
          │
          ▼
┌─────────────────────────────────────┐
│  4. 新規物件の詳細取得               │
│     - 2秒間隔で順次取得              │
│     - エラー時即停止                 │
└─────────┬───────────────────────────┘
          │
          ▼
┌─────────────────────────────────────┐
│  5. 消滅物件の論理削除               │
│     - properties.status = 'removed' │
│     - Meilisearchから削除            │
│     - 完全性チェックOK時のみ実行     │
└─────────────────────────────────────┘
```

---

## Go 擬似コード（実装に近い形）

### 1. メインバッチ処理

```go
package batch

import (
    "context"
    "fmt"
    "time"
)

// DailyScrapeJob は毎日実行されるスクレイピングジョブ
func DailyScrapeJob(ctx context.Context) error {
    executionDate := time.Now().Format("2006-01-02")
    log.Info("=== Daily Scrape Job Started ===", "date", executionDate)

    // Step 1: 全エリアの一覧取得
    completeness, err := fetchAllAreaLists(ctx, executionDate)
    if err != nil {
        log.Error("一覧取得でエラー発生", "error", err)
        recordLog(executionDate, "list", "failed", err.Error())
        return err
    }

    // Step 2: 完全性チェック
    if !completeness.IsComplete() {
        log.Warn("一覧取得が不完全のため削除処理をスキップ",
            "total", completeness.Total,
            "success", completeness.Success,
            "failed", completeness.Failed)
        recordLog(executionDate, "list", "partial",
            fmt.Sprintf("成功: %d/%d エリア", completeness.Success, completeness.Total))
    }

    // Step 3: 差分判定
    diffResult, err := detectDifference(ctx, executionDate)
    if err != nil {
        log.Error("差分判定でエラー発生", "error", err)
        return err
    }

    log.Info("差分判定結果",
        "新規", len(diffResult.NewIDs),
        "継続", len(diffResult.ContinuingIDs),
        "消滅", len(diffResult.RemovedIDs))

    // Step 4: 新規物件の詳細取得
    if len(diffResult.NewIDs) > 0 {
        err = fetchNewPropertyDetails(ctx, diffResult.NewIDs, executionDate)
        if err != nil {
            log.Error("詳細取得でエラー発生", "error", err)
            return err
        }
    }

    // Step 5: 消滅物件の論理削除（完全時のみ）
    if completeness.IsComplete() && len(diffResult.RemovedIDs) > 0 {
        err = markPropertiesAsRemoved(ctx, diffResult.RemovedIDs, executionDate)
        if err != nil {
            log.Error("削除処理でエラー発生", "error", err)
            return err
        }
    } else if !completeness.IsComplete() {
        log.Warn("一覧取得が不完全のため削除処理をスキップしました")
    }

    log.Info("=== Daily Scrape Job Completed ===")
    return nil
}
```

---

### 2. 一覧取得（エリア単位）

```go
type Completeness struct {
    Total   int
    Success int
    Failed  int
    FailedAreas []string
}

func (c *Completeness) IsComplete() bool {
    return c.Failed == 0
}

// fetchAllAreaLists は全エリアの物件ID一覧を取得
func fetchAllAreaLists(ctx context.Context, executionDate string) (*Completeness, error) {
    // 有効なエリア一覧を取得
    areas, err := db.GetEnabledAreas(ctx)
    if err != nil {
        return nil, fmt.Errorf("エリア一覧取得エラー: %w", err)
    }

    completeness := &Completeness{
        Total:       len(areas),
        Success:     0,
        Failed:      0,
        FailedAreas: []string{},
    }

    for _, area := range areas {
        log.Info("エリアの一覧取得開始", "area", area.Name)

        // Yahoo不動産の検索結果ページをスクレイピング
        propertyIDs, err := scraper.FetchPropertyListByArea(area.SearchURL)
        if err != nil {
            log.Error("一覧取得失敗", "area", area.Name, "error", err)
            completeness.Failed++
            completeness.FailedAreas = append(completeness.FailedAreas, area.AreaCode)

            // エラー記録
            db.UpdateAreaStatus(ctx, area.AreaCode, "failed", time.Now())
            continue
        }

        // daily_property_snapshotsに保存
        err = db.SaveSnapshot(ctx, executionDate, area.AreaCode, propertyIDs)
        if err != nil {
            log.Error("スナップショット保存失敗", "area", area.Name, "error", err)
            completeness.Failed++
            completeness.FailedAreas = append(completeness.FailedAreas, area.AreaCode)
            continue
        }

        completeness.Success++
        db.UpdateAreaStatus(ctx, area.AreaCode, "success", time.Now())

        log.Info("エリアの一覧取得完了",
            "area", area.Name,
            "count", len(propertyIDs))

        // レート制限: 次のエリアまで2秒待機
        time.Sleep(2 * time.Second)
    }

    return completeness, nil
}
```

---

### 3. 差分判定

```go
type DifferenceResult struct {
    NewIDs        []PropertyID  // 新規物件
    ContinuingIDs []PropertyID  // 継続物件
    RemovedIDs    []PropertyID  // 消滅物件
}

// detectDifference は昨日と今日のsnapshotを比較
func detectDifference(ctx context.Context, today string) (*DifferenceResult, error) {
    yesterday := getYesterday(today)

    // 今日のスナップショット
    todayIDs, err := db.GetSnapshotIDs(ctx, today)
    if err != nil {
        return nil, fmt.Errorf("今日のスナップショット取得エラー: %w", err)
    }

    // 昨日のスナップショット
    yesterdayIDs, err := db.GetSnapshotIDs(ctx, yesterday)
    if err != nil {
        // 昨日のデータがない場合（初回実行）
        if errors.Is(err, sql.ErrNoRows) {
            log.Info("昨日のスナップショットなし（初回実行）")
            return &DifferenceResult{
                NewIDs:        todayIDs,
                ContinuingIDs: []PropertyID{},
                RemovedIDs:    []PropertyID{},
            }, nil
        }
        return nil, fmt.Errorf("昨日のスナップショット取得エラー: %w", err)
    }

    // 差分計算
    result := calculateDifference(todayIDs, yesterdayIDs)
    return result, nil
}

func calculateDifference(today, yesterday []PropertyID) *DifferenceResult {
    todaySet := make(map[PropertyID]bool)
    yesterdaySet := make(map[PropertyID]bool)

    for _, id := range today {
        todaySet[id] = true
    }
    for _, id := range yesterday {
        yesterdaySet[id] = true
    }

    result := &DifferenceResult{
        NewIDs:        []PropertyID{},
        ContinuingIDs: []PropertyID{},
        RemovedIDs:    []PropertyID{},
    }

    // 新規: 今日にあって昨日にない
    for _, id := range today {
        if !yesterdaySet[id] {
            result.NewIDs = append(result.NewIDs, id)
        } else {
            result.ContinuingIDs = append(result.ContinuingIDs, id)
        }
    }

    // 消滅: 昨日にあって今日にない
    for _, id := range yesterday {
        if !todaySet[id] {
            result.RemovedIDs = append(result.RemovedIDs, id)
        }
    }

    return result
}
```

---

### 4. 新規物件の詳細取得

```go
// fetchNewPropertyDetails は新規物件の詳細をスクレイピング
func fetchNewPropertyDetails(ctx context.Context, newIDs []PropertyID, executionDate string) error {
    successCount := 0
    errorCount := 0

    for i, propertyID := range newIDs {
        log.Info("新規物件の詳細取得",
            "progress", fmt.Sprintf("%d/%d", i+1, len(newIDs)),
            "id", propertyID)

        // 詳細URLを取得（snapshotから）
        detailURL, err := db.GetDetailURLFromSnapshot(ctx, executionDate, propertyID)
        if err != nil {
            log.Error("詳細URL取得エラー", "id", propertyID, "error", err)
            errorCount++
            continue
        }

        // スクレイピング実行
        property, err := scraper.ScrapeProperty(detailURL)
        if err != nil {
            log.Error("スクレイピング失敗", "id", propertyID, "error", err)
            errorCount++

            // 403/429/5xx の場合は即停止
            if isTerminalError(err) {
                return fmt.Errorf("致命的エラー検出、処理を中止: %w", err)
            }
            continue
        }

        // DBに保存
        err = db.SaveProperty(ctx, property)
        if err != nil {
            log.Error("DB保存失敗", "id", propertyID, "error", err)
            errorCount++
            continue
        }

        // Meilisearchにインデックス
        err = search.IndexProperty(property)
        if err != nil {
            log.Error("インデックス失敗", "id", propertyID, "error", err)
            // 検索インデックスは後で修復可能なので続行
        }

        successCount++

        // レート制限: 2秒待機
        time.Sleep(2 * time.Second)
    }

    log.Info("新規物件の詳細取得完了",
        "total", len(newIDs),
        "success", successCount,
        "error", errorCount)

    recordLog(executionDate, "detail", "completed",
        fmt.Sprintf("成功: %d, 失敗: %d", successCount, errorCount))

    return nil
}

func isTerminalError(err error) bool {
    // 403 Forbidden
    // 429 Too Many Requests
    // 5xx Server Error
    // これらが返ってきたら即停止
    return strings.Contains(err.Error(), "403") ||
           strings.Contains(err.Error(), "429") ||
           strings.Contains(err.Error(), "5")
}
```

---

### 5. 消滅物件の論理削除

```go
// markPropertiesAsRemoved は消滅物件を論理削除
// ⚠️ 一覧取得が完全な場合のみ実行される
func markPropertiesAsRemoved(ctx context.Context, removedIDs []PropertyID, executionDate string) error {
    tx, err := db.Begin(ctx)
    if err != nil {
        return fmt.Errorf("トランザクション開始エラー: %w", err)
    }
    defer tx.Rollback()

    now := time.Now()
    successCount := 0

    for _, propertyID := range removedIDs {
        // properties.status = 'removed'
        err := tx.UpdatePropertyStatus(ctx, propertyID, "removed", now)
        if err != nil {
            log.Error("物件ステータス更新失敗", "id", propertyID, "error", err)
            continue
        }

        // Meilisearchから削除
        err = search.DeleteProperty(propertyID)
        if err != nil {
            log.Error("検索インデックス削除失敗", "id", propertyID, "error", err)
            // 検索は後で修復可能なので続行
        }

        successCount++
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("トランザクションコミットエラー: %w", err)
    }

    log.Info("消滅物件の削除完了",
        "total", len(removedIDs),
        "success", successCount)

    recordLog(executionDate, "removal", "completed",
        fmt.Sprintf("削除: %d件", successCount))

    return nil
}
```

---

### 6. ログ記録

```go
// recordLog はscrape_logsテーブルに記録
func recordLog(executionDate, phase, status, message string) error {
    log := ScrapeLog{
        ExecutionDate: executionDate,
        Phase:         phase,  // "list", "detail", "diff", "removal"
        Status:        status, // "started", "completed", "failed", "partial"
        ErrorMessage:  message,
        StartedAt:     time.Now(),
        CompletedAt:   time.Now(),
    }

    return db.SaveScrapeLog(context.Background(), &log)
}
```

---

## データベースクエリ例

### 差分判定SQL

```sql
-- 新規物件（今日にあって昨日にない）
SELECT t.property_id
FROM daily_property_snapshots t
LEFT JOIN daily_property_snapshots y
  ON t.property_id = y.property_id
  AND y.snapshot_date = '2025-12-15'
WHERE t.snapshot_date = '2025-12-16'
  AND y.property_id IS NULL;

-- 消滅物件（昨日にあって今日にない）
SELECT y.property_id
FROM daily_property_snapshots y
LEFT JOIN daily_property_snapshots t
  ON y.property_id = t.property_id
  AND t.snapshot_date = '2025-12-16'
WHERE y.snapshot_date = '2025-12-15'
  AND t.property_id IS NULL;
```

### 論理削除

```sql
UPDATE properties
SET status = 'removed',
    removed_at = NOW()
WHERE id IN (
  -- 消滅物件のIDリスト
);
```

---

## エラーハンドリング戦略

| エラー | 対応 |
|-------|------|
| 一部エリア失敗 | ログ記録、削除スキップ、新規取得は継続 |
| 403/429/5xx | 即座に全処理停止 |
| DB接続エラー | 即座に全処理停止 |
| Meilisearchエラー | ログ記録、処理は継続 |
| 詳細取得失敗（個別） | ログ記録、次の物件へ |

---

## スケジューラー設定例

### cron

```go
// 毎日 2:00 AM に実行
c := cron.New()
c.AddFunc("0 2 * * *", func() {
    ctx := context.Background()
    if err := DailyScrapeJob(ctx); err != nil {
        log.Error("Daily scrape job failed", "error", err)
        // アラート送信
    }
})
c.Start()
```

### asynq

```go
// タスクをエンキュー
task := asynq.NewTask("daily_scrape", nil)
client.Enqueue(task, asynq.ProcessAt(nextRunTime))

// ワーカー
mux := asynq.NewServeMux()
mux.HandleFunc("daily_scrape", func(ctx context.Context, t *asynq.Task) error {
    return DailyScrapeJob(ctx)
})
```

---

## 安全性チェックリスト

- [ ] 一覧取得が不完全な場合、削除処理をスキップする
- [ ] 物理削除は絶対にしない（論理削除のみ）
- [ ] 403/429/5xx発生時は即停止
- [ ] 全エリアの処理開始前にログ記録
- [ ] トランザクションで削除処理を保護
- [ ] Meilisearch失敗時も処理継続（後で修復可能）
- [ ] レート制限を必ず守る（2秒間隔）
- [ ] エラーログを必ず記録

---

## 次のステップ

1. **GORM モデル定義** を作成
2. **DB層の実装** （snapshot保存、差分判定クエリ）
3. **スクレイパーの一覧取得機能** を追加
4. **スケジューラー選定** （cron or asynq）
5. **統合テスト** （小規模エリアで試験実行）

---

**最終更新**: 2025-12-16
