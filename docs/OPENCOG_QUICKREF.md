# OpenCog Integration Quick Reference

## Overview

Jan Server now includes OpenCog AGI (Artificial General Intelligence) framework components as Kubernetes-native microservices.

## Services

| Service | Port | Description | Endpoints |
|---------|------|-------------|-----------|
| OpenCog AtomSpace | 8100 | Hypergraph database | `/api/v1/atoms`, `/api/v1/relationships`, `/api/v1/query` |
| OpenCog CogServer | 8101 | Cognitive agent scheduler | `/api/v1/agents`, `/api/v1/scheduler` |
| OpenCog PLN | 8102 | Probabilistic reasoning | `/api/v1/infer/*`, `/api/v1/history`, `/api/v1/stats` |

## Quick Start

### Local Development

1. **Build all services:**
   ```bash
   cd /home/runner/work/jancog-server/jancog-server
   docker build -t menloltd/opencog-atomspace:latest ./apps/opencog-atomspace
   docker build -t menloltd/opencog-cogserver:latest ./apps/opencog-cogserver
   docker build -t menloltd/opencog-pln:latest ./apps/opencog-pln
   ```

2. **Run integration tests:**
   ```bash
   ./scripts/test-opencog-services.sh
   ```

3. **Deploy to Kubernetes:**
   ```bash
   ./scripts/run.sh
   ```

### Kubernetes Deployment

The OpenCog services are deployed automatically when you install the Helm chart:

```bash
helm install jan-server ./charts/jan-server
```

To enable/disable individual services, use these Helm values:

```yaml
opencog:
  atomspace:
    enabled: true  # Set to false to disable
  cogserver:
    enabled: true
  pln:
    enabled: true
```

## Service Configuration

### AtomSpace Configuration

```yaml
opencog:
  atomspace:
    replicaCount: 1
    image:
      repository: menloltd/opencog-atomspace
      tag: "latest"
    service:
      port: 8100
    resources:
      limits:
        cpu: 500m
        memory: 512Mi
```

### CogServer Configuration

```yaml
opencog:
  cogserver:
    replicaCount: 1
    image:
      repository: menloltd/opencog-cogserver
      tag: "latest"
    service:
      port: 8101
    env:
      - name: ATOMSPACE_URL
        value: "http://jan-server-opencog-atomspace:8100"
```

### PLN Configuration

```yaml
opencog:
  pln:
    replicaCount: 1
    image:
      repository: menloltd/opencog-pln
      tag: "latest"
    service:
      port: 8102
    env:
      - name: ATOMSPACE_URL
        value: "http://jan-server-opencog-atomspace:8100"
```

## Service URLs in Jan API Gateway

The gateway automatically receives these environment variables:

```
OPENCOG_ATOMSPACE_URL=http://jan-server-opencog-atomspace:8100
OPENCOG_COGSERVER_URL=http://jan-server-opencog-cogserver:8101
OPENCOG_PLN_URL=http://jan-server-opencog-pln:8102
```

## Common Operations

### Create Knowledge

```bash
# Create an atom
curl -X POST http://localhost:8100/api/v1/atoms \
  -H "Content-Type: application/json" \
  -d '{"type": "ConceptNode", "name": "example", "truth_value": {"strength": 0.9, "confidence": 0.8}}'
```

### Perform Reasoning

```bash
# Deduction inference
curl -X POST http://localhost:8102/api/v1/infer/deduction \
  -H "Content-Type: application/json" \
  -d '{"premise1": {"strength": 0.9, "confidence": 0.8}, "premise2": {"strength": 0.8, "confidence": 0.7}}'
```

### Manage Agents

```bash
# Create an agent
curl -X POST http://localhost:8101/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "my_agent", "type": "PatternMatchingAgent", "config": {}}'

# Start the agent
curl -X POST http://localhost:8101/api/v1/agents/my_agent/start
```

## Health Checks

All services provide health endpoints:

```bash
curl http://localhost:8100/healthcheck  # AtomSpace
curl http://localhost:8101/healthcheck  # CogServer
curl http://localhost:8102/healthcheck  # PLN
```

## Monitoring

### Kubernetes

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/component=opencog-atomspace
kubectl get pods -l app.kubernetes.io/component=opencog-cogserver
kubectl get pods -l app.kubernetes.io/component=opencog-pln

# View logs
kubectl logs -l app.kubernetes.io/component=opencog-atomspace
kubectl logs -l app.kubernetes.io/component=opencog-cogserver
kubectl logs -l app.kubernetes.io/component=opencog-pln
```

### Service Status

```bash
# Get service information
kubectl get svc | grep opencog

# Describe services
kubectl describe svc jan-server-opencog-atomspace
kubectl describe svc jan-server-opencog-cogserver
kubectl describe svc jan-server-opencog-pln
```

## Troubleshooting

### Service Won't Start

1. Check pod logs:
   ```bash
   kubectl logs <pod-name>
   ```

2. Check pod events:
   ```bash
   kubectl describe pod <pod-name>
   ```

3. Verify image is available:
   ```bash
   kubectl get pods -o jsonpath='{.items[*].spec.containers[*].image}' | grep opencog
   ```

### Service Unreachable

1. Check service endpoints:
   ```bash
   kubectl get endpoints | grep opencog
   ```

2. Test connectivity from another pod:
   ```bash
   kubectl run test --rm -it --image=busybox -- wget -O- http://jan-server-opencog-atomspace:8100/healthcheck
   ```

### Performance Issues

1. Check resource usage:
   ```bash
   kubectl top pods | grep opencog
   ```

2. Scale services:
   ```bash
   kubectl scale deployment jan-server-opencog-atomspace --replicas=3
   ```

## Documentation

- **Detailed Examples**: See [docs/OPENCOG_EXAMPLES.md](./OPENCOG_EXAMPLES.md)
- **AtomSpace README**: [apps/opencog-atomspace/README.md](../apps/opencog-atomspace/README.md)
- **CogServer README**: [apps/opencog-cogserver/README.md](../apps/opencog-cogserver/README.md)
- **PLN README**: [apps/opencog-pln/README.md](../apps/opencog-pln/README.md)

## Architecture

The OpenCog services integrate into Jan Server's microservice architecture:

```
Jan API Gateway (Port 8080)
    ├── OpenCog AtomSpace (Port 8100) - Knowledge storage
    ├── OpenCog CogServer (Port 8101) - Agent scheduler
    │   └── Uses AtomSpace for knowledge access
    └── OpenCog PLN (Port 8102) - Reasoning engine
        └── Uses AtomSpace for knowledge queries
```

## Security

All OpenCog services:
- Run with minimal privileges
- Use health checks for liveness/readiness
- Support resource limits and requests
- Implement proper error handling
- Use structured logging

## Performance

Default resource allocations:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi
```

Adjust based on your workload requirements.

## Further Reading

- [OpenCog Framework Documentation](https://wiki.opencog.org/)
- [AtomSpace Design](https://wiki.opencog.org/w/AtomSpace)
- [PLN Documentation](https://wiki.opencog.org/w/PLN)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
