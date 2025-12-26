#!/bin/bash
# daily_check.sh - 毎日実行する監視スクリプト
# Usage: ./daily_check.sh

set -e

echo "=========================================="
echo "   Queue Worker Daily Health Check"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo ""

# Quick 1-line summary
QUICK_PENDING=$(curl -s http://localhost:8084/api/queue/stats 2>/dev/null | jq -r '.pending // "?"')
QUICK_DONE=$(curl -s http://localhost:8084/api/queue/stats 2>/dev/null | jq -r '.done // "?"')
QUICK_FAIL=$(curl -s http://localhost:8084/api/queue/stats 2>/dev/null | jq -r '.permanent_fail // "?"')
QUICK_RUNNING=$(curl -s http://localhost:8084/api/queue/stats 2>/dev/null | jq -r '.is_running // false')

echo "📊 Quick Status: Worker=$QUICK_RUNNING | Pending=$QUICK_PENDING | Done=$QUICK_DONE | PermanentFail=$QUICK_FAIL"
echo ""

# 1. Queue Stats
echo "=== Queue Stats ==="
STATS=$(curl -s http://localhost:8084/api/queue/stats)
echo "$STATS" | jq .

# Extract values for threshold checks
PENDING=$(echo "$STATS" | jq -r '.pending // 0')
DONE=$(echo "$STATS" | jq -r '.done // 0')
PERMANENT_FAIL=$(echo "$STATS" | jq -r '.permanent_fail // 0')
IS_RUNNING=$(echo "$STATS" | jq -r '.is_running // false')

echo ""
echo "=== Status Check ==="

# Check if worker is running
if [ "$IS_RUNNING" = "true" ]; then
    echo "✅ Worker is running"
else
    echo "❌ Worker is NOT running - check logs!"
fi

# Check pending queue size
if [ "$PENDING" -lt 50 ]; then
    echo "✅ Pending queue is healthy ($PENDING items)"
elif [ "$PENDING" -lt 200 ]; then
    echo "⚠️  Pending queue is elevated ($PENDING items) - monitor closely"
else
    echo "🔴 Pending queue is HIGH ($PENDING items) - investigate immediately!"
fi

# Check 404 ratio
TOTAL=$((DONE + PERMANENT_FAIL))
if [ "$TOTAL" -gt 0 ]; then
    FAIL_RATIO=$((PERMANENT_FAIL * 100 / TOTAL))
    if [ "$FAIL_RATIO" -lt 10 ]; then
        echo "✅ 404 ratio is healthy ($FAIL_RATIO%)"
    elif [ "$FAIL_RATIO" -lt 30 ]; then
        echo "⚠️  404 ratio is elevated ($FAIL_RATIO%) - check URL generation"
    else
        echo "🔴 404 ratio is HIGH ($FAIL_RATIO%) - URL generation likely broken!"
    fi
else
    echo "ℹ️  No completed items yet"
fi

echo ""
echo "=== Recent Activity (Last 24h) ==="

# Count successful completions
SUCCESS_COUNT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "QueueWorker: ✅" || echo "0")
echo "✅ Successful scrapes: $SUCCESS_COUNT"
if [ "$SUCCESS_COUNT" -eq 0 ]; then
    echo "   🔴 ZERO successes - Worker may be stuck!"
elif [ "$SUCCESS_COUNT" -lt 50 ]; then
    echo "   ⚠️  Low activity (expected: 100-120/day)"
else
    echo "   ✅ Normal activity (expected: 100-120/day)"
fi

# Count WAF detections
WAF_COUNT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "WAF" || echo "0")
if [ "$WAF_COUNT" -eq 0 ]; then
    echo "✅ WAF detections: 0 (good)"
else
    echo "🔴 WAF detections: $WAF_COUNT (check cooldown is working)"
fi

# Count 404s
NOT_FOUND_COUNT=$(docker-compose logs --since 24h backend 2>/dev/null | grep -c "Permanent failure (404)" || echo "0")
echo "ℹ️  404 errors: $NOT_FOUND_COUNT"
if [ "$SUCCESS_COUNT" -gt 0 ]; then
    RATIO=$((NOT_FOUND_COUNT * 100 / (SUCCESS_COUNT + NOT_FOUND_COUNT)))
    echo "   Ratio: $RATIO% (expected: <10%)"
fi

# Oldest pending check
echo ""
echo "=== Oldest Pending Item ==="
OLDEST=$(docker-compose exec -T backend sh -c '
  mysql -u realestate_user -prealestate_pass realestate_db -N \
  -e "SELECT TIMESTAMPDIFF(HOUR, MIN(created_at), NOW())
      FROM detail_scrape_queue
      WHERE status=\"pending\";" 2>/dev/null' || echo "N/A")

if [ "$OLDEST" = "N/A" ] || [ -z "$OLDEST" ]; then
    echo "ℹ️  No pending items (or DB not accessible)"
elif [ "$OLDEST" -lt 12 ]; then
    echo "✅ Oldest pending: ${OLDEST}h (normal delay)"
elif [ "$OLDEST" -lt 48 ]; then
    echo "⚠️  Oldest pending: ${OLDEST}h (slow processing)"
else
    echo "🔴 Oldest pending: ${OLDEST}h (STUCK - investigate!)"
fi

echo ""
echo "=== Recent Errors (Last 1h) ==="
RECENT_ERRORS=$(docker-compose logs --since 1h backend 2>/dev/null | grep -E "(QueueWorker.*[Ff]ailed|ERROR)" | tail -5 || echo "No errors")
if [ "$RECENT_ERRORS" = "No errors" ]; then
    echo "✅ No recent errors"
else
    echo "$RECENT_ERRORS"
fi

echo ""
echo "=========================================="
echo "   Summary"
echo "=========================================="

# Overall health assessment
HEALTH_SCORE=0

[ "$IS_RUNNING" = "true" ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))
[ "$SUCCESS_COUNT" -gt 0 ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))
[ "$PENDING" -lt 200 ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))
[ "$WAF_COUNT" -eq 0 ] && HEALTH_SCORE=$((HEALTH_SCORE + 1))

if [ $HEALTH_SCORE -eq 4 ]; then
    echo "✅ System is HEALTHY - no action needed"
elif [ $HEALTH_SCORE -ge 2 ]; then
    echo "⚠️  System needs MONITORING - check warnings above"
else
    echo "🔴 System needs ATTENTION - review errors and take action"
fi

echo ""
echo "For detailed logs, run:"
echo "  docker-compose logs -f backend | grep QueueWorker"
echo ""
