package service

import (
	"log"
	"os"
)

// Logger is a wrapper around standard logs but could be enhanced for structured logging
// Currently using stdlib log for simplicity in MVP
type Logger struct {
	*log.Logger
}

// NewLogger creates a simple logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[AMOS] ", log.LstdFlags),
	}
}

// LogFault logs a fault occurrence
func (l *Logger) LogFault(fault UnifiedFault) {
	l.Printf("FAULT DETECTED: [%s] %s/%s - %s: %s", fault.Type, fault.Namespace, fault.Name, fault.Reason, fault.Message)
}

// LogAction logs a remediation action
func (l *Logger) LogAction(resource string, action string) {
	l.Printf("ACTION TAKEN: %s on %s", action, resource)
}

// LogInfo normal info log
func (l *Logger) LogInfo(msg string) {
	l.Printf("INFO: %s", msg)
}

// LogError error log
func (l *Logger) LogError(err error, msg string) {
	l.Printf("ERROR: %s: %v", msg, err)
}