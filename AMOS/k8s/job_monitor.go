package k8s

import (
	"context"
	"fmt"
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// JobFault represents a detected job issue
type JobFault struct {
	Name      string
	Namespace string
	Reason    string
	Message   string
	Timestamp time.Time
}

// JobMonitor watches for job faults
type JobMonitor struct {
	client     *Client
	faultChan  chan<- JobFault
	stopChan   chan struct{}
}

// NewJobMonitor creates a new JobMonitor
func NewJobMonitor(client *Client, faultChan chan<- JobFault) *JobMonitor {
	return &JobMonitor{
		client:    client,
		faultChan: faultChan,
		stopChan:  make(chan struct{}),
	}
}

// Start begins watching for job events
func (jm *JobMonitor) Start(ctx context.Context) {
	log.Println("Starting Job Monitor...")

	watchNamespace := func(ns string) {
		rw := &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return jm.client.Clientset.BatchV1().Jobs(ns).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (options metav1.ListOptions) (watch.Interface, error) {
				return jm.client.Clientset.BatchV1().Jobs(ns).Watch(context.Background(), options)
			},
		}

		_, controller := cache.NewInformer(
			rw,
			&batchv1.Job{},
			1*time.Minute,
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(oldObj, newObj interface{}) {
					job := newObj.(*batchv1.Job)
					jm.checkJobStatus(job)
				},
				AddFunc: func(obj interface{}) {
					job := obj.(*batchv1.Job)
					jm.checkJobStatus(job)
				},
			},
		)

		go controller.Run(jm.stopChan)
	}

	if len(jm.client.Namespaces) == 0 {
		watchNamespace(corev1.NamespaceAll)
	} else {
		for _, ns := range jm.client.Namespaces {
			watchNamespace(ns)
		}
	}

	<-ctx.Done()
	close(jm.stopChan)
	log.Println("Job Monitor stopped.")
}

func (jm *JobMonitor) checkJobStatus(job *batchv1.Job) {
	// Check for Failed condition
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			jm.emitFault(job, "JobFailed", fmt.Sprintf("Job failed. Reason: %s - %s", condition.Reason, condition.Message))
			return
		}
	}

	// Check BackoffLimit
	if job.Spec.BackoffLimit != nil {
		if job.Status.Failed >= *job.Spec.BackoffLimit {
			jm.emitFault(job, "JobBackoffLimitExceeded", fmt.Sprintf("Failed retries (%d) exceeded backoff limit (%d)", job.Status.Failed, *job.Spec.BackoffLimit))
			return
		}
	}
}

func (jm *JobMonitor) emitFault(job *batchv1.Job, reason, message string) {
	log.Printf("Detected fault in job %s/%s: %s", job.Namespace, job.Name, reason)
	
	jm.faultChan <- JobFault{
		Name:      job.Name,
		Namespace: job.Namespace,
		Reason:    reason,
		Message:   message,
		Timestamp: time.Now(),
	}
}
