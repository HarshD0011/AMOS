package services

import (
	"context"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
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
	resolver *agent.Resolver
}

func NewDeploymentMonitor(client kubernetes.Interface, resolver *agent.Resolver) *DeploymentMonitor {
	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Apps().V1().Deployments().Informer()

	c := &DeploymentMonitor{
		informer: informer,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deployment-monitor"),
		resolver: resolver,
	}
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				deploy := obj.(*appsv1.Deployment)
				for _, cond := range deploy.Status.Conditions {
					if cond.Type == appsv1.DeploymentProgressing && cond.Status == "False" && cond.Reason == "ProgressDeadlineExceeded" {
						key, err := cache.MetaNamespaceKeyFunc(deploy)
						if err == nil {
							c.queue.Add(key)
						}
						break
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newDeployment := newObj.(*appsv1.Deployment)
				// Check if deployment has failed.
				for _, cond := range newDeployment.Status.Conditions {
					if cond.Type == appsv1.DeploymentProgressing && cond.Status == "False" && cond.Reason == "ProgressDeadlineExceeded" {
						key, err := cache.MetaNamespaceKeyFunc(newDeployment)
						if err == nil {
							c.queue.Add(key)
						}
						break
					}
				}
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

	deployment := obj.(*appsv1.Deployment)
	klog.Infof("Processing deployment: %s/%s", deployment.Namespace, deployment.Name)

	// Trigger Diagnosis
	if d.resolver != nil {
		d.resolver.Diagnose(context.Background(), deployment.Namespace, "Deployment", deployment.Name, "Deployment Failed")
	}

	return true
}
