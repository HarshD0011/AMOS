package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/k8s"
)

// FaultType enum
type FaultType string

const (
	FaultTypePod        FaultType = "Pod"
	FaultTypeDeployment FaultType = "Deployment"
	FaultTypeJob        FaultType = "Job"
)

// UnifiedFault represents a normalized fault event
type UnifiedFault struct {
	Type         FaultType
	Name         string
	Namespace    string
	Reason       string
	Message      string
	DetectedAt   time.Time
	ResourceID   string // Unique ID for deduplication (e.g. namespace/name)
}

// FaultDetector aggregates faults from monitors and deduplicates/filters them
type FaultDetector struct {
	podFaultChan        chan k8s.PodFault
	deployFaultChan     chan k8s.DeploymentFault
	jobFaultChan        chan k8s.JobFault
	
	orchestratorChan    chan UnifiedFault
	
	activeFaults        map[string]time.Time
	mu                  sync.Mutex
	duplicateWindow     time.Duration
}

// NewFaultDetector creates a new FaultDetector
func NewFaultDetector(orchChan chan UnifiedFault) *FaultDetector {
	return &FaultDetector{
		podFaultChan:     make(chan k8s.PodFault, 100),
		deployFaultChan:  make(chan k8s.DeploymentFault, 100),
		jobFaultChan:     make(chan k8s.JobFault, 100),
		orchestratorChan: orchChan,
		activeFaults:     make(map[string]time.Time),
		duplicateWindow:  5 * time.Minute, // Don't re-report same fault within 5 mins unless resolved (resolved handling complex, so time window for now)
	}
}

// GetChannels returns the write-only channels for monitors
func (fd *FaultDetector) GetChannels() (chan<- k8s.PodFault, chan<- k8s.DeploymentFault, chan<- k8s.JobFault) {
	return fd.podFaultChan, fd.deployFaultChan, fd.jobFaultChan
}

// Start processing faults
func (fd *FaultDetector) Start(ctx context.Context) {
	log.Println("Starting Fault Detector...")
	go fd.cleanupCacheLoop(ctx)

	for {
		select {
		case pf := <-fd.podFaultChan:
			fd.processFault(UnifiedFault{
				Type:       FaultTypePod,
				Name:       pf.PodName,
				Namespace:  pf.Namespace,
				Reason:     pf.Reason,
				Message:    pf.Message,
				DetectedAt: pf.Timestamp,
				ResourceID: fmt.Sprintf("pod/%s/%s", pf.Namespace, pf.PodName),
			})

		case df := <-fd.deployFaultChan:
			fd.processFault(UnifiedFault{
				Type:       FaultTypeDeployment,
				Name:       df.Name,
				Namespace:  df.Namespace,
				Reason:     df.Reason,
				Message:    df.Message,
				DetectedAt: df.Timestamp,
				ResourceID: fmt.Sprintf("deploy/%s/%s", df.Namespace, df.Name),
			})

		case jf := <-fd.jobFaultChan:
			fd.processFault(UnifiedFault{
				Type:       FaultTypeJob,
				Name:       jf.Name,
				Namespace:  jf.Namespace,
				Reason:     jf.Reason,
				Message:    jf.Message,
				DetectedAt: jf.Timestamp,
				ResourceID: fmt.Sprintf("job/%s/%s", jf.Namespace, jf.Name),
			})

		case <-ctx.Done():
			log.Println("Fault Detector stopped.")
			return
		}
	}
}

func (fd *FaultDetector) processFault(fault UnifiedFault) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	lastSeen, exists := fd.activeFaults[fault.ResourceID]
	if exists {
		// If seen recently, ignore to prevent spamming
		if time.Since(lastSeen) < fd.duplicateWindow {
			// Log verbose if debug?
			return
		}
	}

	// Update last seen
	fd.activeFaults[fault.ResourceID] = time.Now()
	
	log.Printf("Fault confirmed: %s %s/%s - %s", fault.Type, fault.Namespace, fault.Name, fault.Reason)
	
	// Send to orchestrator
	select {
	case fd.orchestratorChan <- fault:
	default:
		log.Printf("Warning: Orchestrator channel full, dropping fault event for %s", fault.ResourceID)
	}
}

func (fd *FaultDetector) cleanupCacheLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fd.mu.Lock()
			now := time.Now()
			for id, ts := range fd.activeFaults {
				if now.Sub(ts) > fd.duplicateWindow*2 {
					delete(fd.activeFaults, id)
				}
			}
			fd.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
