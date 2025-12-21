#!/bin/bash

# Fix Backend API Database Connection
# This script updates the systemd service to use MySQL instead of PostgreSQL

set -e

SERVER="grik@162.43.74.38"

echo "🔧 Fixing Backend API Database Connection..."
echo ""

# Create the updated systemd service file
ssh $SERVER 'cat > /tmp/shiboroom-backend.service << "EOF"
[Unit]
Description=Shiboroom Backend API Server
After=network.target mysql.service meilisearch.service

[Service]
Type=simple
User=grik
Group=grik
WorkingDirectory=/var/www/shiboroom/backend
ExecStart=/var/www/shiboroom/backend/shiboroom-api

# Environment variables
Environment="CONFIG_PATH=/var/www/shiboroom/config/scraper_config.yaml"
Environment="DB_TYPE=mysql"
Environment="DB_HOST=127.0.0.1"
Environment="DB_PORT=3306"
Environment="DB_USER=shiboroom_user"
Environment="DB_PASSWORD=Kihara0725$"
Environment="DB_NAME=shiboroom"
Environment="PORT=8085"
Environment="ENV=production"

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=shiboroom-backend

# Restart policy
Restart=always
RestartSec=5

# Security settings
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
'

echo "✓ Created updated service file"
echo ""

# Install and restart the service
echo "📝 Installing updated service file..."
ssh $SERVER << 'ENDSSH'
# Move the service file
sudo mv /tmp/shiboroom-backend.service /etc/systemd/system/shiboroom-backend.service

# Reload systemd
echo "🔄 Reloading systemd..."
sudo systemctl daemon-reload

# Restart the service
echo "🚀 Restarting backend service..."
sudo systemctl restart shiboroom-backend

# Wait for service to start
sleep 3

# Check status
echo ""
echo "📊 Service Status:"
sudo systemctl status shiboroom-backend --no-pager | head -15

# Test API
echo ""
echo "🧪 Testing API..."
if curl -s http://localhost:8085/api/properties?limit=1 > /dev/null 2>&1; then
  echo "✅ API is responding!"
  PROPERTY_COUNT=$(curl -s http://localhost:8085/api/properties 2>/dev/null | jq 'length' 2>/dev/null || echo "?")
  echo "📊 Properties in database: $PROPERTY_COUNT"
else
  echo "❌ API is not responding yet. Check logs:"
  echo "   sudo journalctl -u shiboroom-backend -n 20"
fi
ENDSSH

echo ""
echo "✅ Backend fix complete!"
echo ""
echo "You can verify by visiting: https://shiboroom.com"
