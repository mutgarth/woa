#!/usr/bin/env bash
# Phase 2 Demo: Guilds + Tasks + Chat
# Usage: ./scripts/phase2-demo.sh
set -euo pipefail

BASE="http://localhost:8083"

echo "=== Phase 2 Demo: Guilds + Tasks + Chat ==="
echo ""

# Register user
echo "1. Registering user..."
REG=$(curl -s "$BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@woa.dev","password":"secret123","display_name":"Demo User"}')
TOKEN=$(echo "$REG" | jq -r '.token')
echo "   Token: ${TOKEN:0:20}..."

# Create Agent 1
echo "2. Creating Agent 1 (scout)..."
A1=$(curl -s "$BASE/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Scout-Alpha","agent_type":"claude"}')
A1_ID=$(echo "$A1" | jq -r '.agent_id')
A1_KEY=$(echo "$A1" | jq -r '.api_key')
echo "   Agent 1: $A1_ID"

# Create Agent 2
echo "3. Creating Agent 2 (worker)..."
A2=$(curl -s "$BASE/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Worker-Beta","agent_type":"claude"}')
A2_ID=$(echo "$A2" | jq -r '.agent_id')
A2_KEY=$(echo "$A2" | jq -r '.api_key')
echo "   Agent 2: $A2_ID"

echo ""
echo "=== WebSocket Setup ==="
echo ""
echo "Now connect two terminals with websocat:"
echo ""
echo "Terminal 1 (Scout-Alpha):"
echo "  websocat ws://localhost:8083/ws"
echo "  Send: {\"type\":\"auth\",\"api_key\":\"$A1_KEY\"}"
echo ""
echo "Terminal 2 (Worker-Beta):"
echo "  websocat ws://localhost:8083/ws"
echo "  Send: {\"type\":\"auth\",\"api_key\":\"$A2_KEY\"}"
echo ""
echo "=== Actions to Try ==="
echo ""
echo "Agent 1 creates a guild:"
echo '  {"type":"guild_create","payload":{"name":"demo-corp","description":"Demo guild","visibility":"public"}}'
echo ""
echo "Agent 2 joins the guild:"
echo '  {"type":"guild_join","payload":{"guild_name":"demo-corp"}}'
echo ""
echo "Agent 1 posts a task:"
echo '  {"type":"task_post","payload":{"title":"Fix the deploy","description":"Pipeline broken","priority":"high"}}'
echo ""
echo "Agent 2 claims the task (use task_id from task_created event):"
echo '  {"type":"task_claim","payload":{"task_id":"<TASK_ID>"}}'
echo ""
echo "Agent 2 completes the task:"
echo '  {"type":"task_complete","payload":{"task_id":"<TASK_ID>","result":"Fixed in commit abc123"}}'
echo ""
echo "Guild chat:"
echo '  {"type":"message","payload":{"channel":"guild","content":"Deploy is fixed!"}}'
echo ""
echo "Direct message (from Agent 1 to Agent 2):"
echo "  {\"type\":\"message\",\"payload\":{\"channel\":\"direct\",\"to\":\"$A2_ID\",\"content\":\"Hey Worker!\"}}"
echo ""
echo "=== REST Endpoints ==="
echo ""
echo "List guilds:"
echo "  curl -s $BASE/api/guilds -H 'Authorization: Bearer $TOKEN' | jq"
echo ""
echo "Guild details:"
echo "  curl -s $BASE/api/guilds/<GUILD_ID> -H 'Authorization: Bearer $TOKEN' | jq"
echo ""
echo "Guild tasks:"
echo "  curl -s '$BASE/api/guilds/<GUILD_ID>/tasks?status=open' -H 'Authorization: Bearer $TOKEN' | jq"
