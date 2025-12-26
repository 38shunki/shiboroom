# スクレイピングシステム - 運用ガイド

**最終更新**: 2025-12-22
**ステータス**: ✅ 運用可能

---

## 📚 ドキュメント一覧

| ファイル | 用途 | 読むタイミング |
|---------|------|--------------|
| **README_SCRAPING.md** | **このファイル（最初に読む）** | 最初 |
| `IMPLEMENTATION_COMPLETE.md` | 実装の全体像・技術詳細 | 実装理解時 |
| `OPERATIONS_MANUAL.md` | 運用マニュアル（詳細版） | トラブル時 |
| `OPERATIONS_MANUAL_ADDENDUM.md` | 重要な補足・現実的基準 | トラブル時 |
| `QUICK_REFERENCE.md` | よく使うコマンド早見表 | 日常運用 |
| `FINAL_IMPROVEMENTS.md` | 最新改善内容・WAF対策強化 | 改善理解時 |
| `DEPLOYMENT_SUCCESS.md` | デプロイ記録 | デプロイ時 |
| `FINAL_STATUS.md` | 最終ステータス報告 | 状況確認時 |

---

## 🚀 クイックスタート（初めての方）

### 1分で理解：システムの仕組み

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
    ├─ WAF → cooldown 4h + pause 5m
    └─ 他 → retry (5m→15m→1h→4h→12h)
```

**重要**: 詳細取得は **Worker のみ**。他の経路は全て投入だけ。

---

## ⚡ 毎日やること（10秒）

```bash
cd /Users/shu/Documents/dev/real-estate-portal
./daily_check.sh
```

**Quick Status を見る**:
```
📊 Quick Status: Worker=true | Pending=2 | Done=0 | PermanentFail=5
```

**判定**:
- `Worker=true` → OK
- `Pending` が減ってる or 横ばい → OK
- `Done` が増えてる → OK
- `PermanentFail` が <10% → OK

---

## 🎯 よく使うコマンド

### 状態確認
```bash
# キュー状態（API）
curl http://localhost:8084/api/queue/stats | jq .

# Worker稼働確認
docker-compose logs backend | grep "Queue worker started"

# 最近の成功件数
docker-compose logs backend | grep "✅ Completed" | wc -l
```

### 新URLの投入
```bash
curl -X POST 'http://localhost:8084/api/scrape/list' \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://realestate.yahoo.co.jp/rent/search/?nc=1&pf=13&ct=23","limit":20}'
```

### トラブル時
```bash
# 診断スクリプト
./scraping_diagnosis.sh

# Worker再起動
docker-compose restart backend
```

---

## 🔒 絶対に守るルール

### ❌ やってはいけない

1. **DetailLimiter を緩める**（5件/時→10件/時等）
   - WAFリスクが跳ね上がる

2. **Worker を迂回して直接scrape**
   - レート制限が効かなくなる

3. **permanent_fail を retry に戻す**
   - 無駄なリトライでリソース消耗

4. **WAF検知時に焦って再実行**
   - 自動cooldownで回復するまで待つ

### ✅ やるべき

1. **毎日 `./daily_check.sh` を実行**
   - 所要時間: 10秒

2. **pending が 200+ になったら調査**
   - Worker停止 or WAF cooldown

3. **WAF検知は「待つ」が正解**
   - 4h→4h→12h で自動復帰

4. **404が多い場合はURL生成を調査**
   - 30%+ なら scraper.go 確認

---

## 📊 正常な状態の目安

### 24時間後
```
物件数: 100-120件
Done: 100-120件（5件/時 × 24h）
PermanentFail率: <10%
WAF検知: 0回
Pending: 0-50件
```

### 1週間後
```
物件数: 700-1000件
Done: 700-840件
PermanentFail率: <10%
WAF検知: 0回
Pending: 安定（増え続けない）
```

---

## ⚠️ トラブルシューティング

### Workerが止まった
```bash
docker-compose restart backend
docker-compose logs -f backend | grep "Queue worker started"
```

### pending が減らない
```bash
# WAF確認
docker-compose logs --since 1h backend | grep WAF

# DetailLimiter確認
docker-compose logs --tail 20 backend | grep DetailLimiter
```

### 404が多すぎる（30%+）
```bash
# URLサンプル取得
docker-compose exec backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT detail_url FROM detail_scrape_queue WHERE status=\"permanent_fail\" LIMIT 5;"
'

