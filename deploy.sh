#!/usr/bin/env bash
set -euo pipefail

##
## Real Estate Portal デプロイスクリプト
## ドメイン: shiboroom.com
##

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_SRC="$PROJECT_ROOT/backend"
FRONTEND_SRC="$PROJECT_ROOT/frontend-next"
BUILD_DIR="$PROJECT_ROOT/.deploy-build"
BACKEND_BIN_LOCAL="$BUILD_DIR/shiboroom-api-linux-amd64"

# サーバー側
REMOTE_HOST="grik@162.43.74.38"
REMOTE_APP_ROOT="/var/www/shiboroom"
REMOTE_BACKEND_BIN="$REMOTE_APP_ROOT/backend/shiboroom-api"
REMOTE_FRONTEND_DIR="$REMOTE_APP_ROOT/frontend"
REMOTE_CONFIG_DIR="$REMOTE_APP_ROOT/config"

echo "==== [0] ビルド用ディレクトリ作成 ===="
mkdir -p "$BUILD_DIR"

########################################
# [1] Go バックエンド ビルド（ローカル → Linux バイナリ）
########################################
echo "==== [1] Go バックエンド ビルド ===="
cd "$PROJECT_ROOT"

if command -v go >/dev/null 2>&1; then
  echo "→ ローカルの go でビルドします"
  (
    cd "$BACKEND_SRC"
    echo "  - APIサーバーをビルド"
    GOOS=linux GOARCH=amd64 go build -o "$BACKEND_BIN_LOCAL" ./cmd/api
  )
else
  echo "→ go がローカルに無いので Docker(golang) でビルドします"
  docker run --rm \
    -v "$PROJECT_ROOT":/app \
    -w /app/backend \
    golang:1.23 \
    bash -c "go mod tidy && go mod download && GOOS=linux GOARCH=amd64 go build -o /app/.deploy-build/shiboroom-api-linux-amd64 ./cmd/api"
fi

echo "  ビルド済みバイナリ: $BACKEND_BIN_LOCAL"

########################################
# [2] Next.js フロントエンド ビルド（ローカル）
########################################
echo "==== [2] Next.js フロントエンド ビルド ===="
cd "$FRONTEND_SRC"

if [ -f package-lock.json ]; then
  echo "→ npm ci を実行..."
  npm ci
else
  echo "→ npm install を実行..."
  npm install
fi

echo "→ 本番環境用にビルドします (NODE_ENV=production)"
export NODE_ENV=production
export NEXT_PUBLIC_API_URL=https://shiboroom.com
npm run build

echo "  Next.js build 完了"

########################################
# [3] フロントエンド成果物をサーバーへ反映
########################################
echo "==== [3] フロントエンドをサーバーへデプロイ ===="

# サーバー側ディレクトリを事前作成
ssh "$REMOTE_HOST" "mkdir -p ${REMOTE_FRONTEND_DIR}/{.next/{standalone,static},public}"
ssh "$REMOTE_HOST" "mkdir -p ${REMOTE_APP_ROOT}/backend"
ssh "$REMOTE_HOST" "mkdir -p ${REMOTE_CONFIG_DIR}"

# .next/standalone
rsync -avz --delete \
  .next/standalone/ \
  "${REMOTE_HOST}:${REMOTE_FRONTEND_DIR}/.next/standalone/"

# .next/static
rsync -avz --delete \
  .next/static/ \
  "${REMOTE_HOST}:${REMOTE_FRONTEND_DIR}/.next/static/"

# public（存在する場合のみ）
if [ -d "public" ]; then
  rsync -avz --delete \
    public/ \
    "${REMOTE_HOST}:${REMOTE_FRONTEND_DIR}/public/"
else
  echo "  public ディレクトリが存在しないためスキップ"
fi

# 設定ファイルも同期
rsync -avz \
  next.config.js package.json \
  "${REMOTE_HOST}:${REMOTE_FRONTEND_DIR}/"

echo "==== [3.5] 静的ファイルをstandaloneディレクトリにコピー ===="
ssh "$REMOTE_HOST" << 'EOF'
set -e
echo "→ .next/static をコピー"
cp -r /var/www/shiboroom/frontend/.next/static /var/www/shiboroom/frontend/.next/standalone/.next/static
echo "→ public をコピー（存在する場合）"
if [ -d "/var/www/shiboroom/frontend/public" ]; then
  cp -r /var/www/shiboroom/frontend/public /var/www/shiboroom/frontend/.next/standalone/public
fi
echo "✅ 静的ファイルのコピー完了"
EOF

########################################
# [4] バックエンド設定ファイルをデプロイ
########################################
echo "==== [4] バックエンド設定ファイルをデプロイ (スキップ) ===="
echo "→ 設定ファイルは手動管理のためデプロイをスキップします"
# NOTE: scraper_config.yaml contains sensitive data (passwords) and is managed manually on the server
# rsync -avz "$BACKEND_SRC/config/scraper_config.yaml" "${REMOTE_HOST}:${REMOTE_CONFIG_DIR}/"

########################################
# [5] バックエンドバイナリをサーバーへ反映
########################################
echo "==== [5] バックエンドバイナリをサーバーへデプロイ ===="

# 別名でアップロードしてから置き換え
scp "$BACKEND_BIN_LOCAL" "${REMOTE_HOST}:${REMOTE_APP_ROOT}/backend/shiboroom-api.new"
ssh "$REMOTE_HOST" "chmod +x ${REMOTE_APP_ROOT}/backend/shiboroom-api.new && mv ${REMOTE_APP_ROOT}/backend/shiboroom-api.new ${REMOTE_BACKEND_BIN}"

########################################
# [6] サーバー側サービス再起動
########################################
echo "==== [6] サーバー側サービス再起動 ===="

# サーバー側の再起動スクリプトを実行
ssh "$REMOTE_HOST" "/var/www/shiboroom/restart-shiboroom-services.sh"

echo ""
echo "==== ✅ ローカルビルド → 本番デプロイ 完了 ===="
echo ""
echo "アクセスURL:"
echo "  - https://shiboroom.com"
echo ""
