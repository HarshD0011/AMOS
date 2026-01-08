package services

import (
	"context"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type PodMonitor struct {
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
	resolver *agent.Resolver
}

func NewPodMonitor(client kubernetes.Interface, resolver *agent.Resolver) *PodMonitor {

	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Core().V1().Pods().Informer()

	c := &PodMonitor{
		informer: informer,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "pod-monitor"),
		resolver: resolver,
	}
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
					key, err := cache.MetaNamespaceKeyFunc(pod)
					if err == nil {
						c.queue.Add(key)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newpod := newObj.(*corev1.Pod)
				if newpod.Status.Phase == corev1.PodFailed || newpod.Status.Phase == corev1.PodUnknown {
					key, err := cache.MetaNamespaceKeyFunc(newpod)
					if err == nil {
						c.queue.Add(key)
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

func (c *PodMonitor) Run(ctx context.Context) {
	defer c.queue.ShutDown()

	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		klog.Error("failed to sync caches")
		return
	}
	go wait.Until(c.runWorker, time.Second, ctx.Done())

	<-ctx.Done()
}
func (c *PodMonitor) runWorker() {
	for c.processNextItem() {
	}
}
func (c *PodMonitor) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		klog.Error("failed to get pod", "key", key, "error", err)
		return true
	}
	if !exists {
		return true
	}

	pod := obj.(*corev1.Pod)
	klog.Infof("Processing pod: %s/%s", pod.Namespace, pod.Name)

	// Trigger Diagnosis
	if c.resolver != nil {
		c.resolver.Diagnose(context.Background(), pod.Namespace, "Pod", pod.Name, string(pod.Status.Phase))
	}

	return true
}
