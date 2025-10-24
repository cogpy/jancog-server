#!/bin/bash
# Integration test script for OpenCog services

set -e

echo "================================================"
echo "OpenCog Services Integration Test"
echo "================================================"
echo ""

# Start all services
echo "Starting OpenCog services..."
cd /home/runner/work/jancog-server/jancog-server

python3 apps/opencog-atomspace/app.py &> /tmp/atomspace.log &
ATOMSPACE_PID=$!
echo "AtomSpace started (PID: $ATOMSPACE_PID)"

python3 apps/opencog-cogserver/app.py &> /tmp/cogserver.log &
COGSERVER_PID=$!
echo "CogServer started (PID: $COGSERVER_PID)"

python3 apps/opencog-pln/app.py &> /tmp/pln.log &
PLN_PID=$!
echo "PLN started (PID: $PLN_PID)"

# Wait for services to be ready
echo ""
echo "Waiting for services to be ready..."
sleep 5

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up services..."
    kill $ATOMSPACE_PID 2>/dev/null || true
    kill $COGSERVER_PID 2>/dev/null || true
    kill $PLN_PID 2>/dev/null || true
    wait 2>/dev/null || true
}

trap cleanup EXIT

# Test AtomSpace
echo ""
echo "================================================"
echo "Testing AtomSpace Service"
echo "================================================"

echo "1. Health check..."
curl -s http://localhost:8100/healthcheck | python3 -m json.tool | grep -q "healthy" && echo "✓ Health check passed" || echo "✗ Health check failed"

echo "2. Creating atoms..."
curl -s -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{"type": "ConceptNode", "name": "human", "truth_value": {"strength": 0.9, "confidence": 0.8}}' > /tmp/atom1.json
cat /tmp/atom1.json | python3 -m json.tool | grep -q "success" && echo "✓ Created atom 'human'" || echo "✗ Failed to create atom"

curl -s -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{"type": "ConceptNode", "name": "mammal", "truth_value": {"strength": 0.95, "confidence": 0.9}}' > /tmp/atom2.json
cat /tmp/atom2.json | python3 -m json.tool | grep -q "success" && echo "✓ Created atom 'mammal'" || echo "✗ Failed to create atom"

echo "3. Creating relationship..."
ATOM1_ID=$(cat /tmp/atom1.json | python3 -c "import sys, json; print(json.load(sys.stdin)['atom']['id'])")
ATOM2_ID=$(cat /tmp/atom2.json | python3 -c "import sys, json; print(json.load(sys.stdin)['atom']['id'])")
curl -s -X POST http://localhost:8100/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d "{\"type\": \"InheritanceLink\", \"outgoing\": [$ATOM1_ID, $ATOM2_ID], \"truth_value\": {\"strength\": 0.9, \"confidence\": 0.85}}" | python3 -m json.tool | grep -q "success" && echo "✓ Created relationship" || echo "✗ Failed to create relationship"

echo "4. Querying atoms..."
curl -s http://localhost:8100/api/v1/atoms | python3 -m json.tool | grep -q "count.*2" && echo "✓ Query returned 2 atoms" || echo "✗ Query failed"

# Test PLN
echo ""
echo "================================================"
echo "Testing PLN Service"
echo "================================================"

echo "1. Health check..."
curl -s http://localhost:8102/healthcheck | python3 -m json.tool | grep -q "healthy" && echo "✓ Health check passed" || echo "✗ Health check failed"

echo "2. Testing deduction inference..."
curl -s -X POST http://localhost:8102/api/v1/infer/deduction \
  -H "Content-Type: application/json" \
  -d '{"premise1": {"strength": 0.9, "confidence": 0.8}, "premise2": {"strength": 0.8, "confidence": 0.7}}' | python3 -m json.tool | grep -q "success" && echo "✓ Deduction inference passed" || echo "✗ Deduction inference failed"

echo "3. Testing conjunction inference..."
curl -s -X POST http://localhost:8102/api/v1/infer/conjunction \
  -H "Content-Type: application/json" \
  -d '{"premise1": {"strength": 0.9, "confidence": 0.8}, "premise2": {"strength": 0.7, "confidence": 0.9}}' | python3 -m json.tool | grep -q "success" && echo "✓ Conjunction inference passed" || echo "✗ Conjunction inference failed"

echo "4. Testing negation inference..."
curl -s -X POST http://localhost:8102/api/v1/infer/negation \
  -H "Content-Type: application/json" \
  -d '{"premise": {"strength": 0.8, "confidence": 0.9}}' | python3 -m json.tool | grep -q "success" && echo "✓ Negation inference passed" || echo "✗ Negation inference failed"

echo "5. Checking statistics..."
curl -s http://localhost:8102/api/v1/stats | python3 -m json.tool | grep -q "total_inferences.*3" && echo "✓ Statistics show 3 inferences" || echo "✗ Statistics failed"

# Test CogServer
echo ""
echo "================================================"
echo "Testing CogServer Service"
echo "================================================"

echo "1. Health check..."
curl -s http://localhost:8101/healthcheck | python3 -m json.tool | grep -q "healthy" && echo "✓ Health check passed" || echo "✗ Health check failed"

echo "2. Creating cognitive agent..."
curl -s -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "test_agent", "type": "PatternMatchingAgent", "config": {"threshold": 0.5}}' | python3 -m json.tool | grep -q "success" && echo "✓ Created agent" || echo "✗ Failed to create agent"

echo "3. Starting agent..."
curl -s -X POST http://localhost:8101/api/v1/agents/test_agent/start | python3 -m json.tool | grep -q "success" && echo "✓ Started agent" || echo "✗ Failed to start agent"

echo "4. Checking scheduler status..."
curl -s http://localhost:8101/api/v1/scheduler/status | python3 -m json.tool | grep -q "running.*true" && echo "✓ Scheduler running" || echo "✗ Scheduler not running"

echo "5. Listing agents..."
curl -s http://localhost:8101/api/v1/agents | python3 -m json.tool | grep -q "count.*1" && echo "✓ Found 1 agent" || echo "✗ Agent list failed"

echo ""
echo "================================================"
echo "All tests completed!"
echo "================================================"
