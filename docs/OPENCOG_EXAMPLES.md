# OpenCog Integration Examples

This document provides examples of using the OpenCog services in Jan Server.

## Overview

Jan Server now includes three OpenCog microservices:
- **AtomSpace**: Hypergraph database for knowledge representation
- **CogServer**: Cognitive algorithm scheduler
- **PLN**: Probabilistic Logic Networks reasoning engine

## Example 1: Knowledge Representation with AtomSpace

### Create Concepts

```bash
# Create concept: human
curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ConceptNode",
    "name": "human",
    "truth_value": {"strength": 0.9, "confidence": 0.8}
  }'

# Create concept: mammal
curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ConceptNode",
    "name": "mammal",
    "truth_value": {"strength": 0.95, "confidence": 0.9}
  }'

# Create concept: animal
curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ConceptNode",
    "name": "animal",
    "truth_value": {"strength": 0.98, "confidence": 0.95}
  }'
```

### Create Relationships

```bash
# Create inheritance relationship: human -> mammal
curl -X POST http://localhost:8100/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "type": "InheritanceLink",
    "outgoing": [0, 1],
    "truth_value": {"strength": 0.9, "confidence": 0.85}
  }'

# Create inheritance relationship: mammal -> animal
curl -X POST http://localhost:8100/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "type": "InheritanceLink",
    "outgoing": [1, 2],
    "truth_value": {"strength": 0.95, "confidence": 0.9}
  }'
```

### Query Knowledge

```bash
# Get all concept nodes
curl http://localhost:8100/api/v1/atoms?type=ConceptNode

# Query by pattern
curl -X POST http://localhost:8100/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "pattern": {
      "type": "ConceptNode",
      "name": "human"
    }
  }'

# Get all relationships
curl http://localhost:8100/api/v1/relationships
```

## Example 2: Probabilistic Reasoning with PLN

### Basic Inference Rules

```bash
# Deduction: If A→B and B→C, then A→C
# If human→mammal (0.9, 0.85) and mammal→animal (0.95, 0.9)
# What is the truth value of human→animal?
curl -X POST http://localhost:8102/api/v1/infer/deduction \
  -H "Content-Type: application/json" \
  -d '{
    "premise1": {"strength": 0.9, "confidence": 0.85},
    "premise2": {"strength": 0.95, "confidence": 0.9}
  }'
# Result: {"strength": 0.855, "confidence": 0.85}
```

### Logical Operations

```bash
# Conjunction (AND): What is P(A AND B)?
curl -X POST http://localhost:8102/api/v1/infer/conjunction \
  -H "Content-Type: application/json" \
  -d '{
    "premise1": {"strength": 0.9, "confidence": 0.8},
    "premise2": {"strength": 0.7, "confidence": 0.9}
  }'

# Disjunction (OR): What is P(A OR B)?
curl -X POST http://localhost:8102/api/v1/infer/disjunction \
  -H "Content-Type: application/json" \
  -d '{
    "premise1": {"strength": 0.6, "confidence": 0.8},
    "premise2": {"strength": 0.5, "confidence": 0.7}
  }'

# Negation (NOT): What is P(NOT A)?
curl -X POST http://localhost:8102/api/v1/infer/negation \
  -H "Content-Type: application/json" \
  -d '{
    "premise": {"strength": 0.8, "confidence": 0.9}
  }'
# Result: {"strength": 0.2, "confidence": 0.9}
```

### Advanced Inference

```bash
# Abduction: Generate hypothesis
curl -X POST http://localhost:8102/api/v1/infer/abduction \
  -H "Content-Type: application/json" \
  -d '{
    "premise1": {"strength": 0.8, "confidence": 0.7},
    "premise2": {"strength": 0.9, "confidence": 0.8}
  }'

# Revision: Combine two truth values for the same statement
curl -X POST http://localhost:8102/api/v1/infer/revision \
  -H "Content-Type: application/json" \
  -d '{
    "truth_value1": {"strength": 0.8, "confidence": 0.6},
    "truth_value2": {"strength": 0.7, "confidence": 0.7}
  }'
```

### View Inference History

```bash
# Get recent inference history
curl http://localhost:8102/api/v1/history?limit=10

# Get statistics
curl http://localhost:8102/api/v1/stats
```

