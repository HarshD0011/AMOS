package agent

import (
	"log"
	"strings"
)

// QueryProcessor handles validation and extraction of useful info from Agent responses
// ADK likely handles the tool execution loop, so the final response is text explanation.
// We can use this to parse structured decisions if we forced JSON output, 
// strictly for now it just logs and validates emptiness.
type QueryProcessor struct{}

func NewQueryProcessor() *QueryProcessor {
	return &QueryProcessor{}
}

// ProcessResponse analyzes the agent's output
func (qp *QueryProcessor) ProcessResponse(response string) (actionsTaken []string, summary string) {
	if strings.TrimSpace(response) == "" {
		log.Println("Warning: Agent returned empty response.")
		return nil, "No response from agent."
	}

	// Simple heuristic: check if response mentions actions
	// Real implementation might parse a structured log if we instructed agent to output JSON
	
	return nil, response
}