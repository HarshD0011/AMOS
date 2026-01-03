package k8s

import (
	"context"
	"fmt"
	"log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// DeploymentFault represents a detected deployment issue
type DeploymentFault struct {
	Name      string
	Namespace string
	Reason    string
	Message   string
	Timestamp time.Time
}

// DeploymentMonitor watches for deployment faults
type DeploymentMonitor struct {
	client     *Client
	faultChan  chan<- DeploymentFault
	stopChan   chan struct{}
}

// NewDeploymentMonitor creates a new DeploymentMonitor
func NewDeploymentMonitor(client *Client, faultChan chan<- DeploymentFault) *DeploymentMonitor {
	return &DeploymentMonitor{
		client:    client,
		faultChan: faultChan,
		stopChan:  make(chan struct{}),
	}
}

// Start begins watching for deployment events
func (dm *DeploymentMonitor) Start(ctx context.Context) {
	log.Println("Starting Deployment Monitor...")

	watchNamespace := func(ns string) {
		rw := &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return dm.client.Clientset.AppsV1().Deployments(ns).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return dm.client.Clientset.AppsV1().Deployments(ns).Watch(context.Background(), options)
			},
		}

		_, controller := cache.NewInformer(
			rw,
			&appsv1.Deployment{},
			1*time.Minute,
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(oldObj, newObj interface{}) {
					deployment := newObj.(*appsv1.Deployment)
					dm.checkDeploymentStatus(deployment)
				},
				AddFunc: func(obj interface{}) {
					deployment := obj.(*appsv1.Deployment)
					dm.checkDeploymentStatus(deployment)
				},
			},
		)

		go controller.Run(dm.stopChan)
	}

	if len(dm.client.Namespaces) == 0 {
		watchNamespace(corev1.NamespaceAll)
	} else {
		for _, ns := range dm.client.Namespaces {
			watchNamespace(ns)
		}
	}

	<-ctx.Done()
	close(dm.stopChan)
	log.Println("Deployment Monitor stopped.")
}

func (dm *DeploymentMonitor) checkDeploymentStatus(deploy *appsv1.Deployment) {
	// 1. Check Conditions
	for _, condition := range deploy.Status.Conditions {
		// Available = False
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionFalse {
			dm.emitFault(deploy, "DeploymentUnavailable", fmt.Sprintf("Deployment available: False. Reason: %s - %s", condition.Reason, condition.Message))
			return
		}
		
		// Progressing = False (Rollout stuck)
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse {
			dm.emitFault(deploy, "DeploymentStuck", fmt.Sprintf("Deployment progressing: False. Reason: %s - %s", condition.Reason, condition.Message))
			return
		}
	}

	// 2. Check Replica Mismatch (Desired vs Available)
	// Only flag if it's been mismatched for a while? For now, we rely on Conditions mostly, 
	// but if we have 0 available for a long time vs >0 spec, it's bad.
	if deploy.Spec.Replicas != nil && *deploy.Spec.Replicas > 0 {
		if deploy.Status.AvailableReplicas == 0 && deploy.Status.Replicas > 0 {
			// This might overlap with DeploymentAvailable=False, but good to catch
			// Avoid double emitting if condition caught it.
			// Let's rely on Condition checks above as primary.
		}
	}
}

func (dm *DeploymentMonitor) emitFault(deploy *appsv1.Deployment, reason, message string) {
	log.Printf("Detected fault in deployment %s/%s: %s", deploy.Namespace, deploy.Name, reason)
	
	dm.faultChan <- DeploymentFault{
		Name:      deploy.Name,
		Namespace: deploy.Namespace,
		Reason:    reason,
		Message:   message,
		Timestamp: time.Now(),
	}
}
