-- ============================================
-- Yahoo不動産テキスト クリーンアップスクリプト
-- 本番環境用（手動実行）
-- ============================================
--
-- 使用方法:
-- 1. 本番サーバーにSSHで接続
-- 2. このファイルをアップロード
-- 3. バックアップを作成
-- 4. このSQLを実行
--
-- バックアップコマンド:
-- mysqldump -u shiboroom_user -p shiboroom_db > /var/www/shiboroom.com/backups/db_before_cleanup_$(date +%Y%m%d_%H%M%S).sql
--
-- 実行コマンド:
-- mysql -u shiboroom_user -p shiboroom_db < cleanup_production.sql
--
-- ============================================

-- データベースを選択
USE shiboroom_db;

-- 実行前の確認: 影響を受けるレコード数を表示
SELECT '========================================' as '';
SELECT 'クリーンアップ前の確認' as '';
SELECT '========================================' as '';

SELECT COUNT(*) as '対象レコード数'
FROM properties
WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%'
   OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%' OR title LIKE '%【%';

SELECT '' as '';
SELECT 'クリーンアップ前のサンプル（5件）:' as '';
SELECT id, SUBSTRING(title, 1, 100) as title
FROM properties
WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%'
   OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%' OR title LIKE '%【%'
LIMIT 5;

SELECT '' as '';
SELECT '========================================' as '';
SELECT 'クリーンアップ開始...' as '';
SELECT '========================================' as '';

-- Step 1: Remove brackets with Yahoo不動産 (e.g., "【Yahoo!不動産】" -> "")
UPDATE properties SET title = REPLACE(title, '【Yahoo不動産】', '') WHERE title LIKE '%【Yahoo不動産】%';
UPDATE properties SET title = REPLACE(title, '【Yahoo!不動産】', '') WHERE title LIKE '%【Yahoo!不動産】%';
UPDATE properties SET title = REPLACE(title, '【yahoo不動産】', '') WHERE title LIKE '%【yahoo不動産】%';
UPDATE properties SET title = REPLACE(title, '【YAHOO不動産】', '') WHERE title LIKE '%【YAHOO不動産】%';

-- Step 2: Remove "Yahoo不動産" and variations (without brackets)
UPDATE properties SET title = REPLACE(title, 'Yahoo不動産', '') WHERE title LIKE '%Yahoo不動産%';
UPDATE properties SET title = REPLACE(title, 'Yahoo!不動産', '') WHERE title LIKE '%Yahoo!不動産%';
UPDATE properties SET title = REPLACE(title, 'yahoo不動産', '') WHERE title LIKE '%yahoo不動産%';
UPDATE properties SET title = REPLACE(title, 'YAHOO不動産', '') WHERE title LIKE '%YAHOO不動産%';

-- Step 3: Remove text after separators (e.g., "物件名 - Yahoo不動産" -> "物件名")
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, ' - ', 1)) WHERE title LIKE '% - %';
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, ' | ', 1)) WHERE title LIKE '% | %';
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, '｜', 1)) WHERE title LIKE '%｜%';
UPDATE properties SET title = TRIM(SUBSTRING_INDEX(title, ' 【', 1)) WHERE title LIKE '% 【%';

-- Step 4: Remove empty brackets and extra whitespace
UPDATE properties SET title = REPLACE(title, '【】', '') WHERE title LIKE '%【】%';
UPDATE properties SET title = REPLACE(title, '【 】', '') WHERE title LIKE '%【 】%';
UPDATE properties SET title = REPLACE(title, '  ', ' ') WHERE title LIKE '%  %';

-- Step 5: Trim whitespace and special characters
UPDATE properties SET title = TRIM(title);
UPDATE properties SET title = TRIM(BOTH '-' FROM title);
UPDATE properties SET title = TRIM(BOTH '|' FROM title);
UPDATE properties SET title = TRIM(BOTH '｜' FROM title);
UPDATE properties SET title = TRIM(BOTH '【' FROM title);
UPDATE properties SET title = TRIM(BOTH '】' FROM title);
UPDATE properties SET title = TRIM(title);

-- Step 6: Final cleanup - remove any remaining double spaces
UPDATE properties SET title = REPLACE(title, '  ', ' ') WHERE title LIKE '%  %';
UPDATE properties SET title = TRIM(title);

-- 実行後の確認: クリーンアップ結果を表示
SELECT '' as '';
SELECT '========================================' as '';
SELECT 'クリーンアップ完了！' as '';
SELECT '========================================' as '';

SELECT '' as '';
SELECT 'クリーンアップ後のサンプル（最新10件）:' as '';
SELECT id, SUBSTRING(title, 1, 100) as title
FROM properties
ORDER BY updated_at DESC
LIMIT 10;

SELECT '' as '';
SELECT '残存確認: Yahoo関連のテキストが残っていないか確認' as '';
SELECT COUNT(*) as 'Yahoo関連が残っているレコード数'
FROM properties
WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%';

SELECT '' as '';
SELECT '========================================' as '';
SELECT 'すべて完了しました！' as '';
SELECT '========================================' as '';
