package notification

import (
	"fmt"
	"log"

	"github.com/HarshD0011/AMOS/AMOS/service"
)

// EscalationService manages notifications to engineers
type EscalationService struct {
	emailService *EmailService
	engineerEmail string
}

// NewEscalationService creates a new service
func NewEscalationService(emailService *EmailService, engineerEmail string) *EscalationService {
	return &EscalationService{
		emailService:  emailService,
		engineerEmail: engineerEmail,
	}
}

// NotifySuccess sends a success report
func (es *EscalationService) NotifySuccess(fault service.UnifiedFault, actions, explanation string) {
	subject := fmt.Sprintf("AMOS: RESOLVED - %s %s/%s", fault.Type, fault.Namespace, fault.Name)
	
	body := fmt.Sprintf(`
	<h2>Fault Resolved</h2>
	<p><strong>Resource:</strong> %s/%s</p>
	<p><strong>Issue:</strong> %s</p>
	<hr>
	<h3>Action Taken</h3>
	<p>%s</p>
	<h3>Agent Explanation</h3>
	<p>%s</p>
	<p><em>Please review the changes in the cluster.</em></p>
	`, fault.Namespace, fault.Name, fault.Reason, actions, explanation)

	if err := es.emailService.SendEmail([]string{es.engineerEmail}, subject, body); err != nil {
		log.Printf("Failed to send success email: %v", err)
	} else {
		log.Println("Success notification sent to engineer.")
	}
}

// NotifyFailure sends an escalation alert (rollback happened or max retries exceeded)
func (es *EscalationService) NotifyFailure(fault service.UnifiedFault, reason string) {
	subject := fmt.Sprintf("AMOS: ALERT - Failed to Resolve %s %s/%s", fault.Type, fault.Namespace, fault.Name)
	
	body := fmt.Sprintf(`
	<h2 style="color: red;">Intervention Required</h2>
	<p><strong>Resource:</strong> %s/%s</p>
	<p><strong>Issue:</strong> %s</p>
	<p><strong>Status:</strong> %s</p>
	<hr>
	<p>AMOS could not autonomously fix this issue after maximum retries. Changes have been rolled back to ensure stability.</p>
	<p><strong>Please investigate immediately.</strong></p>
	`, fault.Namespace, fault.Name, fault.Reason, reason)

	if err := es.emailService.SendEmail([]string{es.engineerEmail}, subject, body); err != nil {
		log.Printf("Failed to send escalation email: %v", err)
	} else {
		log.Println("Escalation notification sent to engineer.")
	}
}
