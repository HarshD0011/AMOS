package services

import (
	"context"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/pkg/state"
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
func NewInformerDeployment(client kubernetes.Interface, sm *state.StateManager) *InformerDeployment {
	k8sTools := tools.NewK8sTools(client)
	resolver := agent.NewResolver(k8sTools, sm)

	// Note: We need to update monitors to accept sm too, for now let's just update monitors signatures in other files as well
	// But wait, I haven't updated DeploymentMonitor and JobMonitor yet. I should do that.
	// For now, I will assume I will update them next.

	// Actually, I can pass nil for now to compilation pass if I haven't updated them, but standard prevents broken state.
	// Let's proceed with updating NewInformerDeployment properly assuming other files will match.
	return &InformerDeployment{
		client:            client,
		podMonitor:        NewPodMonitor(client, resolver, sm),
		deploymentMonitor: NewDeploymentMonitor(client, resolver, sm),
		jobMonitor:        NewJobMonitor(client, resolver, sm),
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
