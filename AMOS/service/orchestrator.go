package service

import (
	"context"
	"fmt"
	"log"

	"github.com/HarshD0011/AMOS/AMOS/agent"
	"github.com/HarshD0011/AMOS/AMOS/notification"
)

// Orchestrator coordinates the self-healing process
type Orchestrator struct {
	faultDetector *FaultDetector
	agent         *agent.RemediationAgent
	ctxGen        *ContextGenerator
	snapshot      *SnapshotService
	retry         *RetryManager
	rollback      *RollbackService
	escalation    *notification.EscalationService
	faultChan     chan UnifiedFault
}

// NewOrchestrator creates the main orchestrator
func NewOrchestrator(
	fd *FaultDetector,
	ag *agent.RemediationAgent,
	cg *ContextGenerator,
	ss *SnapshotService,
	rm *RetryManager,
	rb *RollbackService,
	es *notification.EscalationService,
	faultChan chan UnifiedFault,
) *Orchestrator {
	return &Orchestrator{
		faultDetector: fd,
		agent:         ag,
		ctxGen:        cg,
		snapshot:      ss,
		retry:         rm,
		rollback:      rb,
		escalation:    es,
		faultChan:     faultChan,
	}
}

// Start begins the orchestration loop
func (o *Orchestrator) Start(ctx context.Context) {
	log.Println("AMOS Orchestrator started. Waiting for faults...")

	// Listen for faults from Detector
	for {
		select {
		case fault := <-o.faultChan:
			// Process in a goroutine to not block detection
			go o.handleFault(ctx, fault)
		case <-ctx.Done():
			log.Println("Orchestrator stopped.")
			return
		}
	}
}

func (o *Orchestrator) handleFault(ctx context.Context, fault UnifiedFault) {
	log.Printf("Orchestrator handling fault: %s", fault.ResourceID)

	// 1. Check Retry Limits
	if !o.retry.CanRetry(fault.ResourceID) {
		log.Printf("Max retries exceeded for %s. Initiating Rollback & Escalation.", fault.ResourceID)
		
		reason := fmt.Sprintf("Max retries (%d) exceeded.", o.retry.GetAttemptCount(fault.ResourceID))
		
		// 1a. Rollback (only supported for Deployments currently in our implementation)
		if fault.Type == FaultTypeDeployment {
			if err := o.rollback.PerformRollback(string(fault.Type), fault.Namespace, fault.Name); err != nil {
				log.Printf("Rollback failed: %v", err)
				reason += fmt.Sprintf(" Rollback failed: %v", err)
			} else {
				reason += " Rollback performed successfully."
			}
		}

		// 1b. Escalate
		o.escalation.NotifyFailure(fault, reason)
		
		// Reset counters? Maybe manual intervention required, so we shouldn't reset immediately.
		// For now we leave it so it doesn't loop until restarted or cleared manually.
		return
	}

	// 2. Snapshot State (for first attempt preferably, or continuous? First is safer)
	// Only snapshot if attempts == 0 to save original state? 
	// Or snapshot before every change? If we want to rollback to *working* state, we needed snapshot execution *before* the bad change.
	// But AMOS starts *after* the bad change is detected.
	// So "Rollback" here means "Undo AMOS changes" if AMOS makes it worse? 
	// OR "Rollback" means "Rollback the user's bad deployment"? 
	// Usually "Rollback user deployment" implies `kubectl rollout undo`.
	// Our `state_snapshot` captures state *at detection time*. 
	// If the state is ALREADY broken (which it is), restoring it keeps it broken.
	// So `state_snapshot` is useful if the AGENT tries a fix that breaks it FURTHER.
	// For "Cluster Rectification", the Agent should use `Rollback` tool or Patch to previous image.
	// Let's assume Snapshot is for safety against Agent actions.
	
	if err := o.snapshot.Capture(string(fault.Type), fault.Namespace, fault.Name); err != nil {
		log.Printf("Warning: Failed to capture snapshot: %v", err)
	}

	// 3. Increment Retry Count
	o.retry.IncrementRetry(fault.ResourceID)

	// 4. Generate AI Context
	prompt, err := o.ctxGen.GenerateContext(fault)
	if err != nil {
		log.Printf("Error generating context: %v", err)
		return
	}

	// 5. Invoke Agent
	log.Printf("Invoking Remediation Agent for %s...", fault.ResourceID)
	response, err := o.agent.Fix(ctx, prompt)
	if err != nil {
		log.Printf("Agent failed to run: %v", err)
		return 
	}

	// 6. Notify engineer of attempt/success
	// We assume success if Agent ran without error for now, ideally Agent reports "I fixed it".
	// Verification step would wait for monitoring to clear.
	// For MVP, we send a notification of the ACTION taken.
	o.escalation.NotifySuccess(fault, "Agent Remediation Attempted", response)
}
