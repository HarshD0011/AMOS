package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/HarshD0011/AMOS/AMOS/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// K8sTools provides functions for the ADK agent to interact with Kubernetes
type K8sTools struct {
	client *k8s.Client
}

// NewK8sTools creates a new toolbox
func NewK8sTools(client *k8s.Client) *K8sTools {
	return &K8sTools{client: client}
}

// GetPodLogs retrieves the logs for a specific pod
func (t *K8sTools) GetPodLogs(podName, namespace string, tailLines int64) (string, error) {
	req := t.client.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		TailLines: &tailLines,
	})

	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("error in opening stream: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf: %w", err)
	}

	return buf.String(), nil
}

// DescribeResource gets the YAML/JSON representation of a resource
// For simplicity, we'll return basic info or stringified struct
func (t *K8sTools) DescribeResource(kind, name, namespace string) (string, error) {
	switch kind {
	case "Pod":
		pod, err := t.client.Clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Pod Status: %s, Message: %s, Reason: %s", pod.Status.Phase, pod.Status.Message, pod.Status.Reason), nil
	case "Deployment":
		deploy, err := t.client.Clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Replicas: %d/%d, Available: %d, Conditions: %+v", deploy.Status.Replicas, *deploy.Spec.Replicas, deploy.Status.AvailableReplicas, deploy.Status.Conditions), nil
	case "Job":
		job, err := t.client.Clientset.BatchV1().Jobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Active: %d, Succeeded: %d, Failed: %d, Conditions: %+v", job.Status.Active, job.Status.Succeeded, job.Status.Failed, job.Status.Conditions), nil
	default:
		return "", fmt.Errorf("unsupported kind: %s", kind)
	}
}

// PatchDeployment applies a patch to a deployment
// This is somewhat generic, agent needs to provide valid merge patch or json patch
func (t *K8sTools) PatchDeployment(name, namespace, patchType, patchData string) (string, error) {
	pt := types.StrategicMergePatchType
	if patchType == "json" {
		pt = types.JSONPatchType
	} else if patchType == "merge" {
		pt = types.MergePatchType
	}

	_, err := t.client.Clientset.AppsV1().Deployments(namespace).Patch(context.Background(), name, pt, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to patch deployment: %w", err)
	}
	return "Deployment patched successfully", nil
}

// ScaleDeployment scales a deployment to n replicas
func (t *K8sTools) ScaleDeployment(name, namespace string, replicas int32) (string, error) {
	scale, err := t.client.Clientset.AppsV1().Deployments(namespace).GetScale(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	scale.Spec.Replicas = replicas
	_, err = t.client.Clientset.AppsV1().Deployments(namespace).UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to scale deployment: %w", err)
	}
	return fmt.Sprintf("Deployment scaled to %d replicas", replicas), nil
}

// DeletePod deletes a pod (forcing restart if controllable)
func (t *K8sTools) DeletePod(name, namespace string) (string, error) {
	err := t.client.Clientset.CoreV1().Pods(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to delete pod: %w", err)
	}
	return "Pod deleted successfully", nil
}

// RollbackDeployment rolls back to previous version
// NOTE: K8s client-go doesn't have a direct "rollback" method like kubectl rollout undo.
// We have to iterate revisions or patch to previous RS. 
// For MVP, simplistic approach: We won't fully implement robust history rollback here without complex logic.
// We'll rely on our internal "State Snapshot" rollback which is safer.
// But for the tool itself, maybe we can expose a "undo" if we want the agent to try it.
func (t *K8sTools) RollbackDeployment(name, namespace string) (string, error) {
	// Implementation of actual kubectl rollout undo is complex via API (getting ControllerRevision etc)
	// For now, let's return a "Not Implemented" or use the internal rollback mechanism
	return "", fmt.Errorf("native rollback tool not fully implemented, use internal rollback service")
}
