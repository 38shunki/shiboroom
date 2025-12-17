#!/bin/bash

# PoC検証スクリプト実行
# Phase 0の4つの検証項目を自動テスト

set -e

echo "======================================"
echo "Phase 0: PoC検証スクリプト"
echo "======================================"
echo ""

# カレントディレクトリを確認
if [ ! -d "backend" ]; then
  echo "Error: このスクリプトはプロジェクトルートから実行してください"
  exit 1
fi

cd backend

# テスト対象URLの設定（環境変数で上書き可能）
if [ -z "$TEST_LIST_URL" ]; then
  # デフォルト: 東京23区の賃貸検索結果
  export TEST_LIST_URL="https://realestate.yahoo.co.jp/rent/search/0123/list/"
  echo "TEST_LIST_URL: $TEST_LIST_URL (default)"
else
  echo "TEST_LIST_URL: $TEST_LIST_URL (custom)"
fi

echo ""
echo "テスト開始..."
echo ""

# Goモジュールの依存関係を確認
if [ ! -f "go.mod" ]; then
  echo "Error: go.mod が見つかりません"
  exit 1
fi

# テストスクリプトの実行
go run cmd/test-poc/main.go

echo ""
echo "======================================"
echo "テスト完了"
echo "======================================"
