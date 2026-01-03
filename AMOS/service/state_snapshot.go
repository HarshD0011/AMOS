package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/k8s"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Snapshot represents a saved state of a resource
type Snapshot struct {
	CapturedAt time.Time
	Kind       string
	Name       string
	Namespace  string
	// For deployments, we specifically care about Replicas, Image, Env vars usually
	// For full rollback, we need the full object or mostly spec.
	// Saving the full struct is easiest.
	Deployment *appsv1.Deployment
	// Add other types as needed (StatefulSet, etc.)
}

// SnapshotService manages state snapshots
type SnapshotService struct {
	client    *k8s.Client
	snapshots map[string]Snapshot // Key: kind/namespace/name
	mu        sync.RWMutex
}

// NewSnapshotService creates a new service
func NewSnapshotService(client *k8s.Client) *SnapshotService {
	return &SnapshotService{
		client:    client,
		snapshots: make(map[string]Snapshot),
	}
}

// Capture saves the current state of a resource
func (ss *SnapshotService) Capture(kind, namespace, name string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", kind, namespace, name)
	log.Printf("Capturing snapshot for %s", key)

	switch kind {
	case "Deployment":
		deploy, err := ss.client.Clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// Copy relevant parts or whole object
		// We need a deep copy if we were mutating, but Get returns a fresh ptr usually
		ss.snapshots[key] = Snapshot{
			CapturedAt: time.Now(),
			Kind:       kind,
			Namespace:  namespace,
			Name:       name,
			Deployment: deploy,
		}
	default:
		// For Pods/Jobs, usually we don't snapshot to restore exact state because they are ephemeral or immutable specs (mostly)
		// For Job, maybe. For Pod, we usually care about the controller (deployment).
		// We'll skip for now or implement if needed.
		return fmt.Errorf("snapshot not supported for kind: %s", kind)
	}
	return nil
}

// GetSnapshot retrieves a snapshot
func (ss *SnapshotService) GetSnapshot(kind, namespace, name string) (*Snapshot, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	key := fmt.Sprintf("%s/%s/%s", kind, namespace, name)
	snap, found := ss.snapshots[key]
	return &snap, found
}
