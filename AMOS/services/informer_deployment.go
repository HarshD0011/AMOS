package services

import (
	"context"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/tools"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// InformerDeployment manages the lifecycle of all informers/monitors.
type InformerDeployment struct {
	client            kubernetes.Interface
	podMonitor        *PodMonitor
	deploymentMonitor *DeploymentMonitor
	jobMonitor        *JobMonitor
	resolver          *agent.Resolver
}

// NewInformerDeployment creates a new InformerDeployment.
func NewInformerDeployment(client kubernetes.Interface) *InformerDeployment {
	k8sTools := tools.NewK8sTools(client)
	resolver := agent.NewResolver(k8sTools)

	return &InformerDeployment{
		client:            client,
		podMonitor:        NewPodMonitor(client, resolver),
		deploymentMonitor: NewDeploymentMonitor(client, resolver),
		jobMonitor:        NewJobMonitor(client, resolver),
		resolver:          resolver,
	}
}

// Run starts all informers and blocks until the context is cancelled.
func (id *InformerDeployment) Run(ctx context.Context) {
	klog.Info("Starting InformerDeployment...")

	// Start all monitors
	go id.podMonitor.Run(ctx)
	go id.deploymentMonitor.Run(ctx)
	go id.jobMonitor.Run(ctx)

	klog.Info("All informers started.")
	<-ctx.Done()
	klog.Info("InformerDeployment shutting down...")
}
