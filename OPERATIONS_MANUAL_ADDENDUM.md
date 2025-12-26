# 運用マニュアル追記（重要な現実的補足）

**追記日**: 2025-12-22
**目的**: 本番運用で事故らないための穴埋め

---

## ⚠️ 重要な補足（OPERATIONS_MANUAL.md への追記内容）

### 1. WAFリスクの正確な表現

**修正前**:
> WAFリスク: ほぼゼロ

**修正後**:
> **WAFリスクを最小化**（5件/時厳守 + cooldown/pause + backoff）
>
> ⚠️ **注意**: 5件/時でもWAF検知は起こり得ます（IP評価・User-Agent・cookie・時間帯などで変動）。
> ただし、**WAF検知時に自動撤退する**ため、深刻化しにくい設計です。

---

### 2. 仕様の明確化（将来の誤変更防止）

**OPERATIONS_MANUAL.md の冒頭に追加**:

```markdown
## 🔒 絶対に守る仕様（変更禁止）

### 詳細取得の経路制限

**✅ 許可されている詳細取得**:
- Queue Worker のみ（worker.go）
- `/api/scrape`（単発テスト用のみ、DetailLimiter必須）

**❌ 禁止されている詳細取得**:
- **`/api/scrape/list` は詳細取得しない（Queue投入のみ）**
- **Scheduler は詳細取得しない（Queue投入のみ）**
- **scrapeAndUpdate からの直接呼び出し（Queue推奨）**

**理由**: 詳細取得を Worker 以外が行うと、DetailLimiter の制御が効かなくなり、WAF発動リスクが跳ね上がります。
```

---

### 3. DetailLimiterの待機時間は正常（誤解防止）

**監視セクションに追加**:

```markdown
### DetailLimiter の待機時間（正常範囲）

DetailLimiter のログで `wait_sec` が出るのは**正常動作**です。

**正常範囲**:
- `wait_sec`: 0〜900秒（0〜15分）
  - 5件/時 = 平均12分間隔なので、待機は普通

**注意が必要**:
- `wait_sec`: 3600秒（1時間）が連発
  - → 時刻計算バグまたは集計ミスの可能性
  - → ログで `now_epoch` と `next_epoch` を確認

**例（正常）**:
```
[DetailLimiter] caller=worker now=1703250000 next=1703250720 wait=720s reason=hourly_limit
```
→ 720秒（12分）待機は正常
```

---

### 4. 動作確認の成功条件（現実的基準）

**ステップ2の期待結果を修正**:

```markdown
### ステップ2: 成功の最低ライン

✅ **必須**:
1. QueueWorker が起動している
2. `/api/scrape/list` で `pending` が増える
3. Workerが 1件でも処理して、以下のいずれかに落ちる:
   - ✅ `success`
   - ⚠️ `permanent_fail` (404)
   - ⏱️ `retry` (WAF/500等)
   - 🔴 `cooldown` (WAF検知)

❌ **期待しすぎない**:
- タイトル取得: "No Title" が残る可能性あり
  - 原因: WAF/構造差/JS依存ページ
  - → 致命的エラーではない

- 404発生: 一定数（<10%）は正常
  - 原因: Yahoo側で物件削除済み
  - → 30%+ なら URL生成ロジックを疑う

**判定**:
- ✅ 正常: 1件でも `success` または `permanent_fail(404)` が出る
- ⚠️ 要確認: 全て `retry` または `cooldown` → WAF検知中
- 🔴 異常: 何も処理されない → Worker停止を疑う
```

---

### 5. 監視指標の追加（最古pending滞留時間）

**daily_check.sh に追加すべき指標**:

```markdown
### 監視KPI（改訂版）

| 項目 | 正常範囲 | 警告 | 危険 | 確認方法 |
|------|---------|------|------|---------|
| `pending` | 0-50 | 50-200 | 200+ | `/api/queue/stats` |
| `success/24h` | 100-120 | 50-100 | <50 | ログ集計 |
| **最古pending滞留時間** | <12h | 12-48h | 48h+ | DB直接確認 |
| `permanent_fail率` | <10% | 10-30% | 30%+ | stats計算 |
| `WAF検知/週` | 0回 | 1回 | 2回+ | ログ確認 |

**最古pending確認コマンド**:
```bash
docker-compose exec backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT id, created_at, TIMESTAMPDIFF(HOUR, created_at, NOW()) as hours_old
      FROM detail_scrape_queue
      WHERE status=\"pending\"
      ORDER BY created_at ASC
      LIMIT 1;"
'
```

**判定**:
- <12h: 正常（5件/時なので遅延は普通）
- 12-48h: 要監視（Worker処理速度不足）
- 48h+: 詰まり確定（Worker停止またはWAF連発）
```