## Example 3: Cognitive Agent Management with CogServer

### Create and Manage Agents

```bash
# Create a pattern matching agent
curl -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "pattern_matcher",
    "type": "PatternMatchingAgent",
    "config": {"threshold": 0.5, "max_depth": 3}
  }'

# Create a learning agent
curl -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "learner",
    "type": "LearningAgent",
    "config": {"learning_rate": 0.01, "batch_size": 32}
  }'

# Create a reasoning agent
curl -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "reasoner",
    "type": "ReasoningAgent",
    "config": {"inference_depth": 5}
  }'
```

### Control Agent Execution

```bash
# Start an agent
curl -X POST http://localhost:8101/api/v1/agents/pattern_matcher/start

# Stop an agent
curl -X POST http://localhost:8101/api/v1/agents/pattern_matcher/stop

# Get agent status
curl http://localhost:8101/api/v1/agents/pattern_matcher

# List all agents
curl http://localhost:8101/api/v1/agents
```

### Scheduler Management

```bash
# Start the scheduler
curl -X POST http://localhost:8101/api/v1/scheduler/start

# Get scheduler status
curl http://localhost:8101/api/v1/scheduler/status

# Stop the scheduler
curl -X POST http://localhost:8101/api/v1/scheduler/stop
```

## Example 4: Complete AGI Workflow

This example demonstrates a complete workflow using all three OpenCog services:

```bash
# 1. Create knowledge in AtomSpace
echo "Creating knowledge base..."
curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{"type": "ConceptNode", "name": "cat", "truth_value": {"strength": 0.9, "confidence": 0.85}}'

curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{"type": "ConceptNode", "name": "pet", "truth_value": {"strength": 0.8, "confidence": 0.8}}'

curl -X POST http://localhost:8100/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{"type": "InheritanceLink", "outgoing": [0, 1], "truth_value": {"strength": 0.85, "confidence": 0.8}}'

# 2. Perform reasoning with PLN
echo "Performing deduction..."
curl -X POST http://localhost:8102/api/v1/infer/deduction \
  -H "Content-Type: application/json" \
  -d '{"premise1": {"strength": 0.85, "confidence": 0.8}, "premise2": {"strength": 0.9, "confidence": 0.85}}'

# 3. Create and run cognitive agents
echo "Starting cognitive agents..."
curl -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "knowledge_processor", "type": "ReasoningAgent", "config": {}}'

curl -X POST http://localhost:8101/api/v1/agents/knowledge_processor/start

# 4. Monitor the system
echo "Checking system status..."
curl http://localhost:8100/healthcheck
curl http://localhost:8101/healthcheck
curl http://localhost:8102/healthcheck
```

## Integration with Jan API Gateway

The OpenCog services are accessible through environment variables in the Jan API Gateway:

```go
// In your Go application
atomspaceURL := os.Getenv("OPENCOG_ATOMSPACE_URL")
cogserverURL := os.Getenv("OPENCOG_COGSERVER_URL")
plnURL := os.Getenv("OPENCOG_PLN_URL")

// Use these URLs to integrate OpenCog functionality
// into your AI workflows
```

## Health Monitoring

All OpenCog services provide health check endpoints:

```bash
# Check AtomSpace health
curl http://localhost:8100/healthcheck

# Check CogServer health
curl http://localhost:8101/healthcheck

# Check PLN health
curl http://localhost:8102/healthcheck
```

## Version Information

```bash
# Get service versions
curl http://localhost:8100/v1/version
curl http://localhost:8101/v1/version
curl http://localhost:8102/v1/version
```

## Best Practices

1. **Knowledge Representation**: Use descriptive names for atoms and maintain consistent truth values
2. **Reasoning**: Chain inference rules for complex reasoning tasks
3. **Agent Management**: Monitor agent performance and adjust configurations
4. **Error Handling**: Always check response status and handle errors appropriately
5. **Performance**: Use appropriate batch sizes and consider service load

## Troubleshooting

If you encounter issues:

1. Check service health: `curl http://localhost:810X/healthcheck`
2. Review service logs in the Kubernetes cluster
3. Verify network connectivity between services
4. Ensure proper environment variable configuration

For more details, refer to the service-specific README files in each service directory.
