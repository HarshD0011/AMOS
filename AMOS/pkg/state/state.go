package state

import (
	"fmt"
	"sync"
	"time"
)

// IssueStatus represents the current state of a resource issue.
type IssueStatus string

const (
	StatusFailed    IssueStatus = "Failed"    // Red
	StatusFixing    IssueStatus = "Fixing"    // Yellow
	StatusDeploying IssueStatus = "Deploying" // Blue
	StatusResolved  IssueStatus = "Resolved"  // Green
)

// Issue represents a tracked resource failure.
type Issue struct {
	ID           string      `json:"id"`
	Resource     string      `json:"resource"` // e.g., "pod/my-pod"
	Namespace    string      `json:"namespace"`
	Kind         string      `json:"kind"`
	Name         string      `json:"name"`
	Status       IssueStatus `json:"status"`
	ErrorMessage string      `json:"errorMessage"`
	Logs         string      `json:"logs"`
	Events       string      `json:"events"`
	Diagnosis    string      `json:"diagnosis"`
	Suggestion   string      `json:"suggestion"`
	Diagnosed    bool        `json:"diagnosed"` // True if LLM diagnosis was already performed
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// StateManager handles thread-safe access to issues.
type StateManager struct {
	mu     sync.RWMutex
	issues map[string]*Issue
}

// NewStateManager creates a new StateManager.
func NewStateManager() *StateManager {
	return &StateManager{
		issues: make(map[string]*Issue),
	}
}

// ReportFailure registers or updates a failure.
// Transition: -> Failed (Red)
func (sm *StateManager) ReportFailure(namespace, kind, name, errorMessage string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, kind, name)
	if issue, exists := sm.issues[key]; exists {
		// Update existing issue if it's not already resolved
		if issue.Status != StatusResolved {
			issue.ErrorMessage = errorMessage
			issue.UpdatedAt = time.Now()
			// If it was fixing/deploying but failed again, maybe keep current status or reset to Failed?
			// For simplicity, let's keep it as is if it's progressing, unless error changed significantly?
			// Actually, if we report failure again, it might mean the fix failed. Let's reset to Failed/Fixing based on flow.
			// But monitors call this constantly on sync loop. We shouldn't flicker.
			// Only update if status is Resolved? No, if it's Resolved and we report failure, it's a regression.
			return
		}
		// If it was resolved, reopen it
		issue.Status = StatusFailed
		issue.ErrorMessage = errorMessage
		issue.UpdatedAt = time.Now()
	} else {
		// New Issue
		sm.issues[key] = &Issue{
			ID:           key,
			Resource:     fmt.Sprintf("%s/%s", kind, name),
			Namespace:    namespace,
			Kind:         kind,
			Name:         name,
			Status:       StatusFailed,
			ErrorMessage: errorMessage,
			UpdatedAt:    time.Now(),
		}
	}
}

// UpdateStatus moves the issue to a new stage.
func (sm *StateManager) UpdateStatus(namespace, kind, name string, status IssueStatus) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, kind, name)
	if issue, exists := sm.issues[key]; exists {
		issue.Status = status
		issue.UpdatedAt = time.Now()
	}
}

// AddDiagnosis adds analysis details to the issue and marks it as diagnosed.
func (sm *StateManager) AddDiagnosis(namespace, kind, name, logs, events, diagnosis string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, kind, name)
	if issue, exists := sm.issues[key]; exists {
		issue.Logs = logs
		issue.Events = events
		issue.Diagnosis = diagnosis
		issue.Diagnosed = true // Mark as diagnosed to prevent repeated LLM calls
		issue.UpdatedAt = time.Now()
	}
}

// Resolve marks an issue as resolved.
// Transition: -> Resolved (Green)
func (sm *StateManager) Resolve(namespace, kind, name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, kind, name)
	if issue, exists := sm.issues[key]; exists {
		issue.Status = StatusResolved
		issue.ErrorMessage = ""
		issue.Diagnosed = false // Reset so it can be re-diagnosed if issue recurs
		issue.UpdatedAt = time.Now()
	}
}

// NeedsDiagnosis checks if an issue exists and hasn't been diagnosed yet.
// Returns true only if the issue exists, is not resolved, and hasn't been diagnosed.
func (sm *StateManager) NeedsDiagnosis(namespace, kind, name string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := fmt.Sprintf("%s/%s/%s", namespace, kind, name)
	if issue, exists := sm.issues[key]; exists {
		return issue.Status != StatusResolved && !issue.Diagnosed
	}
	return false
}

// GetAllIssues returns a list of all tracked issues.
func (sm *StateManager) GetAllIssues() []*Issue {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	list := make([]*Issue, 0, len(sm.issues))
	for _, issue := range sm.issues {
		list = append(list, issue)
	}
	return list
}
