package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/smtp"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type K8sTools struct {
	client kubernetes.Interface
}

func NewK8sTools(client kubernetes.Interface) *K8sTools {
	return &K8sTools{client: client}
}

// GetPodLogs fetches the logs of a pod.
func (t *K8sTools) GetPodLogs(namespace, name string) (string, error) {
	req := t.client.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{
		TailLines: getInt64(50), // Last 50 lines
	})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("error in opening stream: %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf: %v", err)
	}
	return buf.String(), nil
}

// GetDeploymentLogs finds pods for a deployment and fetches logs.
func (t *K8sTools) GetDeploymentLogs(namespace, deploymentName string) (string, error) {
	// 1. Get Deployment to find selector
	deploy, err := t.client.AppsV1().Deployments(namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// 2. List Pods
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return "", err
	}
	pods, err := t.client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "No pods found for deployment", nil
	}

	return t.GetPodLogs(namespace, pods.Items[0].Name)
}
func (t *K8sTools) GetJobLogs(namespace, jobName string) (string, error) {
	job, err := t.client.BatchV1().Jobs(namespace).Get(context.Background(), jobName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	selector, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return "", err
	}
	pods, err := t.client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "No pods found for job", nil
	}

	return t.GetPodLogs(namespace, pods.Items[0].Name)
}

// GetSecret fetches a secret and returns its Data map (redacted manually if needed, but agent might need it).
func (t *K8sTools) GetSecret(namespace, name string) (string, error) {
	secret, err := t.client.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Convert data to string for LLM
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Secret: %s/%s\n", namespace, name))
	for k, v := range secret.Data {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, string(v)))
	}
	return sb.String(), nil
}

// ApplyPatch applies a JSON patch to a resource.
// Simplification: We usually patch deployments or pods.
func (t *K8sTools) ApplyPatch(namespace string, kind string, name string, patch string) (string, error) {
	ctx := context.Background()
	patchBytes := []byte(patch)

	switch strings.ToLower(kind) {
	case "deployment":
		_, err := t.client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return "", err
		}
	case "pod":
		_, err := t.client.CoreV1().Pods(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported kind for patch: %s", kind)
	}
	return "Patch applied successfully", nil
}

// SendEmail sends an email to the configured engineer.
func (t *K8sTools) SendEmail(subject, body string) error {
	host := os.Getenv("EMAIL_SMTP_HOST")
	port := os.Getenv("EMAIL_SMTP_PORT")
	user := os.Getenv("EMAIL_USER")
	password := os.Getenv("EMAIL_PASSWORD")
	to := os.Getenv("ENGINEER_EMAIL")

	if host == "" || to == "" {
		return fmt.Errorf("email configuration missing")
	}

	auth := smtp.PlainAuth("", user, password, host)
	addr := fmt.Sprintf("%s:%s", host, port)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))

	return smtp.SendMail(addr, auth, user, []string{to}, msg)
}

func getInt64(i int64) *int64 {
	return &i
}
