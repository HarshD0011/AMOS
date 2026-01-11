package services

import (
	"context"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/pkg/state"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type PodMonitor struct {
	informer     cache.SharedIndexInformer
	queue        workqueue.RateLimitingInterface
	resolver     *agent.Resolver
	stateManager *state.StateManager
}

func NewPodMonitor(client kubernetes.Interface, resolver *agent.Resolver, sm *state.StateManager) *PodMonitor {

	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Core().V1().Pods().Informer()

	c := &PodMonitor{
		informer:     informer,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "pod-monitor"),
		resolver:     resolver,
		stateManager: sm,
	}

	isPodFailed := func(pod *corev1.Pod) bool {
		if pod.Status.Phase == corev1.PodFailed {
			return true
		}
		// Check for specific container errors (CrashLoopBackOff, ImagePullBackOff, etc.)
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Waiting != nil {
				reason := status.State.Waiting.Reason
				if reason == "CrashLoopBackOff" || reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "CreateContainerConfigError" || reason == "RunContainerError" {
					return true
				}
			}
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				return true
			}
		}
		// Also check InitContainers
		for _, status := range pod.Status.InitContainerStatuses {
			if status.State.Waiting != nil {
				reason := status.State.Waiting.Reason
				if reason == "CrashLoopBackOff" || reason == "ImagePullBackOff" || reason == "ErrImagePull" {
					return true
				}
			}
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				return true
			}
		}
		return false
	}

	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				if isPodFailed(pod) {
					// Report Failure (Red)
					if c.stateManager != nil {
						c.stateManager.ReportFailure(pod.Namespace, "Pod", pod.Name, string(pod.Status.Phase))
					}
					key, err := cache.MetaNamespaceKeyFunc(pod)
					if err == nil {
						c.queue.Add(key)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newpod := newObj.(*corev1.Pod)
				if isPodFailed(newpod) {
					// Report Failure (Red)
					if c.stateManager != nil {
						c.stateManager.ReportFailure(newpod.Namespace, "Pod", newpod.Name, string(newpod.Status.Phase))
					}
					key, err := cache.MetaNamespaceKeyFunc(newpod)
					if err == nil {
						c.queue.Add(key)
					}
				} else if newpod.Status.Phase == corev1.PodRunning || newpod.Status.Phase == corev1.PodSucceeded {
					// Resolve Issue (Green)
					if c.stateManager != nil {
						c.stateManager.Resolve(newpod.Namespace, "Pod", newpod.Name)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				// If deleted, maybe resolve? or just ignore.
				// Let's resolve usage if deleted to clean up UI.
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					// Process DeletedFinalStateUnknown
					tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
					if !ok {
						return
					}
					pod, ok = tombstone.Obj.(*corev1.Pod)
					if !ok {
						return
					}
				}
				if c.stateManager != nil {
					c.stateManager.Resolve(pod.Namespace, "Pod", pod.Name)
				}
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
