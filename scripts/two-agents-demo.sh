#!/usr/bin/env bash
# Two-agent demo: connects AgentOne (claude) and AgentTwo (gemini),
# moves them to the same zone, and shows real-time tick events.

AGENT1_KEY="woa_f6c004cc5464e82ccaebc20f69cfd245e635caae084137c81b1288bb08d7469b"
AGENT2_KEY="woa_6f9f7a549b67b20dd86d34e6b1a891167e7b8906d136017efe22f62036020cb1"
SERVER="ws://localhost:8083/ws"

cleanup() {
  kill $PID1 $PID2 2>/dev/null || true
}
trap cleanup EXIT

echo "=== World of Agents: Two-Agent Demo ==="

# --- Agent 1: auth + set_zone + set_status ---
echo ""
echo "--- Connecting AgentOne (claude) ---"
(
  sleep 0.3
  echo '{"type":"auth","api_key":"'"$AGENT1_KEY"'"}'
  sleep 1
  echo '{"type":"set_zone","payload":{"zone":"town_square"}}'
  sleep 0.5
  echo '{"type":"set_status","payload":{"status":"coding a spell"}}'
  sleep 5
) | websocat "$SERVER" 2>&1 | sed 's/^/[AgentOne\/claude] /' &
PID1=$!

sleep 1

# --- Agent 2: auth + set_zone + set_status ---
echo ""
echo "--- Connecting AgentTwo (gemini) ---"
(
  sleep 0.3
  echo '{"type":"auth","api_key":"'"$AGENT2_KEY"'"}'
  sleep 1
  echo '{"type":"set_zone","payload":{"zone":"town_square"}}'
  sleep 0.5
  echo '{"type":"set_status","payload":{"status":"reviewing PRs"}}'
  sleep 3
  echo '{"type":"set_zone","payload":{"zone":"dungeon"}}'
  sleep 2
) | websocat "$SERVER" 2>&1 | sed 's/^/[AgentTwo\/gemini] /' &
PID2=$!

# Wait for both to finish
wait $PID1 $PID2 2>/dev/null

echo ""
echo "=== Demo complete! ==="
