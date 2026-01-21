package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/HarshD0011/AMOS/AMOS/pkg/state"
	"github.com/HarshD0011/AMOS/AMOS/tools"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

// Resolver manages the diagnosis and resolution of K8s faults.
type Resolver struct {
	tools        *tools.K8sTools
	llm          model.LLM
	stateManager *state.StateManager
}

// NewResolver creates a new Resolver.
func NewResolver(t *tools.K8sTools, sm *state.StateManager) *Resolver {
	ctx := context.Background()
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GOOGLE_API_KEY not set")
	}

	// Initialize Gemini Model
	m, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("Error initializing model: %v\n", err)
		return &Resolver{tools: t, stateManager: sm}
	}

	return &Resolver{
		tools:        t,
		llm:          m,
		stateManager: sm,
	}
}

// Diagnose investigates a failure and attempts to fix it.
func (r *Resolver) Diagnose(ctx context.Context, namespace, kind, name string, errMessage string) {
	fmt.Printf("Diagnosing %s/%s %s: %s\n", kind, namespace, name, errMessage)

	// Update State: Fixing (Yellow)
	if r.stateManager != nil {
		r.stateManager.UpdateStatus(namespace, kind, name, state.StatusFixing)
	}

	// 1. Gather Information
	var logs string
	var err error
	if kind == "Pod" {
		logs, err = r.tools.GetPodLogs(namespace, name)
	} else if kind == "Deployment" {
		logs, err = r.tools.GetDeploymentLogs(namespace, name)
	} else if kind == "Job" {
		logs, err = r.tools.GetJobLogs(namespace, name)
	}

	if err != nil {
		fmt.Printf("Failed to get logs: %v\n", err)
		logs = "Could not fetch logs."
	}

	// 1b. Fetch Events
	events, err := r.tools.GetResourceEvents(kind, namespace, name)
	if err != nil {
		fmt.Printf("Failed to get events: %v\n", err)
		events = "Could not fetch events."
	}

	// 2. Ask Model for diagnosis
	input := fmt.Sprintf(`Context: Resource %s/%s (%s) failed.
Error Message: %s

Events (kubectl describe):
%s

Logs (last 50 lines):
%s

Please diagnose the issue and provide:
1. A brief RECOMMENDED SOLUTION (2-3 lines)
2. A detailed ANALYSIS (full explanation)

Format your response with these exact headers:
## RECOMMENDED SOLUTION
[your brief solution here]

## ANALYSIS
[your detailed analysis here]`, namespace, name, kind, errMessage, events, logs)

	var diagnosisBuilder strings.Builder
	if r.llm != nil {
		req := &model.LLMRequest{
			Model: "gemini-2.5-flash",
			Contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: input},
					},
				},
			},
		}

		// Use iterator to get response
		for resp, err := range r.llm.GenerateContent(ctx, req, false) {
			if err != nil {
				diagnosisBuilder.WriteString(fmt.Sprintf("\n[Error: %v]", err))
				break
			}
			if resp.Content != nil {
				for _, part := range resp.Content.Parts {
					diagnosisBuilder.WriteString(part.Text)
				}
			}
		}
	} else {
		diagnosisBuilder.WriteString("Model not initialized (check API key).")
	}

	diagnosis := diagnosisBuilder.String()
	fmt.Printf("Diagnosis: %s\n", diagnosis)

	// Update State: Add Diagnosis and Events
	if r.stateManager != nil {
		r.stateManager.AddDiagnosis(namespace, kind, name, logs, events, diagnosis)
	}

	// 3. Notification (with diagnosis)
	subject := fmt.Sprintf("Alert: %s %s/%s Failed", kind, namespace, name)
	body := fmt.Sprintf("Diagnosis:\n%s\n\nLogs (tail):\n%s", diagnosis, logs)

	if err := r.tools.SendEmail(subject, body); err != nil {
		fmt.Printf("Failed to send email: %v\n", err)
	} else {
		fmt.Printf("Email notification sent to engineer regarding %s/%s.\n", namespace, name)
		// Update State: Deploying (Blue) - Assuming email triggers action
		if r.stateManager != nil {
			r.stateManager.UpdateStatus(namespace, kind, name, state.StatusDeploying)
		}
	}
}
