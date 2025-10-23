# OpenCog CogServer Service

A microservice that provides cognitive algorithm scheduling and management functionality.

## Overview

CogServer is a container and scheduler for plug-in cognitive algorithms in OpenCog. It manages the execution and coordination of various AI processes.

## Features

- Create and manage cognitive agents
- Start/stop agents dynamically
- Scheduler for running agents in cycles
- Agent status monitoring and error tracking
- Support for different agent types (PatternMatching, Learning, etc.)

## API Endpoints

### Health & Version
- `GET /healthcheck` - Health check
- `GET /v1/version` - Service version information

### Agent Management
- `POST /api/v1/agents` - Create a new cognitive agent
- `GET /api/v1/agents` - List all agents
- `GET /api/v1/agents/{name}` - Get agent by name
- `DELETE /api/v1/agents/{name}` - Delete an agent
- `POST /api/v1/agents/{name}/start` - Start an agent
- `POST /api/v1/agents/{name}/stop` - Stop an agent

### Scheduler Management
- `POST /api/v1/scheduler/start` - Start the agent scheduler
- `POST /api/v1/scheduler/stop` - Stop the agent scheduler
- `GET /api/v1/scheduler/status` - Get scheduler status

## Example Usage

### Create an Agent
```bash
curl -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "pattern_matcher",
    "type": "PatternMatchingAgent",
    "config": {"threshold": 0.5}
  }'
```

### Start an Agent
```bash
curl -X POST http://localhost:8101/api/v1/agents/pattern_matcher/start
```

### Start Scheduler
```bash
curl -X POST http://localhost:8101/api/v1/scheduler/start
```

### Check Scheduler Status
```bash
curl http://localhost:8101/api/v1/scheduler/status
```

## Configuration

Environment variables:
- `PORT` - Service port (default: 8101)
- `HOST` - Service host (default: 0.0.0.0)

## Building and Running

### Docker
```bash
docker build -t opencog-cogserver:latest .
docker run -p 8101:8101 opencog-cogserver:latest
```

### Local Development
```bash
pip install -r requirements.txt
python app.py
```

## Agent Types

The service supports various cognitive agent types:
- **PatternMatchingAgent** - Pattern recognition and matching
- **LearningAgent** - Machine learning algorithms
- **ReasoningAgent** - Logical reasoning
- **PlanningAgent** - Goal-oriented planning
- **PerceptionAgent** - Sensory data processing

## Architecture

Agents run in a scheduler loop with configurable frequency. The scheduler is implemented as a background thread that executes each active agent in sequence.

## Integration

This service is designed to work with:
- OpenCog AtomSpace (knowledge storage)
- OpenCog PLN (reasoning engine)
- Jan API Gateway (main API gateway)
