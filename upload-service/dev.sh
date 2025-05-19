````bash
#!/bin/bash
# filepath: /Users/miapalovaara/Xtream/upload-service/dev.sh

# Set variables
NAMESPACE="video-app"
IMAGE_NAME="video-server:latest"

# Build the Docker image
echo "Building Docker image..."
eval $(minikube docker-env)
docker build -t $IMAGE_NAME .

# Apply Kubernetes manifests
echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/deployment.yaml
kubectl rollout restart deployment video-server -n $NAMESPACE

# Follow the logs
echo "Following logs..."
kubectl logs -f -l app=video-server -n $NAMESPACE