package services

import (
	"context"
	"time"

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
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
}

func NewDeploymentMonitor(client kubernetes.Interface) *DeploymentMonitor {
	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Apps().V1().Deployments().Informer()

	c := &DeploymentMonitor{
		informer: informer,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deployment-monitor"),
	}
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				newDeployment := newObj.(*appsv1.Deployment)
				// oldDeployment := oldObj.(*appsv1.Deployment) // unused

				// Check if deployment has failed.
				// A deployment is considered failed if the Progressing condition is False with Reason=ProgressDeadlineExceeded
				for _, cond := range newDeployment.Status.Conditions {
					if cond.Type == appsv1.DeploymentProgressing && cond.Status == "False" && cond.Reason == "ProgressDeadlineExceeded" {
						// We can also check other failure modes, or if AvailableReplicas == 0 and Replicas > 0 for a long time?
						// For now, let's catch explicit failures or obscure states?
						// The prompt implied "monitoring... for faults".
						// Let's add it to queue.
						c.queue.Add(newDeployment)
						break
					}
				}

				// Also check if Replicas > 0 and different from AvailableReplicas for a prolonged time?
				// But that's hard to distinguish from rolling update.
				// For the "issue persists" logic, maybe we need to track it.
				// For now simpler check: if we see a failure condition.
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err != nil {
					return
				}
				c.queue.Add(key)
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

	// Process the object?
	// For now just return true.
	klog.Infof("Processing deployment: %s/%s", obj.(*appsv1.Deployment).Namespace, obj.(*appsv1.Deployment).Name)
	return true
}
