package services

import (
	"context"
	"fmt"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/pkg/state"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// this is for monitoring the deployment of the k8s cluster

type DeploymentMonitor struct {
	informer     cache.SharedIndexInformer
	queue        workqueue.RateLimitingInterface
	resolver     *agent.Resolver
	stateManager *state.StateManager
}

func NewDeploymentMonitor(client kubernetes.Interface, resolver *agent.Resolver, sm *state.StateManager) *DeploymentMonitor {
	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Apps().V1().Deployments().Informer()

	c := &DeploymentMonitor{
		informer:     informer,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deployment-monitor"),
		resolver:     resolver,
		stateManager: sm,
	}

	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				deployment := obj.(*appsv1.Deployment)
				// Check for failure conditions (Replica mismatch)
				if deployment.Status.Replicas != deployment.Status.ReadyReplicas {
					// Report Failure (Red)
					if c.stateManager != nil {
						msg := fmt.Sprintf("Replica Mismatch: %d/%d", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
						c.stateManager.ReportFailure(deployment.Namespace, "Deployment", deployment.Name, msg)
					}
					key, err := cache.MetaNamespaceKeyFunc(deployment)
					if err == nil {
						c.queue.Add(key)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newDeployment := newObj.(*appsv1.Deployment)
				if newDeployment.Status.Replicas != newDeployment.Status.ReadyReplicas {
					// Report Failure (Red)
					if c.stateManager != nil {
						msg := fmt.Sprintf("Replica Mismatch: %d/%d", newDeployment.Status.ReadyReplicas, newDeployment.Status.Replicas)
						c.stateManager.ReportFailure(newDeployment.Namespace, "Deployment", newDeployment.Name, msg)
					}
					key, err := cache.MetaNamespaceKeyFunc(newDeployment)
					if err == nil {
						c.queue.Add(key)
					}
				} else {
					// Resolve Issue (Green)
					if c.stateManager != nil {
						c.stateManager.Resolve(newDeployment.Namespace, "Deployment", newDeployment.Name)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				// Resolve on delete
				deployment, ok := obj.(*appsv1.Deployment)
				if !ok {
					tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
					if !ok {
						return
					}
					deployment, ok = tombstone.Obj.(*appsv1.Deployment)
					if !ok {
						return
					}
				}
				if c.stateManager != nil {
					c.stateManager.Resolve(deployment.Namespace, "Deployment", deployment.Name)
				}
			},
		},
	)

	return c
}

func (d *DeploymentMonitor) Run(ctx context.Context) {
	defer d.queue.ShutDown()

	go d.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), d.informer.HasSynced) {
		klog.Error("failed to sync caches")
		return
	}
	go wait.Until(d.runWorker, time.Second, ctx.Done())

	<-ctx.Done()
}

func (d *DeploymentMonitor) runWorker() {
	for d.processNextItem() {

	}
}

func (d *DeploymentMonitor) processNextItem() bool {
	key, quit := d.queue.Get()
	if quit {
		return false
	}
	defer d.queue.Done(key)

	obj, exists, err := d.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		klog.Error("failed to get deployment", "key", key, "error", err)
		return true
	}
	if !exists {
		return true
	}

	deployment := obj.(*appsv1.Deployment)
	klog.Infof("Processing deployment: %s/%s", deployment.Namespace, deployment.Name)

	// Trigger Diagnosis
	if d.resolver != nil {
		d.resolver.Diagnose(context.Background(), deployment.Namespace, "Deployment", deployment.Name, "Deployment Failed")
	}

	return true
}