---

### 6. WAF連発時の対応（"速度を下げる"ではなく"待つ"）

**パターンC: WAF検知が出る（修正版）**:

```markdown
### パターンC: WAF検知が出る

**症状**:
```
QueueWorker: WAF/circuit breaker detected for id=456 - entering cooldown
QueueWorker: Pausing for 5 minutes due to WAF detection
```

**原則**: **いじらない・待つ**

❌ **やってはいけないこと**:
- DetailLimiterの制限を緩める
- Workerを再起動する
- キューを手動で流す
- **「速度を下げる」ためにWorkerを止める** ← これは不要

✅ **やるべきこと**:
1. cooldown/pauseが効いているか確認（ログで）
2. **何もせず1時間待つ**
3. 1時間後に自動復帰を確認

**重要**: WAF連発時の正解は「**待つ**」
- cooldown (1h) + pause (5m) が自動で効く
- "速度を下げる" = Worker停止は不要（既に十分遅い）
- 週2回以上WAFが出る場合のみ、DetailLimiterを 4件/時 に下げることを検討
  → ただし、まずは1週間様子見が推奨
```

---

### 7. 最終チェックリストに追加

**OPERATIONS_MANUAL.md の最終チェックリストに追加**:

```markdown
## 🎯 最終チェックリスト（本番投入前）

運用開始前に全てチェック:

- [ ] `docker-compose up -d` でエラーなし
- [ ] `Queue worker started` がログに出る
- [ ] **Worker が 1プロセスだけ動いている（多重起動なし）**
- [ ] `/api/queue/stats` で `is_running: true`
- [ ] リスト取得でキューに投入される（`new_to_queue > 0`）
- [ ] Workerがキューを消化する（`done` が増える）
- [ ] DetailLimiterが働く（ログに `wait=0-900s` 程度）
- [ ] **Scheduler が 100件/回の上限を守っている**
- [ ] 404が `permanent_fail` になる
- [ ] WAF検知時に `cooldown` + `pause` が動く
- [ ] **cooldown中にWorkerが暴れない（pause/cooldownが効く）**
- [ ] `daily_check.sh` で異常なし
```

---

### 8. WAFヘルスチェックの推奨（効果大）

**次のステップ（オプション）に追加**:

```markdown
### 優先度: 高（WAFが1回でも出たら実装推奨）

**WAFヘルスチェック機能の追加**

Worker起動時に1回だけ軽いリクエストを送り、WAFが生きているか確認:

**実装場所**: `backend/internal/scheduler/worker.go`

```go
// Start() の最初に追加
func (w *QueueWorker) Start() {
    if w.isRunning {
        return
    }

    // WAF Health Check（起動前に1回だけ）
    log.Println("QueueWorker: Running WAF health check...")
    if !w.healthCheck() {
        log.Println("QueueWorker: WAF detected, delaying start by 1 hour")
        time.Sleep(1 * time.Hour)
    } else {
        log.Println("QueueWorker: Health check passed")
    }

    w.isRunning = true
    log.Printf("QueueWorker: Started (poll_interval=%v, max_concurrency=%d)",
        w.pollInterval, w.maxConcurrency)
    go w.run()
}

// healthCheck は軽いリクエストでWAFを確認
func (w *QueueWorker) healthCheck() bool {
    testURL := "https://realestate.yahoo.co.jp/rent/"
    req, err := http.NewRequest("GET", testURL, nil)
    if err != nil {
        log.Printf("QueueWorker: Health check request creation failed: %v", err)
        return false
    }

    // Browser-like headers
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("QueueWorker: Health check failed: %v", err)
        return false
    }
    defer resp.Body.Close()

    // Check for WAF block
    if resp.StatusCode >= 500 {
        body, _ := io.ReadAll(resp.Body)
        if strings.Contains(string(body), "ご覧になろうとしているページは現在表示できません") {
            log.Printf("QueueWorker: WAF block detected in health check")
            return false
        }
    }

    log.Printf("QueueWorker: Health check OK (status: %d)", resp.StatusCode)
    return true
}
```

**効果**:
- Worker起動直後のWAF遭遇を回避
- 無駄打ちを防ぎ、IP評価を悪化させにくい
- cooldown明けの再起動時に特に有効

**実装タイミング**:
- 初回運用で1週間様子見
- WAFが1回でも出たら実装推奨
```

---

## 📊 改訂版 daily_check.sh

より現実的な判定基準を追加したバージョン:

