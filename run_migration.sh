#!/bin/bash

# Migration Runner Script for shiboroom.com
# Run database migrations on production server

set -e

SERVER="grik@162.43.74.38"
MIGRATION_FILE="$1"

if [ -z "$MIGRATION_FILE" ]; then
  echo "Usage: $0 <migration_file>"
  echo "Example: $0 backend/migrations/007_add_performance_indexes.sql"
  exit 1
fi

if [ ! -f "$MIGRATION_FILE" ]; then
  echo "Error: Migration file not found: $MIGRATION_FILE"
  exit 1
fi

echo "📤 Uploading migration file to server..."
scp "$MIGRATION_FILE" "$SERVER:/tmp/migration.sql"

echo "🔄 Executing migration on production database..."
ssh "$SERVER" << 'ENDSSH'
  # Execute migration as deploy user with MySQL access
  mysql -u root shiboroom < /tmp/migration.sql

  if [ $? -eq 0 ]; then
    echo "✅ Migration executed successfully"
    rm /tmp/migration.sql
  else
    echo "❌ Migration failed"
    exit 1
  fi
ENDSSH

echo "✅ Migration completed"
