# Deployment Troubleshooting Guide

## 概要
このドキュメントは、shiboroom.comのバックエンドデプロイ時に発生した問題と解決方法をまとめたものです。

---

## 問題1: GORM マイグレーションエラー (datetime NOT NULL)

### エラーメッセージ
```
Error 1292 (22007): Incorrect datetime value: '0000-00-00 00:00:00' for column 'last_seen_at' at row 1
[rows:0] ALTER TABLE `properties` ADD `last_seen_at` datetime NOT NULL
Failed to initialize schema
```

### 原因
- GORMが既存テーブルに新しいカラム `last_seen_at` を追加しようとした
- `NOT NULL` 制約付きで追加しようとしたが、既存の99件のレコードにデフォルト値が設定できず失敗

### 解決方法
モデル定義で `*time.Time` (ポインタ型) を使用してNULL許容にする：

**変更前** (`internal/models/property.go`):
```go
LastSeenAt time.Time `gorm:"type:datetime;not null;index" json:"last_seen_at"`
```

**変更後**:
```go
LastSeenAt *time.Time `gorm:"type:datetime;index" json:"last_seen_at,omitempty"`
```

関連メソッドも更新：
```go
func (p *Property) DaysSinceLastSeen() int {
    if p.LastSeenAt == nil {
        return 9999 // Never seen
    }
    return int(time.Since(*p.LastSeenAt).Hours() / 24)
}

func (p *Property) UpdateLastSeen() {
    now := time.Now()
    p.LastSeenAt = &now
}
```

### 重要ポイント
- ✅ 既存データは保持される（削除されない）
- ✅ 新しいレコードにはNULL値が許容される
- ✅ 後から値を設定可能

---

## 問題2: UNIQUE INDEX作成エラー (重複データ)

### エラーメッセージ
```
Error 1062 (23000): Duplicate entry 'yahoo-' for key 'properties.idx_source_property'
CREATE UNIQUE INDEX `idx_source_property` ON `properties`(`source`,`source_property_id`)
```

### 原因
- 99件の既存物件すべてが `source_property_id = ''` (空文字列)
- UNIQUE INDEX `(source, source_property_id)` を作成しようとしたが、重複のため失敗

### 診断コマンド
```sql
-- 重複を確認
SELECT source, source_property_id, COUNT(*) as count
FROM properties
GROUP BY source, source_property_id
HAVING COUNT(*) > 1;
```

### 解決方法
MySQLで直接、空の`source_property_id`に一意のIDを割り当てる：

```bash
# サーバーでMySQLに接続
mysql -u shiboroom_user -p shiboroom

# または直接実行
mysql -u shiboroom_user -p'Kihara0725$' shiboroom -e \
  "UPDATE properties
   SET source_property_id = CONCAT('fallback-', id)
   WHERE source_property_id = '' OR source_property_id IS NULL;"
```

### 結果
- 空だった `source_property_id` が `fallback-<md5_hash>` に変換される
- 例: `source_property_id = 'fallback-1a2b3c4d5e6f...'`
- UNIQUE INDEXが正常に作成される

### 重要ポイント
- ✅ 既存データは保持される（IDだけが変更される）
- ✅ 重複が解消され、UNIQUE制約が適用可能になる
- ✅ 将来的に正しいIDでスクレイプし直せば上書きされる

---

## 問題3: ポート衝突 (Address Already in Use)

### エラーメッセージ
```
[ERROR] listen tcp :8085: bind: address already in use
Failed to start server
```

### 原因
- 古いバックエンドプロセスが手動で起動されており、ポート8085を使用中
- systemdで新しいプロセスを起動しようとしたが、ポートが取得できず失敗

### 診断コマンド
```bash
# ポート8085を使用しているプロセスを確認
sudo lsof -i :8085

# 出力例:
# COMMAND      PID USER   FD   TYPE  DEVICE SIZE/OFF NODE NAME
# shiboroom 237666 grik    6u  IPv6 1798196      0t0  TCP *:8085 (LISTEN)
```

### 解決方法
古いプロセスを停止してから、systemd経由で起動：

```bash
# 1. 古いプロセスを停止
sudo kill 237666

# 2. systemd経由で起動
sudo systemctl start shiboroom-backend

# 3. 状態確認
systemctl status shiboroom-backend
```

### 重要ポイント
- ✅ プロセスを停止してもデータベースには影響なし
- ✅ MySQLデータベースは別プロセスで動作
- ✅ 既存の99件の物件データは保持される
- ⚠️ systemd管理外の手動起動は避ける

---

## 問題4: Dockerビルドとアーキテクチャ互換性

### エラーメッセージ (初期)
```
Exec format error
```

### 原因
- Apple Silicon (ARM64) でビルドされたバイナリを x86_64 サーバーで実行しようとした
- さらに、Alpine Linux (musl) でビルドされたバイナリを Ubuntu (glibc) で実行しようとした

