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

kubectl apply -f k8s/
kubectl rollout restart deployment video-server -n $NAMESPACE
echo "Restarting PostgreSQL deployment..."
kubectl rollout restart deployment postgres -n $NAMESPACE
echo "Restarting MinIO deployment..."
kubectl rollout restart deployment minio -n $NAMESPACE

# Function to run k6 stress tests
run_k6_tests() {
  echo "Running k6 stress tests..."
  # Ensure Minikube tunnel is running for LoadBalancer access if needed, or use NodePort/port-forward
  # For simplicity, this example assumes video-server is accessible via video.server.localhost
  # You might need to run `minikube tunnel` in a separate terminal
  # or adjust the k6 script to target the service via `kubectl port-forward`

  # Check if k6 is installed
  if ! command -v k6 &> /dev/null
  then
      echo "k6 could not be found. Please install k6 first."
      echo "See: https://k6.io/docs/getting-started/installation/"
      return
  fi

  # Check if /etc/hosts entry exists for server.video.localhost
  if ! grep -q "server.video.localhost" /etc/hosts; then
    echo "Host 'server.video.localhost' not found in /etc/hosts."
    echo "Please add the Minikube IP to your /etc/hosts file for server.video.localhost."
    echo "Example: $(minikube ip) server.video.localhost"
    echo "Alternatively, run 'minikube tunnel' in a separate terminal and ensure your Ingress is working."
    # return # Uncomment this if you want to stop the script if host is not found
  fi
  
  if ! grep -q "minio-api.video.localhost" /etc/hosts; then
    echo "Host 'minio-api.video.localhost' not found in /etc/hosts."
    echo "Please add the Minikube IP to your /etc/hosts file for minio-api.video.localhost."
    echo "Example: $(minikube ip) minio-api.video.localhost"
    echo "Alternatively, run 'minikube tunnel' in a separate terminal and ensure your Ingress is working."
    # return # Uncomment this if you want to stop the script if host is not found
  fi

  if ! grep -q "minio-console.video.localhost" /etc/hosts; then
    echo "Host 'minio-console.video.localhost' not found in /etc/hosts."
    echo "Please add the Minikube IP to your /etc/hosts file for minio-console.video.localhost."
    echo "Example: $(minikube ip) minio-console.video.localhost"
    echo "Alternatively, run 'minikube tunnel' in a separate terminal and ensure your Ingress is working."
    # return # Uncomment this if you want to stop the script if host is not found
  fi
  # Run k6 script
  # The k6 script needs a video file to upload.
  
  # Run the k6 script, passing the Minikube IP if needed or relying on /etc/hosts
  # K6_PROMETHEUS_RW_SERVER_URL=http://localhost:9090/api/v1/write k6 run ./scripts/stress_test.js
  # The above line is for Prometheus remote write, if you have it set up.
  # For now, let's run without it.
  k6 run ./scripts/stress_test.js

  echo "k6 stress tests finished."
}

# Follow the logs
echo "Following logs..."
# Check for arguments
if [ "$1" == "k6" ]; then
  run_k6_tests
else
  kubectl logs -f -l app=video-server -n $NAMESPACE
fi