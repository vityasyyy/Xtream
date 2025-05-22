
---
# Xtream

**Scalable Video Streaming Platform**

Xtream is a scalable video streaming platform developed as part of the KOM (Software Engineering) course. This project is designed to demonstrate scalable service deployment using Kubernetes and Minikube, with stress testing facilitated through k6.

## 👥 Anggota Kelompok

1. **Andrian Danar Perdana** (23/513040/PA/21917)  
2. **Andreandhiki Riyanta Putra** (23/517511/PA/22191)
3. **Muhammad Argya Vityasy** (23/522547/PA/22475) – Kubernetes  
4. **Nasya Putri Raudhah Dahlan** (23/513931/PA/21967)

---

## 🚀 Features

- Video upload and streaming service
- Kubernetes-based deployment
- Local single-node cluster setup using Minikube
- Stress testing using [k6](https://k6.io/)

---

## 📦 Tech Stack

- Go (Gin Framework)
- Kubernetes (Minikube)
- Docker
- k6 (for load/stress testing)
- NGINX Ingress / Traefik (for local routing)

---

## 🧰 Local Development with Minikube

Follow these steps to set up a **local Kubernetes single-node cluster** using Minikube.

### 1. Install Dependencies

- [Docker](https://docs.docker.com/get-docker/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Minikube](https://minikube.sigs.k8s.io/docs/start/)

### 2. Start Minikube

```bash
minikube start --driver=docker
````

### 3. Enable Ingress Addon

```bash
minikube addons enable ingress
```

### 4. Set Up Local DNS

Map your domain `server.video.localhost` to the Minikube IP.

#### Option A: Automatic (if using `minikube tunnel`)

```bash
sudo minikube tunnel
```

#### Option B: Manual `/etc/hosts` entry

1. Get the Minikube IP:

   ```bash
   minikube ip
   ```

2. Add this line to `/etc/hosts`:

   ```
   <minikube_ip> server.video.localhost
   ```

---

## 📂 Project Structure

```
xRabbit/
├── upload-service/
│   ├── main.go
│   ├── Dockerfile
│   ├── ...
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
├── scripts/
│   ├── stress_test.js
├── dev.sh
├── README.md
```

---

## 💥 Stress Testing with k6

This project includes a stress testing setup using [k6](https://k6.io/).

### 🔧 Prerequisites

* Install k6: [https://k6.io/docs/getting-started/installation/](https://k6.io/docs/getting-started/installation/)
* Ensure the app is running in Minikube (see above).

### ▶️ Running Stress Tests

From the root or `upload-service/` directory, run:

```bash
./dev.sh k6
```

This script will:

1. Run the stress test located at `scripts/stress_test.js`
2. Simulate load on the following endpoints:

   * `GET /health`
   * `POST /videos`

### 🔁 Modifying Test Parameters

Edit the file `scripts/stress_test.js` to change:

* Duration
* Number of virtual users (VUs)
* Endpoints tested
* Payloads sent

Example snippet inside `stress_test.js`:

```js
export let options = {
  vus: 50,
  duration: '30s',
};
```

---

## 🛠 Deployment on Minikube

Apply Kubernetes manifests:

```bash
kubectl apply -f k8s/
```

Check if pods are running:

```bash
kubectl get pods
```

Access the app via:
[http://server.video.localhost](http://server.video.localhost)

---

## 📃 License

MIT License.

---
