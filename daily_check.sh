#!/bin/bash
# Quick daily health check for shiboroom scraping

SERVER="grik@162.43.74.38"
HOURS=${1:-24}

echo "======================================"
echo " Shiboroom Daily Health Check (${HOURS}h)"
echo "======================================"
echo ""

# Success count
SUCCESS=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager 2>/dev/null | grep -i 'successfully scraped' | wc -l")
echo "✅ Successful scrapes: $SUCCESS"

# Circuit breaker / WAF blocks
BLOCKS=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager 2>/dev/null | grep -iE 'circuit.*breaker.*open|waf.*block|blocked by waf' | wc -l")
if [ "$BLOCKS" -eq 0 ]; then
    echo "🟢 No WAF/Circuit Breaker blocks"
else
    echo "🔴 WAF/CB blocks detected: $BLOCKS"
fi

# Current rate
RATE=$(ssh $SERVER "journalctl -u shiboroom-backend --since '1 hour ago' --no-pager 2>/dev/null | grep 'DetailLimiter' | tail -1 | grep -oE '[0-9]+/[0-9]+' | tail -1")
if [ -n "$RATE" ]; then
    echo "⚡ Current rate: $RATE per hour"
else
    echo "ℹ️  No recent rate limiter activity"
fi

# Queue status
QUEUE_STATUS=$(ssh $SERVER "mysql -h 127.0.0.1 -u shiboroom_user -p'Kihara0725\$' shiboroom -sN -e \"SELECT CONCAT('Pending: ', COUNT(*)) FROM detail_scrape_queue WHERE status='pending';\" 2>/dev/null")
echo "📋 Queue: $QUEUE_STATUS"

echo ""
echo "======================================"
