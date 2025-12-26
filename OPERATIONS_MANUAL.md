# 運用マニュアル：事故らないための最終チェックリスト

**最終更新**: 2025-12-22
**対象**: 本番運用開始前〜運用中の監視・対処

---

## 🎯 到達点（現状の確認）

✅ **詳細取得は Worker だけが行い、5件/時で必ず制御される**

- Scheduler は投入専用（直接 scrape しない）
- Worker が唯一の実行者（DetailLimiter 必須通過）
- 404 は全経路で `permanent_fail`（無限リトライなし）
- WAF は自動 cooldown（1h）+ worker pause（5m）

**再発確率**: かなり低い
**今やるべきこと**: "正しく回っている証明" と "詰まり時の逃げ道"

---

## 📋 必須動作確認（最短・運用開始前）

### ステップ1: Worker起動の証明

```bash
# コンテナ起動
docker-compose up -d

# Worker起動ログを確認
docker-compose logs backend | grep "Queue worker started"
```

**期待される出力**:
```
Queue worker started
QueueWorker: Started (poll_interval=30s, max_concurrency=1)
```

✅ この2行が出れば起動OK

---

### ステップ2: キュー投入→処理の一連を1回通す

```bash
# 1. リスト取得でキューに投入（5件のみ）
curl -X POST http://localhost:8084/api/scrape/list \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://realestate.yahoo.co.jp/rent/search/...",
    "limit": 5
  }'
```

**期待されるレスポンス**:
```json
{
  "message": "List page scraped successfully. URLs added to queue.",
  "urls_found": 20,
  "existing": 0,
  "new_to_queue": 5,
  "queue_status": {
    "pending": 5,
    "processing": 0,
    "done": 0,
    "failed": 0
  }
}
```

✅ `new_to_queue: 5` が出ればキュー投入OK

```bash
# 2. Workerの処理を観測（30秒〜数分待つ）
docker-compose logs -f backend | grep -E "(QueueWorker|DetailLimiter)"
```

**期待されるログ**:
```
QueueWorker: Processing id=1 url=https://... attempt=1
[DetailLimiter] caller=worker now=1703250000 next=1703250720 wait=720s reason=hourly_limit
QueueWorker: ✅ Completed id=1 property_id=abc123
```

✅ この流れが5回繰り返されれば処理OK

---

### ステップ3: `/api/queue/stats` で状態確認

```bash
curl http://localhost:8084/api/queue/stats
```

**期待される出力**:
```json
{
  "pending": 0,
  "processing": 0,
  "done": 5,
  "failed": 0,
  "permanent_fail": 0,
  "is_running": true
}
```

**確認ポイント**:
- ✅ `pending` が 0 になる（処理完了）
- ✅ `done` が 5 になる（成功）
- ✅ `is_running: true`（Worker稼働中）

---

## 🔍 日常監視：ここだけ見ればOK

### 毎日1回：キューの溜まりチェック

```bash
# stats確認（1日1回でOK）
curl http://localhost:8084/api/queue/stats
```

**正常な状態**:
```json
{
  "pending": 0-50,      // ← この数字が "増え続けていない" ことを確認
  "processing": 0-1,
  "done": 100+,
  "failed": 0-10,
  "permanent_fail": 0-20,
  "is_running": true
}
```

### 監視KPI（数字の目安）

| 項目 | 正常範囲 | 警告 | 危険 |
|------|---------|------|------|
| `pending` | 0-50 | 50-200 | 200+ |
| `done` | 増加傾向 | 横ばい | 減少 |
| `permanent_fail` | 総数の <10% | 10-30% | 30%+ |
| `failed` (retry待ち) | 0-10 | 10-50 | 50+ |
| `is_running` | `true` | - | `false` |

**判断基準**:

✅ **正常**: pending が減る傾向、done が増える傾向
⚠️ **警告**: pending が 50-200 で横ばい → Workerが追いついていない
🔴 **危険**: pending が 200+ かつ増加中 → 詰まり発生（後述の対処へ）

---

### 週1回：ログ確認（パターン把握）

```bash
# 過去24時間のWorkerログを確認
docker-compose logs --since 24h backend | grep QueueWorker | tail -100
```

**見るべきポイント**:

1. **成功率**: `✅ Completed` の頻度
   - 期待: 5件/時ペース（= 120件/日）

2. **404の頻度**: `Permanent failure (404)` の頻度
   - 正常: 総数の <10%
   - 異常: 30%+ → URL生成ロジックのミスを疑う

