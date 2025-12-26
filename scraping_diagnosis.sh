#!/bin/bash
# scraping_diagnosis.sh - スクレイピング状況の診断
# Usage: ./scraping_diagnosis.sh

set -e

echo "=========================================="
echo "   Scraping System Diagnosis"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo ""

# 1. コンテナ状態確認
echo "=== Container Status ==="
docker-compose ps | grep backend || echo "Backend container not running!"
echo ""

# 2. 物件数確認
echo "=== Property Count ==="
PROPERTY_COUNT=$(docker-compose exec -T backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db -N \
  -e "SELECT COUNT(*) FROM properties;" 2>/dev/null' || echo "0")

if [ "$PROPERTY_COUNT" = "0" ]; then
    echo "🔴 NO PROPERTIES IN DATABASE"
    echo "   Possible causes:"
    echo "   - Worker never started"
    echo "   - No URLs were enqueued"
    echo "   - All scraping attempts failed"
else
    echo "✅ Total properties: $PROPERTY_COUNT"
fi

# 最近追加された物件
RECENT_COUNT=$(docker-compose exec -T backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db -N \
  -e "SELECT COUNT(*) FROM properties WHERE created_at > NOW() - INTERVAL 24 HOUR;" 2>/dev/null' || echo "0")
echo "ℹ️  Added in last 24h: $RECENT_COUNT"

echo ""

# 3. キュー状態確認
echo "=== Queue Status ==="
docker-compose exec -T backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db \
  -e "SELECT status, COUNT(*) as count FROM detail_scrape_queue GROUP BY status;" 2>/dev/null' || echo "Queue table not accessible"

echo ""

# 4. Scheduler状態確認
echo "=== Scheduler Status ==="
SCHEDULER_ENABLED=$(docker-compose logs backend 2>/dev/null | grep -c "Scheduler: Started" || echo "0")
SCHEDULER_DISABLED=$(docker-compose logs backend 2>/dev/null | grep -c "Scheduler: Daily run is disabled" || echo "0")

if [ "$SCHEDULER_ENABLED" -gt 0 ]; then
    echo "✅ Scheduler is ENABLED"
elif [ "$SCHEDULER_DISABLED" -gt 0 ]; then
    echo "⚠️  Scheduler is DISABLED in configuration"
    echo "   To enable: Set 'daily_run_enabled: true' in config"
else
    echo "❓ Scheduler status unknown"
fi

echo ""

# 5. Worker状態確認
echo "=== Worker Status ==="
WORKER_STARTED=$(docker-compose logs backend 2>/dev/null | grep -c "Queue worker started" || echo "0")
WORKER_PROCESSING=$(docker-compose logs --since 1h backend 2>/dev/null | grep -c "QueueWorker: Processing" || echo "0")

if [ "$WORKER_STARTED" -eq 0 ]; then
    echo "🔴 Worker never started"
    echo "   Check: logs for startup errors"
elif [ "$WORKER_PROCESSING" -eq 0 ]; then
    echo "⚠️  Worker started but no activity in last hour"
    echo "   Possible causes:"
    echo "   - Queue is empty"
    echo "   - DetailLimiter is waiting"
    echo "   - WAF cooldown active"
else
    echo "✅ Worker is active ($WORKER_PROCESSING items processed in last hour)"
fi

echo ""

# 6. 最近のエラー確認
echo "=== Recent Errors (Last 24h) ==="
ERROR_COUNT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "ERROR\|Failed" || echo "0")
if [ "$ERROR_COUNT" -eq 0 ]; then
    echo "✅ No errors in last 24 hours"
else
    echo "⚠️  $ERROR_COUNT errors detected"
    echo ""
    echo "Last 5 errors:"
    docker-compose logs --since 24h backend 2>/dev/null | grep "ERROR\|Failed" | tail -5
fi

echo ""

# 7. WAF検知確認
echo "=== WAF Detection History ==="
WAF_ALL_TIME=$(docker-compose logs backend 2>/dev/null | grep -c "WAF" || echo "0")
WAF_RECENT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "WAF" || echo "0")

if [ "$WAF_ALL_TIME" -eq 0 ]; then
    echo "✅ No WAF detections (all time)"
else
    echo "ℹ️  Total WAF detections: $WAF_ALL_TIME"
    echo "ℹ️  WAF in last 24h: $WAF_RECENT"
    if [ "$WAF_RECENT" -gt 0 ]; then
        echo "⚠️  Recent WAF activity - system should be in cooldown"
    fi
fi

echo ""

# 8. 推奨アクション
echo "=========================================="
echo "   Recommended Actions"
echo "=========================================="

if [ "$PROPERTY_COUNT" -eq 0 ]; then
    echo "🔴 CRITICAL: No properties in database"
    echo ""
    echo "1. Check if any URLs have been enqueued:"
    echo "   docker-compose exec backend sh -c 'mysql -u realestate_user -prealestate_pass realestate_db -e \"SELECT COUNT(*) FROM detail_scrape_queue;\"'"
    echo ""
    echo "2. If queue is empty, manually enqueue some URLs:"
    echo "   curl -X POST http://localhost:8084/api/scrape/list -H 'Content-Type: application/json' -d '{\"url\": \"...\", \"limit\": 5}'"
    echo ""
    echo "3. Monitor worker activity:"
    echo "   docker-compose logs -f backend | grep QueueWorker"
elif [ "$RECENT_COUNT" -eq 0 ]; then
    echo "⚠️  WARNING: No new properties in last 24 hours"
    echo ""
    echo "1. Check if queue has pending items:"
    echo "   curl http://localhost:8084/api/queue/stats"
    echo ""
    echo "2. Check DetailLimiter wait time:"
    echo "   docker-compose logs --tail 20 backend | grep DetailLimiter"
    echo ""
    echo "3. Monitor for WAF cooldown:"
    echo "   docker-compose logs --tail 50 backend | grep -E '(cooldown|WAF)'"
else
    echo "✅ System appears to be functioning"
    echo ""
    echo "Continue monitoring with:"
    echo "   ./daily_check.sh"
fi

echo ""
