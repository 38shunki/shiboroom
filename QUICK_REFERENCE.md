# クイックリファレンス

**最終更新**: 2025-12-22

---

## 🚀 1分でわかる：システムの仕組み

```
リスト取得 (/api/scrape/list)
    ↓ URLを抽出
    ↓ キューに投入（詳細は取らない）
    ↓
detail_scrape_queue テーブル
    ↓ 30秒ごとにポーリング
    ↓
Queue Worker ← DetailLimiter (5件/時)
    ↓
    ├─ 成功 → properties + snapshot
    ├─ 404 → permanent_fail (終了)
    ├─ WAF → cooldown 1h + pause 5m
    └─ 他 → retry (5m→15m→1h→4h→12h)
```

**重要**: 詳細取得は **Worker のみ**。他の経路は全て投入だけ。

---

## 📝 よく使うコマンド

### 起動・停止

```bash
# 起動
docker-compose up -d

# 停止
docker-compose down

# 再起動（Workerが止まった時）
docker-compose restart backend

# ログ確認（リアルタイム）
docker-compose logs -f backend | grep -E "(Scheduler|QueueWorker|DetailLimiter)"
```

---

### 監視

```bash
# 毎日実行（推奨）
./daily_check.sh

# キューの状態確認
curl http://localhost:8084/api/queue/stats | jq .

# 過去24時間の成功件数
docker-compose logs --since 24h backend | grep "QueueWorker: ✅" | wc -l

# WAF検知の有無
docker-compose logs --since 24h backend | grep -c "WAF"

# 404の件数
docker-compose logs --since 24h backend | grep -c "Permanent failure (404)"
```

---

### トラブルシューティング

```bash
# Workerが動いているか確認
docker-compose logs backend | grep "Queue worker started"

# 最新のエラー確認
docker-compose logs --tail 50 backend | grep -E "(ERROR|Failed)"

# キューの中身を直接確認（MySQL）
docker-compose exec backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT status, COUNT(*) FROM detail_scrape_queue GROUP BY status;"
'

# permanent_failのURLサンプル
docker-compose exec backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT detail_url FROM detail_scrape_queue WHERE status=\"permanent_fail\" LIMIT 5;"
'
```

---

## 🎯 状態別：すぐやること

### ✅ 正常（何もしない）

```json
{
  "pending": 10,
  "processing": 0,
  "done": 500,
  "failed": 5,
  "permanent_fail": 20,
  "is_running": true
}
```

→ pending が減る傾向、done が増える傾向なら正常

---

### ⚠️ pending が増えている

```json
{
  "pending": 150,  // ← 増加中
  "done": 100,
  "is_running": true
}
```

**やること**:
1. Workerログ確認: `docker-compose logs --tail 50 backend | grep QueueWorker`
2. WAF検知の有無: `docker-compose logs --since 1h backend | grep WAF`
3. WAFが出ていたら **何もしない**（自動でcooldown）
4. Workerが止まっていたら再起動: `docker-compose restart backend`

---

### 🔴 404が30%以上

```json
{
  "done": 100,
  "permanent_fail": 50  // ← 33%
}
```

**やること**:
1. URLをサンプル確認:
   ```bash
   docker-compose exec backend sh -c '
     mysql -u realestate_user -prealestate_pass realestate_db \
     -e "SELECT detail_url FROM detail_scrape_queue WHERE status=\"permanent_fail\" LIMIT 3;"
   '
   ```
2. ブラウザで開いてみる
3. ブラウザでも404 → 正常（Yahoo側で削除済み）
4. ブラウザで表示される → URL生成バグ（scraper.go修正）

---

### 🔴 WAF検知が出た

```
QueueWorker: WAF/circuit breaker detected for id=123
```

**やること**: **何もしない**

- 自動で1時間cooldown + 5分pause
- DetailLimiterは緩めない（5件/時のまま）
- 1時間後に自動復帰

---

## 📊 数字の目安

| 指標 | 正常 | 警告 | 危険 |
|------|------|------|------|
| pending | 0-50 | 50-200 | 200+ |
| 成功/日 | 120件前後 | 60-120 | <60 |
| 404率 | <10% | 10-30% | 30%+ |
| WAF検知/週 | 0回 | 1回 | 2回+ |

---

## 🔒 絶対ルール

### ❌ やってはいけない

1. DetailLimiterを緩める（5件/時→10件/時等）
2. Workerを迂回して直接scrape
3. permanent_failをretryに戻す

### ✅ やるべき

1. 毎日 `./daily_check.sh` を実行
2. pending が 200+ になったら原因調査
3. WAF検知が週2回以上なら速度を下げる（worker停止時間を増やす）

---

## 📁 ファイル一覧

| ファイル | 役割 |
|---------|------|
| `IMPLEMENTATION_COMPLETE.md` | 実装の詳細・完成度チェック |
| `OPERATIONS_MANUAL.md` | 運用マニュアル・詳細な対処手順 |
| `QUICK_REFERENCE.md` | このファイル（クイックリファレンス） |
| `daily_check.sh` | 毎日実行する監視スクリプト |

---

## 🆘 困った時

### Workerが止まった
```bash
docker-compose restart backend
docker-compose logs -f backend | grep "Queue worker started"
```

### pending が減らない
```bash
# WAF確認
docker-compose logs --since 1h backend | grep WAF

# WAFなし → Workerログ確認
docker-compose logs --tail 100 backend | grep QueueWorker
```

### 404が多すぎる
```bash
# URLサンプル取得
docker-compose exec backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT detail_url FROM detail_scrape_queue WHERE status=\"permanent_fail\" LIMIT 5;"
'

# ブラウザで開く → 表示されたらバグ、404なら正常
```

---

## 🔗 API エンドポイント

### キュー状態確認
```bash
curl http://localhost:8084/api/queue/stats
```

### リスト取得（キューに投入）
```bash
curl -X POST http://localhost:8084/api/scrape/list \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://realestate.yahoo.co.jp/rent/search/...",
    "limit": 20
  }'
```

### 単発詳細取得（テスト用のみ）
```bash
curl -X POST http://localhost:8084/api/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://realestate.yahoo.co.jp/rent/detail/..."
  }'
```

---

**迷ったら**: `OPERATIONS_MANUAL.md` を見る
**日常監視**: `./daily_check.sh` を実行