```bash
#!/bin/bash
# daily_check.sh - 改訂版（現実的基準）

set -e

echo "=========================================="
echo "   Queue Worker Daily Health Check"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo ""

# 1. Queue Stats
echo "=== Queue Stats ==="
STATS=$(curl -s http://localhost:8084/api/queue/stats)
echo "$STATS" | jq .

PENDING=$(echo "$STATS" | jq -r '.pending // 0')
DONE=$(echo "$STATS" | jq -r '.done // 0')
PERMANENT_FAIL=$(echo "$STATS" | jq -r '.permanent_fail // 0')
FAILED=$(echo "$STATS" | jq -r '.failed // 0')
IS_RUNNING=$(echo "$STATS" | jq -r '.is_running // false')

echo ""
echo "=== Status Check ==="

# Worker running check
if [ "$IS_RUNNING" = "true" ]; then
    echo "✅ Worker is running"
else
    echo "❌ Worker is NOT running - check logs!"
fi

# Pending queue check
if [ "$PENDING" -lt 50 ]; then
    echo "✅ Pending queue is healthy ($PENDING items)"
elif [ "$PENDING" -lt 200 ]; then
    echo "⚠️  Pending queue is elevated ($PENDING items) - monitor closely"
else
    echo "🔴 Pending queue is HIGH ($PENDING items) - investigate immediately!"
fi

# 404 ratio check
TOTAL=$((DONE + PERMANENT_FAIL))
if [ "$TOTAL" -gt 0 ]; then
    FAIL_RATIO=$((PERMANENT_FAIL * 100 / TOTAL))
    if [ "$FAIL_RATIO" -lt 10 ]; then
        echo "✅ 404 ratio is healthy ($FAIL_RATIO%)"
    elif [ "$FAIL_RATIO" -lt 30 ]; then
        echo "⚠️  404 ratio is elevated ($FAIL_RATIO%) - check URL generation"
    else
        echo "🔴 404 ratio is HIGH ($FAIL_RATIO%) - URL generation likely broken!"
    fi
fi

echo ""
echo "=== Recent Activity (Last 24h) ==="

# Success count
SUCCESS_COUNT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "QueueWorker: ✅" || echo "0")
echo "✅ Successful scrapes: $SUCCESS_COUNT"
if [ "$SUCCESS_COUNT" -eq 0 ]; then
    echo "   🔴 ZERO successes - Worker may be stuck!"
elif [ "$SUCCESS_COUNT" -lt 50 ]; then
    echo "   ⚠️  Low activity (expected: 100-120/day)"
else
    echo "   ✅ Normal activity (expected: 100-120/day)"
fi

# WAF detections
WAF_COUNT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "WAF" || echo "0")
if [ "$WAF_COUNT" -eq 0 ]; then
    echo "✅ WAF detections: 0 (good)"
else
    echo "🔴 WAF detections: $WAF_COUNT (check cooldown is working)"
fi

# Oldest pending check (NEW)
echo ""
echo "=== Oldest Pending Item ==="
OLDEST=$(docker-compose exec -T backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db -N \
  -e "SELECT TIMESTAMPDIFF(HOUR, MIN(created_at), NOW())
      FROM detail_scrape_queue
      WHERE status=\"pending\";" 2>/dev/null' || echo "N/A")

if [ "$OLDEST" = "N/A" ] || [ -z "$OLDEST" ]; then
    echo "ℹ️  No pending items (or DB not accessible)"
elif [ "$OLDEST" -lt 12 ]; then
    echo "✅ Oldest pending: ${OLDEST}h (normal delay)"
elif [ "$OLDEST" -lt 48 ]; then
    echo "⚠️  Oldest pending: ${OLDEST}h (slow processing)"
else
    echo "🔴 Oldest pending: ${OLDEST}h (STUCK - investigate!)"
fi

echo ""
echo "=========================================="
echo "   Summary"
echo "=========================================="

HEALTH_SCORE=0
[ "$IS_RUNNING" = "true" ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))
[ "$SUCCESS_COUNT" -gt 0 ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))
[ "$PENDING" -lt 200 ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))
[ "$WAF_COUNT" -eq 0 ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))

if [ $HEALTH_SCORE -eq 4 ]; then
    echo "✅ System is HEALTHY - no action needed"
elif [ $HEALTH_SCORE -ge 2 ]; then
    echo "⚠️  System needs MONITORING - check warnings above"
else
    echo "🔴 System needs ATTENTION - review errors and take action"
fi

echo ""
```

---

## 📝 使い方

### この追記ドキュメントの適用方法

1. **OPERATIONS_MANUAL.md に手動で追記**
   - 各セクションの該当箇所に追記内容をコピペ

2. **daily_check.sh を改訂版に置き換え**
   ```bash
   # 上記の改訂版スクリプトで上書き
   ```

3. **worker.go にWAFヘルスチェック追加（推奨）**
   - 初回運用後、WAFが出たら実装

---

**これで本番運用の事故リスクがさらに下がります。**
