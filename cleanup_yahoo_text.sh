#!/bin/bash

# Script to clean "Yahoo不動産" text from property titles in database
# This script is safe and creates backups before making changes

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVER="grik@162.43.74.38"
REMOTE_PROJECT_DIR="/var/www/shiboroom.com"
BACKUP_DIR="$REMOTE_PROJECT_DIR/backups"
TIMESTAMP=$(TZ=Asia/Tokyo date +%Y%m%d_%H%M%S)

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}Yahoo不動産テキスト クリーンアップ${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# Function to show usage
usage() {
  echo "使用方法:"
  echo "  $0 [local|production]"
  echo ""
  echo "引数:"
  echo "  local       - ローカルのDockerデータベースをクリーンアップ"
  echo "  production  - 本番環境のデータベースをクリーンアップ"
  echo ""
  echo "例:"
  echo "  $0 local       # ローカル環境で実行"
  echo "  $0 production  # 本番環境で実行"
  exit 1
}

# Check arguments
if [ $# -eq 0 ]; then
  usage
fi

ENVIRONMENT=$1

if [ "$ENVIRONMENT" != "local" ] && [ "$ENVIRONMENT" != "production" ]; then
  echo -e "${RED}エラー: 無効な環境指定です${NC}"
  usage
fi

echo -e "${YELLOW}環境: $ENVIRONMENT${NC}"
echo ""

# Confirmation
echo -e "${YELLOW}このスクリプトは以下を実行します:${NC}"
echo "1. データベースのバックアップを作成"
echo "2. properties テーブルの title カラムから 'Yahoo不動産' を削除"
echo "3. セパレーター（- | など）以降のテキストを削除"
echo ""
echo -e "${RED}注意: 本番環境のデータを変更します。バックアップは自動で作成されます。${NC}"
echo ""
read -p "続行しますか？ (yes/no): " -r
echo
if [[ ! $REPLY =~ ^[Yy]es$ ]]; then
  echo -e "${YELLOW}キャンセルしました${NC}"
  exit 0
fi

# Function to clean titles in local database
cleanup_local() {
  echo -e "${BLUE}[ローカル] データベースクリーンアップ開始${NC}"
  echo ""

  # 1. Create backup
  echo -e "${YELLOW}[1/4] バックアップ作成中...${NC}"
  BACKUP_FILE="backup_before_cleanup_${TIMESTAMP}.sql"
  docker-compose exec -T mysql mysqldump -u realestate_user -prealestate_pass realestate_db > "$BACKUP_FILE" 2>/dev/null
  echo -e "${GREEN}✓ バックアップ完了: $BACKUP_FILE${NC}"
  echo ""

  # 2. Count affected records
  echo -e "${YELLOW}[2/4] 対象レコード数を確認中...${NC}"
  AFFECTED_COUNT=$(docker-compose exec mysql mysql -u realestate_user -prealestate_pass realestate_db -se "SELECT COUNT(*) FROM properties WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%' OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%';" 2>/dev/null)
  echo -e "${BLUE}対象レコード数: $AFFECTED_COUNT 件${NC}"
  echo ""

  # 3. Show sample before cleanup
  echo -e "${YELLOW}[3/4] クリーンアップ前のサンプル（最大5件）:${NC}"
  docker-compose exec mysql mysql -u realestate_user -prealestate_pass realestate_db -e "SELECT id, SUBSTRING(title, 1, 80) as title FROM properties WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%' OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%' LIMIT 5;" 2>/dev/null || echo "該当データなし"
  echo ""

  # 4. Execute cleanup
  echo -e "${YELLOW}[4/4] クリーンアップ実行中...${NC}"

  # Create SQL script for cleanup
  cat > /tmp/cleanup_titles.sql <<'EOF'
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
EOF

  # Execute cleanup SQL
  docker-compose exec -T mysql mysql -u realestate_user -prealestate_pass realestate_db < /tmp/cleanup_titles.sql 2>/dev/null

  echo -e "${GREEN}✓ クリーンアップ完了${NC}"
  echo ""

  # 5. Show sample after cleanup
  echo -e "${YELLOW}クリーンアップ後のサンプル（最大5件）:${NC}"
  docker-compose exec mysql mysql -u realestate_user -prealestate_pass realestate_db -e "SELECT id, SUBSTRING(title, 1, 80) as title FROM properties ORDER BY updated_at DESC LIMIT 5;" 2>/dev/null
  echo ""

  # Cleanup temp file
  rm -f /tmp/cleanup_titles.sql

  echo -e "${GREEN}======================================${NC}"
  echo -e "${GREEN}ローカルクリーンアップ完了！${NC}"
  echo -e "${GREEN}======================================${NC}"
  echo -e "${BLUE}バックアップファイル: $BACKUP_FILE${NC}"
  echo -e "${BLUE}対象レコード数: $AFFECTED_COUNT 件${NC}"
  echo ""
  echo -e "${YELLOW}問題があった場合、以下のコマンドで復元できます:${NC}"
  echo -e "${YELLOW}docker-compose exec -T mysql mysql -u realestate_user -prealestate_pass realestate_db < $BACKUP_FILE${NC}"
}

# Function to clean titles in production database
cleanup_production() {
  echo -e "${BLUE}[本番環境] データベースクリーンアップ開始${NC}"
  echo ""

  # Create cleanup SQL script
  cat > /tmp/cleanup_titles.sql <<'EOF'
-- Backup check: Show affected records count
SELECT COUNT(*) as affected_count FROM properties
WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%'
   OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%' OR title LIKE '%【%';

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

-- Show sample after cleanup
SELECT id, SUBSTRING(title, 1, 80) as title FROM properties ORDER BY updated_at DESC LIMIT 5;
EOF

  # Upload script to server
  echo -e "${YELLOW}[1/5] クリーンアップスクリプトをサーバーにアップロード中...${NC}"
  scp /tmp/cleanup_titles.sql ${SERVER}:/tmp/cleanup_titles.sql
  echo -e "${GREEN}✓ アップロード完了${NC}"
  echo ""

  # Execute on production server
  echo -e "${YELLOW}[2/5] 本番サーバーでバックアップ作成中...${NC}"
  ssh ${SERVER} << 'ENDSSH'
set -e
TIMESTAMP=$(TZ=Asia/Tokyo date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/www/shiboroom.com/backups"
BACKUP_FILE="$BACKUP_DIR/db_before_cleanup_${TIMESTAMP}.sql"

# Create backup directory if not exists
mkdir -p $BACKUP_DIR

# Create backup
echo "バックアップ作成中: $BACKUP_FILE"
mysqldump -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db > $BACKUP_FILE

# Compress backup
gzip $BACKUP_FILE
echo "✓ バックアップ完了: ${BACKUP_FILE}.gz"

# Keep only last 10 backups
cd $BACKUP_DIR
ls -t db_before_cleanup_*.sql.gz | tail -n +11 | xargs -r rm
echo "✓ 古いバックアップを削除（最新10件を保持）"
ENDSSH

  echo -e "${GREEN}✓ バックアップ完了${NC}"
  echo ""

  # Count affected records
  echo -e "${YELLOW}[3/5] 対象レコード数を確認中...${NC}"
  ssh ${SERVER} "mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db -se \"SELECT COUNT(*) FROM properties WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%' OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%';\""
  echo ""

  # Show sample before cleanup
  echo -e "${YELLOW}[4/5] クリーンアップ前のサンプル:${NC}"
  ssh ${SERVER} "mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db -e \"SELECT id, SUBSTRING(title, 1, 80) as title FROM properties WHERE title LIKE '%Yahoo%' OR title LIKE '%yahoo%' OR title LIKE '% - %' OR title LIKE '% | %' OR title LIKE '%｜%' LIMIT 5;\""
  echo ""

  # Execute cleanup
  echo -e "${YELLOW}[5/5] クリーンアップ実行中...${NC}"
  ssh ${SERVER} "mysql -u shiboroom_user -p'9Ry8tF2nX4hK6vL' shiboroom_db < /tmp/cleanup_titles.sql"
  echo -e "${GREEN}✓ クリーンアップ完了${NC}"
  echo ""

  # Cleanup temp files
  rm -f /tmp/cleanup_titles.sql
  ssh ${SERVER} "rm -f /tmp/cleanup_titles.sql"

  echo -e "${GREEN}======================================${NC}"
  echo -e "${GREEN}本番環境クリーンアップ完了！${NC}"
  echo -e "${GREEN}======================================${NC}"
  echo ""
  echo -e "${BLUE}バックアップは /var/www/shiboroom.com/backups/ に保存されています${NC}"
  echo ""
  echo -e "${YELLOW}問題があった場合、サーバーにSSHで接続して以下のコマンドで復元できます:${NC}"
  echo -e "${YELLOW}cd /var/www/shiboroom.com/backups${NC}"
  echo -e "${YELLOW}gunzip -c db_before_cleanup_${TIMESTAMP}.sql.gz | mysql -u shiboroom_user -p shiboroom_db${NC}"
}

# Execute based on environment
case $ENVIRONMENT in
  local)
    cleanup_local
    ;;
  production)
    cleanup_production
    ;;
esac

echo -e "${GREEN}すべて完了しました！${NC}"
