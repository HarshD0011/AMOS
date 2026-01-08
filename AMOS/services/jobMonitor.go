package services

import (
	"context"
	"fmt"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type JobMonitor struct {
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
	resolver *agent.Resolver
}

func NewJobMonitor(client kubernetes.Interface, resolver *agent.Resolver) *JobMonitor {

	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Batch().V1().Jobs().Informer()

	j := &JobMonitor{
		informer: informer,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "job-monitor"),
		resolver: resolver,
	}
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				job := obj.(*batchv1.Job)
				if job.Status.Failed > 0 {
					key, err := cache.MetaNamespaceKeyFunc(job)
					if err == nil {
						j.queue.Add(key)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newJob := newObj.(*batchv1.Job)
				// Check for failure
				if newJob.Status.Failed > 0 {
					key, err := cache.MetaNamespaceKeyFunc(newJob)
					if err == nil {
						j.queue.Add(key)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err != nil {
					return
				}
				j.queue.Add(key)
			},
		},
	)

	return j
}

func (j *JobMonitor) Run(ctx context.Context) {
	defer j.queue.ShutDown()

	go j.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), j.informer.HasSynced) {
		klog.Error("failed to sync caches")
		return
	}
	go wait.Until(j.runWorker, time.Second, ctx.Done())

	<-ctx.Done()
}

func (j *JobMonitor) runWorker() {
	for j.processNextItem() {
	}
}

func (j *JobMonitor) processNextItem() bool {
	key, quit := j.queue.Get()
	if quit {
		return false
	}
	defer j.queue.Done(key)

	obj, exists, err := j.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		klog.Error("failed to get job", "key", key, "error", err)
		return true
	}
	if !exists {
		return true
	}

	job := obj.(*batchv1.Job)
	klog.Infof("Processing job: %s/%s", job.Namespace, job.Name)

	// Trigger Diagnosis
	if j.resolver != nil {
		j.resolver.Diagnose(context.Background(), job.Namespace, "Job", job.Name, fmt.Sprintf("Job failed with %d failures", job.Status.Failed))
	}

	return true
}
