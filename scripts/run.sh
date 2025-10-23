#!/bin/bash
set -e
minikube start
eval $(minikube docker-env)

# Build Jan API Gateway
docker build -t menloltd/jan-server:latest ./apps/jan-api-gateway

# Build OpenCog services
docker build -t menloltd/opencog-atomspace:latest ./apps/opencog-atomspace
docker build -t menloltd/opencog-cogserver:latest ./apps/opencog-cogserver
docker build -t menloltd/opencog-pln:latest ./apps/opencog-pln

# Deploy with Helm
helm dependency update ./charts/jan-server
helm install jan-server ./charts/jan-server --set gateway.image.tag=latest

# Port forward services
echo "Waiting for services to be ready..."
sleep 10
kubectl port-forward svc/jan-server-jan-api-gateway 8080:8080 &
kubectl port-forward svc/jan-server-opencog-atomspace 8100:8100 &
kubectl port-forward svc/jan-server-opencog-cogserver 8101:8101 &
kubectl port-forward svc/jan-server-opencog-pln 8102:8102 &

echo ""
echo "Services are being deployed and port-forwarded:"
echo "  - Jan API Gateway: http://localhost:8080"
echo "  - Swagger UI: http://localhost:8080/api/swagger/index.html"
echo "  - OpenCog AtomSpace: http://localhost:8100"
echo "  - OpenCog CogServer: http://localhost:8101"
echo "  - OpenCog PLN: http://localhost:8102"
echo ""
echo "To stop port forwarding, press Ctrl+C"
echo "To uninstall: helm uninstall jan-server"

# Wait for all port-forwards
wait