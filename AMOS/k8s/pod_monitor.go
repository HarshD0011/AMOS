package k8s

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// PodFault represents a detected pod issue
type PodFault struct {
	PodName   string
	Namespace string
	Reason    string
	Message   string
	Timestamp time.Time
}

// PodMonitor watches for pod faults
type PodMonitor struct {
	client     *Client
	faultChan  chan<- PodFault
	stopChan   chan struct{}
}

// NewPodMonitor creates a new PodMonitor
func NewPodMonitor(client *Client, faultChan chan<- PodFault) *PodMonitor {
	return &PodMonitor{
		client:    client,
		faultChan: faultChan,
		stopChan:  make(chan struct{}),
	}
}

// Start begins watching for pod events
func (pm *PodMonitor) Start(ctx context.Context) {
	log.Println("Starting Pod Monitor...")
	
	// If specific namespaces are configured, we'd loop through them.
	// For simplicity in this implementation, we'll watch all namespaces if list is empty
	// or just the first one if provided, handling multiple involves multiple informers or a slightly different approach.
	// A better production approach uses SharedInformerFactory. 
	// Here we use a simple watcher for demonstration/MVP.

	// Helper to watch a single namespace (or all)
	watchNamespace := func(ns string) {
		rw := &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return pm.client.Clientset.CoreV1().Pods(ns).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return pm.client.Clientset.CoreV1().Pods(ns).Watch(context.Background(), options)
			},
		}

		_, controller := cache.NewInformer(
			rw,
			&corev1.Pod{},
			1*time.Minute, // Resync
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(oldObj, newObj interface{}) {
					pod := newObj.(*corev1.Pod)
					pm.checkPodStatus(pod)
				},
				AddFunc: func(obj interface{}) {
					pod := obj.(*corev1.Pod)
					pm.checkPodStatus(pod)
				},
			},
		)

		go controller.Run(pm.stopChan)
	}

	if len(pm.client.Namespaces) == 0 {
		watchNamespace(corev1.NamespaceAll)
	} else {
		for _, ns := range pm.client.Namespaces {
			watchNamespace(ns)
		}
	}
	
	<-ctx.Done()
	close(pm.stopChan)
	log.Println("Pod Monitor stopped.")
}

func (pm *PodMonitor) checkPodStatus(pod *corev1.Pod) {
	// Skip if pod is Succeeded (completed job)
	if pod.Status.Phase == corev1.PodSucceeded {
		return
	}

	// Check for CrashLoopBackOff or ImagePullBackOff or other Container errors
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			reason := containerStatus.State.Waiting.Reason
			switch reason {
			case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError":
				pm.emitFault(pod, reason, containerStatus.State.Waiting.Message)
				return // Emit once per pod check to avoid noise
			}
		}
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
			// Terminated with error
			// OOMKilled is a common reason
			if containerStatus.State.Terminated.Reason == "OOMKilled" {
				pm.emitFault(pod, "OOMKilled", fmt.Sprintf("Exit Code: %d", containerStatus.State.Terminated.ExitCode))
				return
			}
			// General error
			if containerStatus.State.Terminated.Reason == "Error" {
				pm.emitFault(pod, "ContainerError", fmt.Sprintf("Exit Code: %d", containerStatus.State.Terminated.ExitCode))
				return 
			}
		}
	}

	// Check for Pod phase Failed
	if pod.Status.Phase == corev1.PodFailed {
		pm.emitFault(pod, "PodFailed", pod.Status.Message)
	}
}

func (pm *PodMonitor) emitFault(pod *corev1.Pod, reason, message string) {
	// Simple deduplication could happen here or in downstream FaultDetector
	// For now, emit everything
	log.Printf("Detected fault in pod %s/%s: %s", pod.Namespace, pod.Name, reason)
	
	pm.faultChan <- PodFault{
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		Reason:    reason,
		Message:   message,
		Timestamp: time.Now(),
	}
}
