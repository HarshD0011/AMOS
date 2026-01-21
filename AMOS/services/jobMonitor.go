package services

import (
	"context"
	"fmt"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/pkg/state"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type JobMonitor struct {
	informer     cache.SharedIndexInformer
	queue        workqueue.RateLimitingInterface
	resolver     *agent.Resolver
	stateManager *state.StateManager
}

func NewJobMonitor(client kubernetes.Interface, resolver *agent.Resolver, sm *state.StateManager) *JobMonitor {

	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	informer := factory.Batch().V1().Jobs().Informer()

	c := &JobMonitor{
		informer:     informer,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "job-monitor"),
		resolver:     resolver,
		stateManager: sm,
	}
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				job := obj.(*batchv1.Job)
				if job.Status.Failed > 0 {
					// Report Failure (Red)
					if c.stateManager != nil {
						msg := fmt.Sprintf("Job Failed: %d failed pods", job.Status.Failed)
						c.stateManager.ReportFailure(job.Namespace, "Job", job.Name, msg)
					}
					key, err := cache.MetaNamespaceKeyFunc(job)
					if err == nil {
						c.queue.Add(key)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newJob := newObj.(*batchv1.Job)
				// Check for failure
				if newJob.Status.Failed > 0 {
					// Report Failure (Red)
					if c.stateManager != nil {
						msg := fmt.Sprintf("Job Failed: %d failed pods", newJob.Status.Failed)
						c.stateManager.ReportFailure(newJob.Namespace, "Job", newJob.Name, msg)
					}
					key, err := cache.MetaNamespaceKeyFunc(newJob)
					if err == nil {
						c.queue.Add(key)
					}
				} else if newJob.Status.Succeeded > 0 {
					// Resolve Issue (Green)
					if c.stateManager != nil {
						c.stateManager.Resolve(newJob.Namespace, "Job", newJob.Name)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				// Resolve on delete
				job, ok := obj.(*batchv1.Job)
				if !ok {
					tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
					if !ok {
						return
					}
					job, ok = tombstone.Obj.(*batchv1.Job)
					if !ok {
						return
					}
				}
				if c.stateManager != nil {
					c.stateManager.Resolve(job.Namespace, "Job", job.Name)
				}
			},
		},
	)

	return c
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

	// Only trigger LLM diagnosis if the issue hasn't been diagnosed yet
	if j.resolver != nil && j.stateManager != nil {
		if j.stateManager.NeedsDiagnosis(job.Namespace, "Job", job.Name) {
			klog.Infof("Triggering LLM diagnosis for job: %s/%s", job.Namespace, job.Name)
			j.resolver.Diagnose(context.Background(), job.Namespace, "Job", job.Name, fmt.Sprintf("Job failed with %d failures", job.Status.Failed))
		} else {
			klog.Infof("Skipping diagnosis for job %s/%s (already diagnosed or resolved)", job.Namespace, job.Name)
		}
	}

	return true
}