3. **WAF検知**: `WAF/circuit breaker detected` の有無
   - 正常: 0回
   - 異常: 1回でも出たら要注意（cooldownが効いているか確認）

---

## 🚨 詰まりパターン別の対処手順

### パターンA: `pending` が増える一方で減らない

**症状**:
```json
// 1日目
{"pending": 50, "done": 100}

// 2日目
{"pending": 120, "done": 150}  // ← pending が増え続ける

// 3日目
{"pending": 250, "done": 180}  // ← 危険
```

**原因候補**:

1. ✅ Workerが落ちている
2. ✅ DetailLimiterが永遠に待っている（時刻バグ）
3. ✅ WAFで cooldown 連発

**対処手順**:

```bash
# 1. Workerが起動しているか確認
docker-compose logs backend | grep "QueueWorker: Started"

# 2. 最新のWorkerログを確認
docker-compose logs --tail 50 backend | grep QueueWorker

# 3. WAF/cooldown の有無を確認
docker-compose logs --since 1h backend | grep -E "(WAF|cooldown)"
```

**判定と対応**:

| 状況 | 判定 | 対応 |
|------|------|------|
| Workerログが出ない | Worker停止 | `docker-compose restart backend` |
| DetailLimiterで長時間待機 | レート制限の問題 | 正常動作（5件/時なので720秒待機は正常） |
| WAFログが頻発 | WAFブロック中 | **何もしない**（自動cooldownで回復を待つ） |

**重要**: WAF連発時は「何もしない」が正解
→ cooldown/pause が自動で効くので、焦って再実行しない

---

### パターンB: `permanent_fail` が急増

**症状**:
```json
{
  "done": 100,
  "permanent_fail": 80  // ← 総数の44%が404
}
```

**原因候補**:

1. ✅ URL生成ロジックのズレ（末尾スラッシュ、`_` プレフィックス等）
2. ✅ Yahoo側で物件が大量削除された（これは正常）

**対処手順**:

```bash
# 1. permanent_failのURLをサンプル確認
docker-compose exec backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT detail_url, last_error FROM detail_scrape_queue WHERE status=\"permanent_fail\" LIMIT 10;"
'
```

**確認ポイント**:

| URL | 判定 | 対応 |
|-----|------|------|
| `https://realestate.yahoo.co.jp/rent/detail/abc...` | 正しい形式 | ブラウザで開いてみる |
| ブラウザで404 | Yahoo側で削除済み | 正常動作（何もしない） |
| ブラウザで表示される | URL生成ミス | scraper.goのURL生成部分を修正 |
| `https://realestate.yahoo.co.jp/rent/detail/_0000abc...` | プレフィックス残ってる | URL正規化を修正 |

**対応が必要なケース**:
- permanent_fail が 30%+ かつ、ブラウザでは表示される
  → scraper.goの `normalizeURL` や ID抽出ロジックを修正

**対応不要なケース**:
- permanent_fail が 10%未満
- ブラウザでも404が出る
  → Yahoo側で物件削除済み（正常動作）

---

### パターンC: WAF検知が出る

**症状**:
```
QueueWorker: WAF/circuit breaker detected for id=456 - entering cooldown
QueueWorker: Pausing for 5 minutes due to WAF detection
```

**原則**: **いじらない**

- DetailLimiterの制限を緩めない（5件/時のまま）
- Workerを再起動しない
- キューを手動で流さない

**確認するだけ**:

```bash
# cooldown/pauseが効いているか確認
docker-compose logs --since 10m backend | grep -E "(cooldown|Pausing)"
```

**期待される動作**:
```
QueueWorker: WAF/circuit breaker detected for id=456 - entering cooldown
next_retry_at: 1時間後
QueueWorker: Pausing for 5 minutes due to WAF detection
```

✅ この2つが出ていれば正常に撤退している

**1時間後に自動復帰**:
- 何もしなくてOK
- Worker が自動で再開する

---

## 🔒 運用ルール（絶対に守る3原則）

### 🔴 禁止事項

| 行為 | 理由 | 結果 |
|------|------|------|
| DetailLimiterを緩める（5件/時→10件/時等） | WAFリスクが跳ね上がる | **即WAF発動** |
| Scheduler/Workerを迂回して直接scrape | レート制限が効かない | **即WAF発動** |
| permanent_failをretryに戻す | 無駄なリトライでリソース消耗 | **WAFリスク上昇** |

### 🟢 推奨事項

| 行為 | 頻度 | 目的 |
|------|------|------|
| `/api/queue/stats` 確認 | 1日1回 | pending増加の早期発見 |
| Workerログ確認 | 週1回 | WAF/404パターンの把握 |
| permanent_failクリーンアップ | 月1回 | ディスク容量節約 |

