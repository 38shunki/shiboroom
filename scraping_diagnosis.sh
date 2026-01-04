#!/bin/bash

# Shiboroom Scraping Health Check (Remote)
# Usage: ./scraping_diagnosis.sh [hours] (default: 24)

HOURS=${1:-24}
SERVER="grik@162.43.74.38"

echo "=================================================="
echo "   Shiboroom Scraping Health Check (${HOURS}h)"
echo "=================================================="
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check 1: Circuit Breaker / WAF / HTTP Errors
echo "🔍 [1] Checking for blocks and errors..."
CB_COUNT=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep -i 'CIRCUIT BREAKER OPEN' | wc -l")
WAF_COUNT=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep -i 'WAF' | wc -l")
ERROR_403=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep -E ' 403 | status.?403' | wc -l")
ERROR_429=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep -E ' 429 | status.?429' | wc -l")
ERROR_500=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep -E ' 500 | status.?500' | wc -l")

BLOCK_TOTAL=$((CB_COUNT + WAF_COUNT + ERROR_403 + ERROR_429))

if [ $BLOCK_TOTAL -eq 0 ]; then
    echo -e "   ${GREEN}✅ PASS${NC}: No blocks detected"
    echo "   - Circuit Breaker: $CB_COUNT"
    echo "   - WAF blocks: $WAF_COUNT"
    echo "   - 403 errors: $ERROR_403"
    echo "   - 429 errors: $ERROR_429"
    echo "   - 500 errors: $ERROR_500"
else
    echo -e "   ${RED}❌ FAIL${NC}: Blocks detected!"
    echo "   - Circuit Breaker: $CB_COUNT"
    echo "   - WAF blocks: $WAF_COUNT"
    echo "   - 403 errors: $ERROR_403"
    echo "   - 429 errors: $ERROR_429"
    echo "   - 500 errors: $ERROR_500"
    echo -e "   ${YELLOW}⚠️  ACTION: Consider reducing to 5-6/hour${NC}"
fi
echo ""

# Check 2: Quality metrics
echo "🎯 [2] Checking scraping quality..."
NO_TITLE=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep -i 'No Title' | wc -l")
STATIONS_ZERO=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep 'stations_len=0' | wc -l")
SUCCESSFUL=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep 'Successfully scraped property' | wc -l")

if [ $SUCCESSFUL -gt 0 ]; then
    NO_TITLE_RATE=$(awk "BEGIN {printf \"%.1f\", ($NO_TITLE/$SUCCESSFUL)*100}")
    STATIONS_RATE=$(awk "BEGIN {printf \"%.1f\", ($STATIONS_ZERO/$SUCCESSFUL)*100}")

    echo "   Total successful scrapes: $SUCCESSFUL"
    echo "   - 'No Title' count: $NO_TITLE (${NO_TITLE_RATE}%)"
    echo "   - stations_len=0: $STATIONS_ZERO (${STATIONS_RATE}%)"

    # Quality check
    if [ $(echo "$NO_TITLE_RATE < 5.0" | bc) -eq 1 ] && [ $(echo "$STATIONS_RATE < 20.0" | bc) -eq 1 ]; then
        echo -e "   ${GREEN}✅ PASS${NC}: Quality metrics good"
    else
        echo -e "   ${YELLOW}⚠️  WARN${NC}: Quality degradation detected"
        echo -e "   ${YELLOW}⚠️  ACTION: Monitor closely or reduce rate${NC}"
    fi
else
    echo -e "   ${YELLOW}⚠️  No scrapes found in last ${HOURS}h${NC}"
fi
echo ""

# Check 3: Rate limiter performance
echo "⏱️  [3] Checking rate limiter..."
LIMITER_LOGS=$(ssh $SERVER "journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager | grep '\[DetailLimiter\]' | tail -10")

# Extract current rate from latest log
CURRENT_RATE=$(echo "$LIMITER_LOGS" | tail -1 | grep -oP '\d+/\d+' | tail -1)

if [ -n "$CURRENT_RATE" ]; then
    echo "   Current rate: $CURRENT_RATE per hour"
    echo -e "   ${GREEN}✅ PASS${NC}: Rate limiter active"
    echo ""
    echo "   Recent activity:"
    echo "$LIMITER_LOGS" | sed 's/^/   /'
else
    echo -e "   ${YELLOW}⚠️  No rate limiter activity found${NC}"
fi
echo ""

# Check 4: Summary
echo "=================================================="
echo "   SUMMARY"
echo "=================================================="

if [ $BLOCK_TOTAL -eq 0 ]; then
    if [ $SUCCESSFUL -gt 0 ]; then
        if [ $(echo "$NO_TITLE_RATE < 5.0" | bc) -eq 1 ] && [ $(echo "$STATIONS_RATE < 20.0" | bc) -eq 1 ]; then
            echo -e "${GREEN}🎉 ALL SYSTEMS GO${NC}"
            echo "   Current rate (8/hour) is safe to continue"
            echo "   Consider increasing to 10/hour after 48h if stable"
        else
            echo -e "${YELLOW}⚠️  QUALITY DEGRADATION${NC}"
            echo "   Monitor for 24h before making changes"
        fi
    fi
else
    echo -e "${RED}🚨 BLOCKS DETECTED${NC}"
    echo "   IMMEDIATE ACTION: Reduce rate to 5-6/hour"
    echo "   Review logs for WAF/rate limit patterns"
fi
echo ""

echo "Next check: ./scraping_diagnosis.sh 24"
echo "Full logs: ssh $SERVER \"journalctl -u shiboroom-backend --since '${HOURS} hours ago' --no-pager\""
echo ""
