# Xtream 🎬

Xtream is a scalable video streaming platform developed for the KOM (Software Engineering) course. It's designed to showcase a modern, cloud-native architecture using Kubernetes, with a focus on asynchronous processing and self-contained systems.

## 👥 Team Members

  * **Andrian Danar Perdana** (23/513040/PA/21917)
  * **Andreandhiki Riyanta Putra** (23/517511/PA/22191)
  * **Muhammad Argya Vityasy** (23/522547/PA/22475) – Kubernetes
  * **Nasya Putri Raudhah Dahlan** (23/513931/PA/21967)

-----

## 🚀 Features

  * **Video Upload & Streaming**: Core services for handling video content.
  * **Kubernetes-Native**: Deployed with Kubernetes for scalability and resilience.
  * **DaemonSet Monitoring**: A cluster-wide monitoring agent is deployed on each node.
  * **Local Development**: Easy setup with a single-node Minikube cluster.
  * **Stress Tested**: Performance validated using k6.

-----

## 📦 Tech Stack

### Current Stack

  * **Backend**: Go (Gin Framework)
  * **Containerization**: Docker
  * **Orchestration**: Kubernetes (Minikube for local setup)
  * **Ingress/Routing**: NGINX Ingress or Traefik
  * **Load Testing**: k6

### Planned Enhancements

  * **Message Broker**: Apache Kafka (for asynchronous transcoding)
  * **Object Storage**: MinIO (as a self-hosted alternative to AWS S3)

-----

## 🏗️ Architecture & Design Notes

### Local Performance vs. Production

During local development, this project uses `minikube tunnel` to expose the service.

**Important Note**: The `minikube tunnel` creates a network route from your local machine to the cluster, which introduces a **significant performance bottleneck**. This is expected and is a limitation of the local development environment. When deployed to a production-grade Kubernetes engine (like AWS EKS, Google GKE, or Azure AKS), the traffic is handled by a high-performance cloud load balancer, and this bottleneck does not exist.

### Monitoring with a DaemonSet

A `DaemonSet` has been implemented to deploy a monitoring agent pod on every node in the cluster. This is intended for collecting node-level metrics and logs.

**Current Status & Suggestions**:
The connection to the monitoring endpoint is currently under development. If you are facing issues connecting to the service, here are some common areas to investigate:

  * **Service & Endpoints**: Ensure the Kubernetes `Service` correctly targets the `DaemonSet` pods using the right labels and ports.
  * **RBAC Permissions**: The `DaemonSet`'s Service Account might need specific `ClusterRole` and `ClusterRoleBinding` permissions to access cluster metrics.
  * **Network Policies**: Check if any `NetworkPolicy` resources are blocking traffic to or from the `DaemonSet` pods.
  * **Pod Logs**: Check the logs of the `DaemonSet` pods for any startup errors: `kubectl logs -l name=your-daemonset-label`

-----

## 💡 Key Improvements & Future Work

The following are planned improvements to make the platform more robust and scalable.

### Asynchronous Processing with Kafka

To improve user experience and system resilience, the video upload and transcoding process will be decoupled using **Apache Kafka**.

  * **Current Flow**: A user uploads a video, and the API handles transcoding directly.
  * **Planned Flow**: The API will receive the video and publish a "new video" message to a Kafka topic. A separate transcoding service will consume this message, process the video, and update its status independently.
  * **Benefit**: This makes the upload API faster and more reliable. Even if the transcoding service is slow or fails, it won't affect new video uploads.

### Self-Hosted Storage with MinIO

To avoid dependency on third-party cloud services like AWS S3, **MinIO** will be implemented for object storage.

  * **Benefit**: MinIO is an S3-compatible, high-performance object storage server that can be hosted within our Kubernetes cluster. This makes the entire platform more self-contained, reduces external dependencies, and gives us full control over our storage infrastructure.

-----

## 🧰 Local Development Setup

Follow these steps to get the project running on a local Minikube cluster.

### 1\. Install Prerequisites

  * [Docker](https://docs.docker.com/get-docker/)
  * [kubectl](https://kubernetes.io/docs/tasks/tools/)
  * [Minikube](https://minikube.sigs.k8s.io/docs/start/)

### 2\. Start Your Minikube Cluster

```bash
minikube start --driver=docker
```

### 3\. Enable the Ingress Addon

```bash
minikube addons enable ingress
```
### 3.5.1 Make sure you have these in your /etc/hosts
(your minikube ip) server.video.localhost minio-api.video.localhost minio-console.video.localhost

### 3.5.2 Run dev.sh
```bash
cd upload-service && chmod +x dev.sh && ./dev.sh
```

### 4\. Route Traffic with Minikube Tunnel

For easy DNS resolution, open a **new, separate terminal window** and run:

```bash
minikube tunnel
```

**Leave this command running.** It manages access to your services.

### 5\. Deploy the Application (Run Twice If Can Not)

```bash
kubectl apply -f upload-service/k8s/
```

### 6\. Verify the Deployment

```bash
kubectl get pods
```

You should see the `upload-service` pod with a `Running` status.

### 7\. Access the Application

Access the app in your browser at: [http://server.video.localhost](https://www.google.com/search?q=http://server.video.localhost)

-----

## 📂 Project Structure

```
Xtream/
├── upload-service/
│   ├── k8s/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml
│   ├── main.go
│   ├── Dockerfile
│   └── ...
├── scripts/
│   └── stress_test.js
├── dev.sh
└── README.md
```

-----

## 💥 Stress Testing with k6

This project uses [k6](https://k6.io/) to simulate load and test the performance of the services.

### Prerequisites

  * Make sure k6 is installed. You can find instructions [here](https://k6.io/docs/getting-started/installation/).
  * The application must be deployed and running in your Minikube cluster.

### Running the Test

To start the stress test, run the following command from the project's root directory:

```bash
./dev.sh k6
```

This script executes the `scripts/stress_test.js` file, which sends requests to the `GET /health` and `GETideos` endpoints.

### Customizing the Test

You can easily modify the test parameters by editing the `scripts/stress_test.js` file. Adjust the `options` to change the number of virtual users (VUs), test duration, and more.
we can handle -+ 1000 VUs (bottlenecked because of minikube tunnel)
```javascript
export let options = {
  vus: 50,
  duration: '30s',
};
```

-----

## 📃 License

This project is licensed under the MIT License.
