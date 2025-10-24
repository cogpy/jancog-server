# OpenCog PLN (Probabilistic Logic Networks) Service

A microservice that provides probabilistic reasoning and inference capabilities.

## Overview

PLN is the reasoning engine in OpenCog that performs logical inference with probabilistic truth values. It enables sophisticated reasoning over knowledge stored in AtomSpace.

## Features

- Deduction inference (A→B, B→C ⇒ A→C)
- Induction inference (generalization from instances)
- Abduction inference (hypothesis generation)
- Conjunction (AND) and Disjunction (OR) operations
- Negation (NOT) operations
- Revision (combining truth values)
- Truth value calculations with strength and confidence
- Inference history tracking

## API Endpoints

### Health & Version
- `GET /healthcheck` - Health check
- `GET /v1/version` - Service version information

### Inference Rules
- `POST /api/v1/infer/deduction` - Apply deduction rule
- `POST /api/v1/infer/induction` - Apply induction rule
- `POST /api/v1/infer/abduction` - Apply abduction rule
- `POST /api/v1/infer/conjunction` - Apply conjunction (AND) rule
- `POST /api/v1/infer/disjunction` - Apply disjunction (OR) rule
- `POST /api/v1/infer/negation` - Apply negation (NOT) rule
- `POST /api/v1/infer/revision` - Apply revision rule

### History & Statistics
- `GET /api/v1/history` - Get inference history
- `POST /api/v1/history/clear` - Clear inference history
- `GET /api/v1/stats` - Get inference statistics

## Truth Values

All PLN operations use truth values with two components:
- **Strength**: Probability value (0.0 to 1.0)
- **Confidence**: Certainty of the probability (0.0 to 1.0)

## Example Usage

### Deduction Inference
```bash
curl -X POST http://localhost:8102/api/v1/infer/deduction \
  -H "Content-Type: application/json" \
  -d '{
    "premise1": {"strength": 0.9, "confidence": 0.8},
    "premise2": {"strength": 0.8, "confidence": 0.7}
  }'
```

### Conjunction (AND)
```bash
curl -X POST http://localhost:8102/api/v1/infer/conjunction \
  -H "Content-Type: application/json" \
  -d '{
    "premise1": {"strength": 0.9, "confidence": 0.8},
    "premise2": {"strength": 0.7, "confidence": 0.9}
  }'
```

### Negation (NOT)
```bash
curl -X POST http://localhost:8102/api/v1/infer/negation \
  -H "Content-Type: application/json" \
  -d '{
    "premise": {"strength": 0.8, "confidence": 0.9}
  }'
```

### Get Statistics
```bash
curl http://localhost:8102/api/v1/stats
```

## Inference Rules

### Deduction
If A→B (with truth value TV1) and B→C (with TV2), then A→C with combined truth value.

### Induction
Generalize from specific instances to create general rules.

### Abduction
Generate hypotheses that could explain observations.

### Conjunction
Combine two statements with AND logic: P(A ∧ B) = P(A) × P(B)

### Disjunction
Combine two statements with OR logic: P(A ∨ B) = P(A) + P(B) - P(A) × P(B)

### Negation
Negate a statement: P(¬A) = 1 - P(A)

### Revision
Combine multiple truth values for the same statement, weighted by confidence.

## Configuration

Environment variables:
- `PORT` - Service port (default: 8102)
- `HOST` - Service host (default: 0.0.0.0)

## Building and Running

### Docker
```bash
docker build -t opencog-pln:latest .
docker run -p 8102:8102 opencog-pln:latest
```

### Local Development
```bash
pip install -r requirements.txt
python app.py
```

## Architecture

PLN implements a simplified version of probabilistic reasoning based on the OpenCog PLN framework. It provides REST API endpoints for applying various inference rules.

## Integration

This service is designed to work with:
- OpenCog AtomSpace (queries knowledge for reasoning)
- OpenCog CogServer (scheduled reasoning tasks)
- Jan API Gateway (main API gateway)