# ブラウザで開く → 表示されたらバグ、404なら正常
```

---

## 🎓 重要な設計原則

### 1. DetailLimiter（5件/時）は神聖不可侵
- WAFリスクを最小化する唯一の防壁
- 増やすと即WAF発動
- **絶対に変更しない**

### 2. Queue Workerが唯一の実行者
- Scheduler は投入のみ
- `/api/scrape/list` も投入のみ
- 全ての詳細取得は Worker 経由

### 3. 404は「失敗」ではない
- Yahoo側で物件削除済み = 正常
- permanent_fail で即終了 = 正解
- リトライは無駄

### 4. WAF対策の3本柱
- **ヘルスチェック**（起動前）
- **DetailLimiter**（5件/時）
- **予防停止**（3成功→5min pause）

---

## 🚀 次のマイルストーン

### 今日（デプロイ完了）
- [x] Worker起動
- [x] DetailLimiter動作
- [x] 404→permanent_fail動作
- [x] WAFヘルスチェック動作

### 明日（24時間後）
- [ ] `./daily_check.sh` で健康状態確認
- [ ] Done が 100-120件に到達
- [ ] WAF検知が 0回
- [ ] Pending が安定

### 来週（7日後）
- [ ] 物件数が 700-1000件
- [ ] PermanentFail率が <10%
- [ ] 運用が安定（daily_check で異常なし）

---

## 💡 よくある質問

### Q: 物件数が増えないのですが？

**A**: 以下を確認してください：

1. Worker が動いているか
   ```bash
   curl http://localhost:8084/api/queue/stats | jq .is_running
   ```

2. Pending が処理されているか
   ```bash
   # Pending が減っているか確認
   curl http://localhost:8084/api/queue/stats | jq .
   ```

3. DetailLimiter で待機中か
   ```bash
   docker-compose logs backend | grep "DetailLimiter.*wait_sec"
   ```

DetailLimiter で wait_sec=2000+ なら、約30-50分待機中（正常動作）。

---

### Q: WAF検知が出たらどうすれば？

**A**: **何もしない**

- 自動で 4h→4h→12h の cooldown に入る
- Worker が自動で 5min pause
- 焦って再実行しない
- 最大24時間で自動復帰

---

### Q: 404が多いのは異常？

**A**: **比率による**

- <10%: 正常（Yahoo側で物件削除済み）
- 10-30%: やや多い（監視継続）
- 30%+: URL生成ミスの可能性（調査必要）

---

### Q: Pending が増え続けるのですが？

**A**: 以下を確認：

1. Worker が停止していないか
2. WAF cooldown 中でないか
3. DetailLimiter の wait時間が異常に長くないか

対応:
```bash
# 診断スクリプト実行
./scraping_diagnosis.sh

# Worker再起動（最終手段）
docker-compose restart backend
```

---

## 📞 サポート

### 問題発生時の手順

1. **診断スクリプト実行**
   ```bash
   ./scraping_diagnosis.sh
   ```

2. **daily_check実行**
   ```bash
   ./daily_check.sh
   ```

3. **ログ確認**
   ```bash
   docker-compose logs -f backend | grep QueueWorker
   ```

4. **ドキュメント参照**
   - コマンド: `QUICK_REFERENCE.md`
   - 運用: `OPERATIONS_MANUAL.md`
   - 補足: `OPERATIONS_MANUAL_ADDENDUM.md`

---

## 🎉 まとめ

**このシステムは運用可能な状態です。**

### 強み
- ✅ DetailLimiter で WAFリスク最小化
- ✅ 404を即座に判定（無駄なリトライなし）
- ✅ WAFヘルスチェックで事前回避
- ✅ 予防停止で人間らしい挙動
- ✅ 1行サマリーで即状況把握

### 弱み（改善予定）
- ⚠️ 5件/時は遅い（安全性とのトレードオフ）
- ⚠️ 単一Worker（スケールアップ余地あり）
- ⚠️ Yahoo不動産のみ（他サイト未対応）

### 次のステップ
1. **24時間後**: daily_check.sh で健康確認
2. **1週間後**: 運用安定性の確認
3. **1ヶ月後**: 品質改善・機能拡張

---

**運用開始おめでとうございます！** 🎊

何か質問があれば、該当ドキュメントを参照してください。
