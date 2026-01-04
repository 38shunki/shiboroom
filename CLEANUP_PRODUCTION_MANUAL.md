# 本番環境データクリーンアップ 手動実行手順

## 概要

本番環境のデータベースから「Yahoo不動産」テキストを安全に削除する手順です。

## ⚠️ 重要な注意事項

- **データは削除されません**: タイトルの文字列を編集するのみです
- **自動バックアップ**: 必ず実行前にバックアップを取ります
- **復元可能**: 問題があればすぐに元に戻せます

## 📋 事前準備

### 必要なファイル

1. `cleanup_production.sql` - クリーンアップ用SQLスクリプト
2. このマニュアル

## 🚀 実行手順

### ステップ1: 本番サーバーにSSH接続

```bash
ssh grik@162.43.74.38
```

### ステップ2: 作業ディレクトリに移動

```bash
cd /var/www/shiboroom.com
```

### ステップ3: バックアップディレクトリを作成（初回のみ）

```bash
mkdir -p backups
```

### ステップ4: データベースのバックアップを作成 ⭐ 重要

```bash
# バックアップを作成（これが最も重要なステップです）
mysqldump -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db > backups/db_before_cleanup_$(date +%Y%m%d_%H%M%S).sql

# バックアップが正常に作成されたか確認
ls -lh backups/db_before_cleanup_*.sql | tail -1
```

**期待される出力例:**
```
-rw-r--r-- 1 grik grik 25M Dec 28 09:00 backups/db_before_cleanup_20251228_090000.sql
```

ファイルサイズが0でないことを確認してください。

### ステップ5: SQLスクリプトをサーバーにアップロード

**ローカルマシンで実行（別のターミナルを開く）:**

```bash
cd /Users/shu/Documents/dev/real-estate-portal
scp cleanup_production.sql grik@162.43.74.38:/var/www/shiboroom.com/
```

### ステップ6: 実行前の確認（データを見る）

```bash
# Yahoo関連のタイトルを確認
mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db -e "SELECT COUNT(*) as count FROM properties WHERE title LIKE '%Yahoo%';"

# サンプルを見る
mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db -e "SELECT SUBSTRING(title, 1, 100) as title FROM properties WHERE title LIKE '%Yahoo%' LIMIT 5;"
```

### ステップ7: クリーンアップ実行 ⭐

```bash
# SQLスクリプトを実行
mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db < cleanup_production.sql
```

**実行中の出力:**
- クリーンアップ前のレコード数
- サンプルデータ（5件）
- クリーンアップ後のサンプル（10件）
- Yahoo関連テキストの残存確認

### ステップ8: 実行後の確認

```bash
# Yahoo関連のタイトルが残っていないか確認
mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db -e "SELECT COUNT(*) as remaining FROM properties WHERE title LIKE '%Yahoo%';"
```

**期待される結果:** `remaining: 0`

```bash
# クリーンアップ後のタイトルを確認
mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db -e "SELECT SUBSTRING(title, 1, 100) as title FROM properties ORDER BY updated_at DESC LIMIT 10;"
```

### ステップ9: Webサイトで確認

ブラウザで本番サイトにアクセスして物件詳細ページを確認:

```
https://shiboroom.com
```

- 物件一覧でタイトルが正しく表示されているか
- 「Yahoo不動産」というテキストが表示されていないか

## 🔄 バックアップからの復元方法（問題があった場合）

### 最新のバックアップを確認

```bash
cd /var/www/shiboroom.com/backups
ls -lt db_before_cleanup_*.sql | head -1
```

### 復元実行

```bash
# バックアップファイル名を確認して実行
mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db < backups/db_before_cleanup_YYYYMMDD_HHMMSS.sql
```

**注意:** `YYYYMMDD_HHMMSS` は実際のファイル名に置き換えてください。

## 📊 クリーンアップ例

| 元のタイトル | クリーンアップ後 |
|-------------|----------------|
| テストマンション【Yahoo!不動産】 | テストマンション |
| 【Yahoo!不動産】サンプル物件 | サンプル物件 |
| アパート名 - Yahoo不動産 | アパート名 |
| 高級マンション｜Yahoo!不動産 | 高級マンション |
| 普通の物件名 | 普通の物件名（変更なし）|

## ✅ 実行チェックリスト

実行前に以下を確認してください:

- [ ] 本番サーバーにSSH接続できる
- [ ] データベースのバックアップを作成した
- [ ] バックアップファイルのサイズが0でないことを確認した
- [ ] `cleanup_production.sql` をサーバーにアップロードした
- [ ] 実行前のデータサンプルを確認した

実行後に以下を確認してください:

- [ ] スクリプトがエラーなく完了した
- [ ] Yahoo関連テキストが残っていない（COUNT = 0）
- [ ] クリーンアップ後のタイトルが適切に表示されている
- [ ] Webサイトで物件詳細が正しく表示されている

## 🆘 トラブルシューティング

### MySQLに接続できない

```bash
# MySQL サービスの状態を確認
sudo systemctl status mysql

# 必要に応じて再起動
sudo systemctl restart mysql
```

### ファイルのアップロードができない

```bash
# SCPではなく、サーバー上で直接ファイルを作成
ssh grik@162.43.74.38
cd /var/www/shiboroom.com
nano cleanup_production.sql
# cleanup_production.sql の内容をコピー＆ペースト
# Ctrl+X -> Y -> Enter で保存
```

### バックアップファイルが見つからない

```bash
# バックアップディレクトリを確認
ls -la /var/www/shiboroom.com/backups/

# 別の場所を確認
find /var/www -name "db_before_cleanup_*.sql" 2>/dev/null
```

## 📞 サポート

問題が発生した場合:

1. **エラーメッセージを保存**
2. **バックアップファイルを確認** - 削除しないでください
3. **復元が必要な場合** - 上記の復元手順を実行
4. **ログを確認** - `/var/log/mysql/error.log`

## 📝 実行記録

実行後、以下の情報を記録してください:

- 実行日時: ________________
- バックアップファイル名: ________________
- 対象レコード数: ________________
- 実行者: ________________
- 結果: [ ] 成功 [ ] 失敗（理由: ________________）

---

**作成日:** 2025-12-28
**バージョン:** 1.0
**関連ファイル:** cleanup_production.sql, cleanup_yahoo_text.sh
