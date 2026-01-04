#!/bin/bash

# Unified Deployment Script for shiboroom.com
# Deploy both frontend and backend to production server

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVER="grik@162.43.74.38"
PROJECT_ROOT="/Users/shu/Documents/dev/real-estate-portal"

# Parse arguments
DEPLOY_FRONTEND=false
DEPLOY_BACKEND=false

if [ $# -eq 0 ]; then
  # No arguments: deploy both
  DEPLOY_FRONTEND=true
  DEPLOY_BACKEND=true
else
  # Parse arguments
  for arg in "$@"; do
    case $arg in
      frontend|front|fe)
        DEPLOY_FRONTEND=true
        ;;
      backend|back|be)
        DEPLOY_BACKEND=true
        ;;
      all|both)
        DEPLOY_FRONTEND=true
        DEPLOY_BACKEND=true
        ;;
      *)
        echo -e "${RED}Unknown argument: $arg${NC}"
        echo "Usage: $0 [frontend|backend|all]"
        echo "  No arguments = deploy both"
        exit 1
        ;;
    esac
  done
fi

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   shiboroom.com Deployment Script     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# ============================================
# Backend Deployment
# ============================================
if [ "$DEPLOY_BACKEND" = true ]; then
  echo -e "${YELLOW}📦 [1/2] Backend Deployment${NC}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  cd "$PROJECT_ROOT/backend"

  echo "🔨 Building Go binary for Linux using Docker..."
  docker run --rm --platform linux/amd64 -v "$(pwd)":/app -w /app golang:1.23-alpine sh -c \
    "apk add --no-cache gcc musl-dev && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-extldflags \"-static\"' -o shiboroom-api ./cmd/api"

  echo "📤 Uploading to server..."
  scp shiboroom-api "$SERVER:/tmp/"

  echo "🚀 Deploying on server..."
  ssh "$SERVER" << 'ENDSSH'
    # Backup (no sudo needed - grik owns the directory)
    if [ -f /var/www/shiboroom/backend/shiboroom-api ]; then
      echo "💾 Backing up current binary..."
      cp /var/www/shiboroom/backend/shiboroom-api /var/www/shiboroom/backend/shiboroom-api.backup.$(date +%Y%m%d_%H%M%S)
    fi

    # Install (no sudo needed - grik owns the directory)
    echo "📂 Installing new binary..."
    mv /tmp/shiboroom-api /var/www/shiboroom/backend/shiboroom-api
    chmod +x /var/www/shiboroom/backend/shiboroom-api

    # Restart (only this needs sudo)
    echo "🔄 Restarting backend service..."
    sudo systemctl restart shiboroom-backend

    # Verify
    sleep 2
    if systemctl is-active --quiet shiboroom-backend; then
      echo "✅ Backend service is running!"
    else
      echo "❌ Backend failed to start!"
      sudo journalctl -u shiboroom-backend -n 10 --no-pager
      exit 1
    fi
ENDSSH

  # Clean up
  rm -f "$PROJECT_ROOT/backend/shiboroom-api"

  echo -e "${GREEN}✅ Backend deployed successfully!${NC}"
  echo ""
fi

# ============================================
# Frontend Deployment
# ============================================
if [ "$DEPLOY_FRONTEND" = true ]; then
  echo -e "${YELLOW}🎨 [2/2] Frontend Deployment${NC}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  # Pre-deployment cleanup - remove conflicting app directory locally
  echo "🧹 Pre-deployment cleanup..."
  if [ -d "$PROJECT_ROOT/frontend/app" ]; then
    echo "   Removing local app/ directory to prevent routing conflicts..."
    rm -rf "$PROJECT_ROOT/frontend/app"
  fi

  echo "📦 Syncing source code to server..."
  rsync -avz --delete \
    --exclude 'node_modules' \
    --exclude '.next' \
    --exclude '.git' \
    --exclude '.env.local' \
    --quiet \
    "$PROJECT_ROOT/frontend/" "$SERVER:/var/www/shiboroom/frontend/"

  echo "🔨 Building on server..."
  ssh "$SERVER" << 'ENDSSH'
    cd /var/www/shiboroom/frontend

    # Critical: Remove conflicting directories and files
    echo "🧹 Removing conflicting directories..."
    rm -rf app  # Remove root app/ directory (causes Pages Router fallback)
    rm -f .env.local  # Remove local env file (would override production)

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
      echo "📥 Installing dependencies..."
      npm install --quiet
    fi

    # Clean build
    echo "🧹 Cleaning old build..."
    rm -rf .next

    # Build with production environment
    echo "🏗️  Building with production environment..."
    NEXT_PUBLIC_API_URL=https://shiboroom.com NODE_ENV=production npm run build

    # Verify App Router was used (not Pages Router)
    if [ -d ".next/server/app" ]; then
      echo "✅ App Router build verified"
    else
      echo "⚠️  Warning: Build may not be using App Router"
    fi

    # Verify production API URL is in build
    if grep -q "shiboroom.com" .next/static/chunks/app/page*.js 2>/dev/null || \
       grep -q "shiboroom.com" .next/server/app/page.js 2>/dev/null; then
      echo "✅ Production API URL verified in build"
    else
      echo "⚠️  Warning: Could not verify production API URL in build"
    fi

    # Copy static files
    echo "📂 Copying static files..."
    cp -r .next/static .next/standalone/.next/
    cp -r public .next/standalone/

    # Restart service (only this needs sudo)
    echo "🔄 Restarting frontend service..."
    sudo systemctl restart shiboroom-frontend

    # Verify
    sleep 2
    if systemctl is-active --quiet shiboroom-frontend; then
      echo "✅ Frontend service is running!"
    else
      echo "❌ Frontend failed to start!"
      sudo journalctl -u shiboroom-frontend -n 10 --no-pager
      exit 1
    fi
ENDSSH

  echo -e "${GREEN}✅ Frontend deployed successfully!${NC}"
  echo ""
fi

# ============================================
# Summary
# ============================================
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          Deployment Complete!          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
if [ "$DEPLOY_BACKEND" = true ]; then
  echo -e "  🔌 Backend API: ${GREEN}https://shiboroom.com/api${NC}"
fi
if [ "$DEPLOY_FRONTEND" = true ]; then
  echo -e "  🌐 Frontend:    ${GREEN}https://shiboroom.com${NC}"
fi
echo ""
echo -e "${YELLOW}💡 Tip: Visit https://shiboroom.com to verify the deployment${NC}"
echo ""
