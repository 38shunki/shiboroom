# Yahoo不動産テキスト クリーンアップ手順

## 概要

このドキュメントは、データベース内の物件タイトルから「Yahoo不動産」というテキストを削除する手順を説明します。

## 背景

物件データのスクレイピング時に、タイトルに「Yahoo不動産」というテキストが含まれていることがあります。これを削除してクリーンなタイトル表示にするため、既存データのクリーンアップとスクレイピングコードの修正を実施しました。

## 対応内容

### 1. コード修正（完了）

#### フロントエンド
- **ファイル**: `frontend/src/app/page.tsx:1714`
- **変更内容**: 「Yahoo不動産で詳細を見る」→「物件詳細を見る」

#### バックエンド
- **ファイル**: `backend/internal/scraper/scraper.go`
- **追加機能**: `cleanTitle()` 関数を実装
  - 「Yahoo不動産」「Yahoo!不動産」などのバリエーションを削除
  - セパレーター（` - `, ` | `, `｜`, ` 【`）以降のテキストを削除
  - 前後の空白や記号を整理

### 2. データベースクリーンアップスクリプト

**スクリプト**: `cleanup_yahoo_text.sh`

## 使用方法

### ローカル環境でのクリーンアップ

```bash
# プロジェクトルートディレクトリで実行
./cleanup_yahoo_text.sh local
```

### 本番環境でのクリーンアップ

```bash
# プロジェクトルートディレクトリで実行
./cleanup_yahoo_text.sh production
```

## スクリプトの実行内容

### 1. バックアップ作成
- **ローカル**: `backup_before_cleanup_YYYYMMDD_HHMMSS.sql`
- **本番**: `/var/www/shiboroom.com/backups/db_before_cleanup_YYYYMMDD_HHMMSS.sql.gz`

### 2. 対象レコード数の確認
以下の条件に該当するレコード数を表示：
- タイトルに「Yahoo」または「yahoo」を含む
- タイトルにセパレーター（` - `, ` | `, `｜`）を含む

### 3. クリーンアップ前のサンプル表示
最大5件のサンプルデータを表示

### 4. クリーンアップ実行
以下の処理を実行：

#### ステップ1: Yahoo不動産テキストの削除
```sql
UPDATE properties SET title = REPLACE(title, 'Yahoo不動産', '');
UPDATE properties SET title = REPLACE(title, 'Yahoo!不動産', '');
UPDATE properties SET title = REPLACE(title, 'yahoo不動産', '');
UPDATE properties SET title = REPLACE(title, 'YAHOO不動産', '');
```

#### ステップ2: セパレーター以降のテキスト削除
```sql
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, ' - ', 1));
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, ' | ', 1));
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, '｜', 1));
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, ' 【', 1));
```

#### ステップ3: 空白と記号のクリーンアップ
```sql
UPDATE properties SET title = TRIM(title);
UPDATE properties SET title = TRIM(BOTH '-' FROM title);
UPDATE properties SET title = TRIM(BOTH '|' FROM title);
UPDATE properties SET title = TRIM(BOTH '｜' FROM title);
UPDATE properties SET title = TRIM(title);
```

### 5. クリーンアップ後のサンプル表示
最新5件のデータを表示

## クリーンアップ例

| 元のタイトル | クリーンアップ後 |
|-------------|----------------|
| テストマンション - Yahoo不動産 | テストマンション |
| サンプル物件 \| Yahoo!不動産 | サンプル物件 |
| アパート名Yahoo不動産 | アパート名 |
| 高級マンション 【Yahoo不動産】 | 高級マンション |
| 普通の物件名 | 普通の物件名 |

## バックアップからの復元方法

### ローカル環境

```bash
# バックアップファイルから復元
docker-compose exec -T mysql mysql -u realestate_user -prealestate_pass realestate_db < backup_before_cleanup_YYYYMMDD_HHMMSS.sql
```

### 本番環境

```bash
# サーバーにSSHで接続
ssh grik@162.43.74.38

# バックアップディレクトリに移動
cd /var/www/shiboroom.com/backups

# バックアップファイルを確認
ls -lh db_before_cleanup_*.sql.gz

# 復元（最新のバックアップを使用）
gunzip -c db_before_cleanup_YYYYMMDD_HHMMSS.sql.gz | mysql -u shiboroom_user -p shiboroom_db
```

## 安全性

### データ保護
1. **自動バックアップ**: 実行前に必ずバックアップを作成
2. **確認プロンプト**: 実行前に確認を求める
3. **サンプル表示**: 実行前後のデータをサンプル表示
4. **バックアップ保持**: 最新10件のバックアップを保持（本番環境）

### エラー対策
- `set -e`: エラー発生時に即座に停止
- バックアップファイルの自動圧縮（本番環境）
- 復元手順の明示

## 注意事項

1. **実行前の確認**
   - 本番環境で実行する前に、必ずローカル環境でテストすること
   - バックアップが正常に作成されることを確認すること

2. **実行タイミング**
   - アクセスの少ない時間帯に実行することを推奨
   - 念のため、メンテナンス時間を設定することも検討

3. **実行後の確認**
   - Webサイトで物件詳細ページを確認
   - タイトルが正しく表示されることを確認

## トラブルシューティング

### スクリプトが実行できない

```bash
# 実行権限を付与
chmod +x cleanup_yahoo_text.sh
```

### MySQLに接続できない（ローカル）

```bash
# Dockerコンテナが起動しているか確認
docker-compose ps

# 必要に応じて起動
docker-compose up -d mysql
```

### SSHで接続できない（本番）

```bash
# SSH接続を確認
ssh grik@162.43.74.38

# 接続できない場合は、サーバー管理者に確認
```

## 今後の対応

### 新規スクレイピングデータ

`backend/internal/scraper/scraper.go` の `cleanTitle()` 関数により、新規スクレイピング時に自動的に「Yahoo不動産」テキストが削除されます。

追加のクリーンアップは不要です。

### 定期メンテナンス

月次メンテナンス時に以下を確認：

```sql
-- Yahoo不動産を含むタイトルがないか確認
SELECT COUNT(*) FROM properties
WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%';

-- 該当データがあればサンプルを確認
SELECT id, title FROM properties
WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%'
LIMIT 10;
```

## 関連ファイル

- **スクリプト**: `cleanup_yahoo_text.sh`
- **バックエンド**: `backend/internal/scraper/scraper.go`
- **フロントエンド**: `frontend/src/app/page.tsx`
- **ドキュメント**: `docs/CLEANUP_YAHOO_TEXT.md`（このファイル）

## 更新履歴

| 日付 | 内容 |
|------|------|
| 2025-12-28 | 初版作成 |