---

## 📊 監視ダッシュボード（最小構成）

最低限これだけ見る：

```bash
#!/bin/bash
# daily_check.sh - 毎日実行

echo "=== Queue Stats ==="
curl -s http://localhost:8084/api/queue/stats | jq .

echo ""
echo "=== Recent Worker Activity ==="
docker-compose logs --since 24h backend | grep "QueueWorker: ✅" | wc -l
echo "^ 成功件数（期待: 120件/日 = 5件/時）"

echo ""
echo "=== WAF Detections ==="
docker-compose logs --since 24h backend | grep -c "WAF"
echo "^ WAF検知回数（期待: 0回）"

echo ""
echo "=== 404 Count ==="
docker-compose logs --since 24h backend | grep -c "Permanent failure (404)"
echo "^ 404件数（期待: 成功件数の <10%）"
```

**使い方**:
```bash
chmod +x daily_check.sh
./daily_check.sh
```

**期待される出力**:
```
=== Queue Stats ===
{
  "pending": 12,
  "done": 450,
  "permanent_fail": 25,
  "is_running": true
}

=== Recent Worker Activity ===
120
^ 成功件数（期待: 120件/日 = 5件/時）

=== WAF Detections ===
0
^ WAF検知回数（期待: 0回）

=== 404 Count ===
8
^ 404件数（期待: 成功件数の <10%）
```

---

## 🎓 次のステップ（オプション・余裕があれば）

### 優先度: 中（WAFが頻発する場合のみ）

**WAFヘルスチェック機能の追加**

Worker起動時に1回だけ軽いリクエストを送り、WAFが生きているか確認:

```go
// worker.go の Start() に追加
func (w *QueueWorker) Start() {
    // WAF Health Check
    if !w.healthCheck() {
        log.Println("QueueWorker: WAF health check failed, delaying start by 1 hour")
        time.Sleep(1 * time.Hour)
    }

    // 通常の起動処理
    w.isRunning = true
    go w.run()
}

func (w *QueueWorker) healthCheck() bool {
    // 軽いリクエストを1回だけ送る
    testURL := "https://realestate.yahoo.co.jp/rent/"
    req, _ := http.NewRequest("GET", testURL, nil)
    applyBrowserHeaders(req, "")

    resp, err := w.scraper.client.Do(req)
    if err != nil || resp.StatusCode >= 500 {
        return false // WAFブロック中
    }
    return true // 正常
}
```

**効果**: Worker起動直後のWAF遭遇を回避

---

### 優先度: 低（運用が安定してから）

**予防的クールダウン**

成功が続いても一定時間ごとに強制pause:

```go
// worker.go の processQueueItem() に追加
successCount := 0

func (w *QueueWorker) processQueueItem(item *models.DetailScrapeQueue) {
    // ... 既存処理 ...

    if success {
        successCount++

        // 10件処理したら5分休む（予防）
        if successCount%10 == 0 {
            log.Println("QueueWorker: Preventive cooldown (10 successes)")
            time.Sleep(5 * time.Minute)
        }
    }
}
```

**効果**: 「攻めすぎ」による突然のWAFを防ぐ

---

## 🎯 最終チェックリスト（本番投入前）

運用開始前に全てチェック:

- [ ] `docker-compose up -d` でエラーなし
- [ ] `Queue worker started` がログに出る
- [ ] `/api/queue/stats` で `is_running: true`
- [ ] リスト取得でキューに投入される（`new_to_queue > 0`）
- [ ] Workerがキューを消化する（`done` が増える）
- [ ] DetailLimiterが働く（ログに `wait=720s` 等）
- [ ] 404が `permanent_fail` になる
- [ ] WAF検知時に `cooldown` + `pause` が動く
- [ ] `daily_check.sh` で異常なし

---

## ✅ まとめ

### やること
- **毎日**: `/api/queue/stats` で pending が増え続けていないか確認
- **週1**: ログで WAF/404 の傾向確認
- **月1**: permanent_fail のクリーンアップ

### やらないこと
- DetailLimiter を緩める
- Worker を迂回する
- permanent_fail を retry に戻す

### 詰まった時
- **pending増加**: Workerログ確認 → WAFなら待つ、停止なら再起動
- **404急増**: URLをブラウザ確認 → 生成ミスなら修正、Yahoo削除なら放置
- **WAF検知**: **何もしない** → 自動で cooldown/pause が効く

---

**この運用を守れば、長期間安定して動きます。**
