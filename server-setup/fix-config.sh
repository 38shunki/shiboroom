#!/usr/bin/env bash
# 設定ファイルを本番環境用に修正
# サーバー上で実行: bash /tmp/fix-config.sh

set -euo pipefail

echo "==== 設定ファイルの修正 ===="
echo ""

CONFIG_FILE="/var/www/shiboroom/config/scraper_config.yaml"

# バックアップを作成
echo "[1/3] 現在の設定ファイルをバックアップ..."
cp "$CONFIG_FILE" "${CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"

# データベースパスワードを取得
echo "[2/3] データベースパスワードを入力してください:"
read -sp "shiboroom_user のパスワード: " DB_PASSWORD
echo ""

# 新しい設定ファイルを作成
echo "[3/3] 本番環境用の設定ファイルを作成..."
cat > "$CONFIG_FILE" <<EOF
# Scraper Configuration (Production)
# This file controls scraping behavior to minimize load and ensure safe operation

# Database configuration
database:
  type: mysql  # Use MySQL for production
  mysql:
    host: "127.0.0.1"
    port: 3306
    user: "shiboroom_user"
    password: "$DB_PASSWORD"
    database: "shiboroom"
  postgres:
    host: "127.0.0.1"
    port: 5432
    user: "postgres"
    password: "postgres"
    database: "realestate"
    sslmode: "disable"

# Search engine configuration
search:
  meilisearch:
    host: "http://127.0.0.1:7700"
    api_key: "masterKey123"

# Scraper settings
scraper:
  # Request timing
  request_delay_seconds: 2  # Minimum delay between requests (rate limiting)
  timeout_seconds: 30       # HTTP timeout for each request

  # Retry policy
  max_retries: 3            # Maximum number of retry attempts
  retry_delay_seconds: 2    # Base delay for exponential backoff

  # Safety limits
  max_requests_per_day: 5000    # Daily request limit
  stop_on_error: false          # Continue on errors (production)
  concurrent_limit: 1           # Number of concurrent scrapers (1 = sequential only)

  # Scraping schedule (日本時間 - JST)
  daily_run_enabled: false      # Enable/disable daily scheduled runs
  daily_run_time: "02:00"      # Time for daily run (HH:MM format, JST/Japan Time)

  # List page scraping
  list_page_limit: 50          # Max properties to scrape from list page

# Rate limiting
rate_limit:
  enabled: true
  requests_per_minute: 30     # Maximum requests per minute
  requests_per_hour: 1800     # Maximum requests per hour (30 req/min * 60 min)

# Error handling
error_handling:
  retry_on_network_error: true
  retry_on_5xx: true
  retry_on_4xx: false         # Don't retry on client errors (400-499)
  log_errors: true

# User Agent
user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

# Logging
logging:
  level: "info"              # debug, info, warn, error
  log_requests: true         # Log all requests
  log_responses: false       # Log response bodies (can be large)

# Timezone
timezone: "Asia/Tokyo"       # Japan Standard Time (JST)
EOF

echo ""
echo "✅ 設定ファイルの修正完了"
echo ""
echo "バックアップ: ${CONFIG_FILE}.backup.*"
echo ""
echo "次のステップ:"
echo "  1. Meilisearchが起動しているか確認: curl http://localhost:7700/health"
echo "  2. バックエンドを再起動: sudo systemctl restart shiboroom-backend"
echo "  3. ステータス確認: sudo systemctl status shiboroom-backend"
echo ""
