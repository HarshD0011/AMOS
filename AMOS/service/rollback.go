package service

import (
	"context"
	"fmt"
	"log"

	"github.com/HarshD0011/AMOS/AMOS/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RollbackService handles restoring state
type RollbackService struct {
	client   *k8s.Client
	snapshot *SnapshotService
}

// NewRollbackService creates a rollback service
func NewRollbackService(client *k8s.Client, snapshot *SnapshotService) *RollbackService {
	return &RollbackService{
		client:   client,
		snapshot: snapshot,
	}
}

// PerformRollback restores the resource to its snapshotted state
func (rs *RollbackService) PerformRollback(kind, namespace, name string) error {
	snap, found := rs.snapshot.GetSnapshot(kind, namespace, name)
	if !found {
		return fmt.Errorf("no snapshot found for %s/%s/%s", kind, namespace, name)
	}

	log.Printf("Initiating ROLLBACK for %s/%s/%s (Captured at: %s)", kind, namespace, name, snap.CapturedAt)

	switch kind {
	case "Deployment":
		if snap.Deployment == nil {
			return fmt.Errorf("snapshot data is nil for deployment")
		}
		// Restore key fields or replace spec. 
		// Replacing Spec is safer but we need to handle resourceVersion conflicts if we just do Update.
		// Safe way: Get current, replace spec with snapshotted spec, Update.
		
		current, err := rs.client.Clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get current deployment for rollback: %w", err)
		}

		// replace spec with old spec
		current.Spec = snap.Deployment.Spec
		
		_, err = rs.client.Clientset.AppsV1().Deployments(namespace).Update(context.Background(), current, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to apply rollback update: %w", err)
		}

		log.Printf("Rollback successful for %s/%s", namespace, name)
		return nil

	default:
		return fmt.Errorf("rollback not supported for kind: %s", kind)
	}
}
