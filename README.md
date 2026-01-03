# AMOS: Autonomous Multi-agent Orchestration Service

AMOS is an intelligent Site Reliability Engineering (SRE) agent that monitors your Kubernetes cluster, detects faults, and proactively attempts to fix them using Google's Agent Development Kit (ADK) and Gemini models.

## Features

- **Autonomous Monitoring**: Real-time monitoring of Pods, Deployments, and Jobs.
- **Intelligent Remediation**: Uses Gemini 2.0 Flash to analyze error logs and K8s events to determining root causes and applying fixes.
- **Safety First**:
  - **State Snapshotting**: Captures resource state before attempting fixes.
  - **Retry Limits**: Maximum of 2 remediation attempts per fault.
  - **Auto-Rollback**: Automatically reverts changes if remediation fails.
- **Notifications**: Alerts engineers via Email with detailed reports of actions taken or escalation requests.

## Architecture

AMOS runs as a standalone service (either in-cluster or external) and communicates with the Kubernetes API server.

1. **Monitors** watch for specific failure patterns (CrashLoopBackOff, Deployment stuck, Job failed).
2. **Fault Detector** aggregates and deduplicates alerts.
3. **Orchestrator** coordinates the response:
   - Checks retry limits.
   - Snapshots the current state.
   - Generates a context prompt with logs and specs.
   - Invokes the **ADK Remediation Agent**.
   - Applies the fix (Patch, Scale, Delete Pod).
   - Verifies or Escalates.

## Prerequisites

- Go 1.24+
- Kubernetes Cluster (GKE, Minikube, Kind, etc.)
- Google Cloud Project with Vertex AI API enabled (or valid API Key for Gemini).
- SMTP Server for notifications (Gmail, SendGrid, MailHog for dev).

## Configuration

AMOS is configured via `config.yaml` or Environment Variables.

### Environment Variables

| Variable              | Description                             |
| --------------------- | --------------------------------------- |
| `KUBECONFIG`          | Path to kubeconfig (if running locally) |
| `GOOGLE_API_KEY`      | **Required**. API Key for Gemini.       |
| `AMOS_SMTP_HOST`      | SMTP Host (e.g. smtp.gmail.com)         |
| `AMOS_SMTP_PORT`      | SMTP Port (e.g. 587)                    |
| `AMOS_SMTP_USERNAME`  | SMTP Username                           |
| `AMOS_SMTP_PASSWORD`  | SMTP Password                           |
| `AMOS_ENGINEER_EMAIL` | Email to notify on alerts               |
| `AMOS_FROM_EMAIL`     | Sender email address                    |

## Installation & Running

### 1. Build

```bash
go mod tidy
go build -o amos AMOS/main.go
```

### 2. Run Locally

```bash
export KUBECONFIG=~/.kube/config
export GOOGLE_API_KEY="your-api-key"
export AMOS_ENGINEER_EMAIL="admin@example.com"
# ... other env vars

./amos
```

### 3. Run in Kubernetes

Build a Docker image and deploy using the provided manifests (not included in MVP, but standard Deployment/ServiceAccount setup).

## Security Note

AMOS requires broad permissions (`get`, `list`, `watch`, `update`, `patch`, `delete`) on `Pods`, `Deployments`, and `Jobs` in the monitored namespaces. Ensure you grant these permissions via RBAC responsibly.
