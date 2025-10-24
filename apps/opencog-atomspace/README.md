# OpenCog AtomSpace Service

A microservice that provides the OpenCog AtomSpace hypergraph database functionality as a REST API.

## Overview

AtomSpace is the core knowledge representation system in OpenCog. It stores atoms (terms, formulas, sentences) and their relationships in a hypergraph structure.

## Features

- Create and manage atoms with different types (ConceptNode, PredicateNode, etc.)
- Store truth values (strength and confidence) for atoms
- Create relationships between atoms (InheritanceLink, SimilarityLink, etc.)
- Query atoms by patterns
- Support for basic AtomSpace operations

## API Endpoints

### Health & Version
- `GET /healthcheck` - Health check
- `GET /v1/version` - Service version information

### Atom Management
- `POST /api/v1/atoms` - Create a new atom
- `GET /api/v1/atoms` - List all atoms (optional type filter)
- `GET /api/v1/atoms/{id}` - Get atom by ID
- `DELETE /api/v1/atoms/{id}` - Delete an atom

### Relationship Management
- `POST /api/v1/relationships` - Create a relationship between atoms
- `GET /api/v1/relationships` - List all relationships

### Query & Utilities
- `POST /api/v1/query` - Query atoms by pattern
- `POST /api/v1/clear` - Clear all atoms and relationships

## Example Usage

### Create an Atom
```bash
curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ConceptNode",
    "name": "human",
    "truth_value": {"strength": 0.9, "confidence": 0.8}
  }'
```

### Create a Relationship
```bash
curl -X POST http://localhost:8100/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "type": "InheritanceLink",
    "outgoing": [0, 1],
    "truth_value": {"strength": 0.9, "confidence": 0.8}
  }'
```

### Query Atoms
```bash
curl -X POST http://localhost:8100/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "pattern": {
      "type": "ConceptNode"
    }
  }'
```

## Configuration

Environment variables:
- `PORT` - Service port (default: 8100)
- `HOST` - Service host (default: 0.0.0.0)

## Building and Running

### Docker
```bash
docker build -t opencog-atomspace:latest .
docker run -p 8100:8100 opencog-atomspace:latest
```

### Local Development
```bash
pip install -r requirements.txt
python app.py
```

## Architecture

This is a simplified Python implementation of OpenCog AtomSpace. For production use with full OpenCog capabilities, consider integrating with the C++ OpenCog library.

## Integration

This service is designed to work with:
- OpenCog CogServer (cognitive algorithm scheduler)
- OpenCog PLN (probabilistic reasoning)
- Jan API Gateway (main API gateway)