### 解決方法
静的リンクされたx86_64バイナリをビルド：

```bash
cd /Users/shu/Documents/dev/real-estate-portal/backend

# Dockerで静的リンクビルド
docker run --rm --platform linux/amd64 \
  -v "$(pwd)":/app -w /app \
  golang:1.23-alpine sh -c \
  "apk add --no-cache gcc musl-dev >/dev/null 2>&1 && \
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
   go build -ldflags '-extldflags \"-static\"' \
   -o shiboroom-api ./cmd/api"

# バイナリ種別を確認
file shiboroom-api
# 期待される出力: "statically linked"
```

### 検証
```bash
# サーバーで実行可能か確認
scp shiboroom-api grik@162.43.74.38:/tmp/test-api
ssh grik@162.43.74.38 '/tmp/test-api --version'
```

### 重要ポイント
- ✅ `--platform linux/amd64`: x86_64アーキテクチャを指定
- ✅ `CGO_ENABLED=0`: 静的リンクを有効化
- ✅ `-ldflags '-extldflags "-static"'`: 完全静的リンク
- ✅ どのLinuxディストリビューションでも動作

---

## デプロイ手順まとめ

### 1. ローカルでビルド
```bash
cd backend
docker run --rm --platform linux/amd64 -v "$(pwd)":/app -w /app golang:1.23-alpine sh -c \
  "apk add --no-cache gcc musl-dev >/dev/null 2>&1 && \
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
   go build -ldflags '-extldflags \"-static\"' -o shiboroom-api ./cmd/api"
```

### 2. サーバーにアップロード
```bash
scp shiboroom-api grik@162.43.74.38:/tmp/shiboroom-api-new
```

### 3. サーバーで配置と再起動
```bash
# 古いプロセスがあれば停止
sudo lsof -i :8085
# PIDがあれば: sudo kill <PID>

# バイナリを配置
sudo mv /tmp/shiboroom-api-new /var/www/shiboroom/backend/shiboroom-api
sudo chown grik:grik /var/www/shiboroom/backend/shiboroom-api
sudo chmod +x /var/www/shiboroom/backend/shiboroom-api

# サービス再起動
sudo systemctl restart shiboroom-backend
systemctl status shiboroom-backend
```

### 4. 動作確認
```bash
# ログ確認
sudo journalctl -u shiboroom-backend -n 50 --no-pager

# APIテスト
curl http://localhost:8085/health
```

---

## トラブルシューティングチェックリスト

### サービスが起動しない場合

1. **ログを確認**
   ```bash
   sudo journalctl -u shiboroom-backend -n 100 --no-pager
   ```

2. **よくあるエラーと対処**
   - `Address already in use` → 古いプロセスを `sudo lsof -i :8085` で見つけてkill
   - `Exec format error` → バイナリを静的リンクで再ビルド
   - `Error 1292` (datetime) → モデルを `*time.Time` に変更して再ビルド
   - `Error 1062` (duplicate) → MySQLで重複データを修正

3. **データベース接続確認**
   ```bash
   mysql -u shiboroom_user -p'Kihara0725$' shiboroom -e "SELECT COUNT(*) FROM properties;"
   ```

4. **設定ファイル確認**
   ```bash
   cat /var/www/shiboroom/config/scraper_config.yaml
   cat /etc/systemd/system/shiboroom-backend.service
   ```

---

## 重要な注意事項

### データ保護
- ✅ プロセスの停止/再起動はデータを削除しない
- ✅ バイナリの置き換えはデータを削除しない
- ✅ マイグレーションは既存データを保持する
- ⚠️ `DROP TABLE` や `TRUNCATE` は絶対に使用しない
- ⚠️ Dockerコンテナの削除はデータに影響しない（MySQLは別プロセス）

### ログの重要性
- 問題発生時は必ず `journalctl` でログを確認
- エラーメッセージの全文をコピーして検索可能にする
- タイムスタンプを確認して新旧のログを区別

### systemd管理
- サービスは必ず systemd 経由で管理する
- 手動起動 (`./shiboroom-api`) は避ける
- サービスの自動起動を有効化: `sudo systemctl enable shiboroom-backend`

---

## 連絡先とリソース

- プロジェクトリポジトリ: `/Users/shu/Documents/dev/real-estate-portal`
- サーバー: `grik@162.43.74.38`
- バックエンドパス: `/var/www/shiboroom/backend/`
- 設定ファイル: `/var/www/shiboroom/config/scraper_config.yaml`
- systemdサービス: `/etc/systemd/system/shiboroom-backend.service`

---

## 更新履歴

- **2025-12-18**: 初版作成 - last_seen_at, source_property_id, port conflict の解決方法を文書化
