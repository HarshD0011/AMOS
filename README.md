# AMOS: Agentic Mesh Observability System

AMOS is an intelligent Kubernetes operator that monitors Pods, Deployments, and Jobs. When a failure is detected (e.g., `CrashLoopBackOff`, `ImagePullBackOff`, `Failed`), it triggers an AI agent (powered by Google Gemini) to diagnose the issue and notify an engineer via email.

## Prerequisites

- **Docker Desktop** (running)
- **Kind** (Kubernetes in Docker)
- **kubectl**
- **Go** (1.24+ for local development)

## Configuration

The application requires the following credentials.

1.  **Google Gemini API Key**: [Get one here](https://aistudio.google.com/).
2.  **Gmail App Password**: [Generate one here](https://myaccount.google.com/apppasswords) (if using Gmail SMTP).

## Replication Steps (Local)

### 1. Create Cluster

If you don't have a cluster running:

```bash
kind create cluster --name amos-cluster
```

### 2. Configure Environment

Edit `deploy/bootstrap.yaml` and update the environment variables in the Deployment section:

```yaml
env:
  - name: GOOGLE_API_KEY
    value: "YOUR_GEMINI_API_KEY"
  - name: EMAIL_SMTP_HOST
    value: "smtp.gmail.com"
  - name: EMAIL_SMTP_PORT
    value: "587"
  - name: EMAIL_USER
    value: "your-email@gmail.com"
  - name: EMAIL_PASSWORD
    value: "your-app-password"
  - name: ENGINEER_EMAIL
    value: "recipient-email@example.com"
```

### 3. Build Docker Image

Build the image with the tag expected by the deployment manifest:

```bash
docker build -t amos:v1 .
```

### 4. Load Image into Kind

This step is critical so Kind can access the local image:

```bash
kind load docker-image amos:v1
```

### 5. Deploy Operator

Apply the bootstrap manifest, which creates the Namespace, ServiceAccount, RBAC roles, and Deployment:

```bash
kubectl apply -f deploy/bootstrap.yaml
```

**Verify successful startup:**

```bash
kubectl get pods -n amos
kubectl logs -n amos -l app=amos-operator -f
```

_You should see "Starting InformerDeployment..." and "All informers started."_

### 6. Verify Functionality (Trigger a Fault)

Create a Pod designed to fail immediately:

```bash
kubectl run test-fail --image=busybox --restart=Never -- /bin/false
```

**Observe Behavior:**

1.  **Log Output**: The operator logs will show `Processing pod...`, `Diagnosing...`, and `Diagnosis: ...`.
2.  **Email**: The recipient email will receive a diagnosis report.

## Cleanup

To remove AMOS and the test pod:

```bash
kubectl delete -f deploy/bootstrap.yaml
kubectl delete pod test-fail
```
