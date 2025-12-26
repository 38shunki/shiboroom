#!/bin/bash
# Server-side backend update script
# Upload this to the server and run it there with sudo

set -e

echo "🔄 Updating shiboroom backend..."

# Stop the service
echo "⏸️  Stopping backend service..."
systemctl stop shiboroom-backend

# Backup current binary
if [ -f /var/www/shiboroom/backend/shiboroom-api ]; then
  echo "💾 Backing up current binary..."
  cp /var/www/shiboroom/backend/shiboroom-api /var/www/shiboroom/backend/shiboroom-api.backup.$(date +%Y%m%d_%H%M%S)
fi

# Install new binary
echo "📂 Installing new binary..."
cp /tmp/shiboroom-api-new /var/www/shiboroom/backend/shiboroom-api
chown grik:grik /var/www/shiboroom/backend/shiboroom-api
chmod +x /var/www/shiboroom/backend/shiboroom-api

# Verify binary
echo "🔍 Verifying binary..."
md5sum /var/www/shiboroom/backend/shiboroom-api

# Start the service
echo "▶️  Starting backend service..."
systemctl start shiboroom-backend

# Wait and verify
sleep 2
if systemctl is-active --quiet shiboroom-backend; then
  echo "✅ Backend service is running!"
  echo ""
  echo "📊 Service status:"
  systemctl status shiboroom-backend --no-pager -l
else
  echo "❌ Backend failed to start!"
  echo ""
  echo "📋 Recent logs:"
  journalctl -u shiboroom-backend -n 20 --no-pager
  exit 1
fi

echo ""
echo "✅ Backend updated successfully!"
echo "Expected MD5: 4c667c6b523c381ec1ab5ef3c837dfbb"
