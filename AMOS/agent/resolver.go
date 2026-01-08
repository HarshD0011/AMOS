package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/HarshD0011/AMOS/AMOS/tools"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

// Resolver manages the diagnosis and resolution of K8s faults.
type Resolver struct {
	tools *tools.K8sTools
	llm   model.LLM
}

// NewResolver creates a new Resolver.
func NewResolver(t *tools.K8sTools) *Resolver {
	ctx := context.Background()
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GOOGLE_API_KEY not set")
	}

	// Initialize Gemini Model
	m, err := gemini.NewModel(ctx, "gemini-1.5-pro", &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("Error initializing model: %v\n", err)
		return &Resolver{tools: t}
	}

	return &Resolver{
		tools: t,
		llm:   m,
	}
}

// Diagnose investigates a failure and attempts to fix it.
func (r *Resolver) Diagnose(ctx context.Context, namespace, kind, name string, errMessage string) {
	fmt.Printf("Diagnosing %s/%s %s: %s\n", kind, namespace, name, errMessage)

	// 1. Gather Information
	var logs string
	var err error
	if kind == "Pod" {
		logs, err = r.tools.GetPodLogs(namespace, name)
	} else if kind == "Deployment" {
		logs, err = r.tools.GetDeploymentLogs(namespace, name)
	}

	if err != nil {
		fmt.Printf("Failed to get logs: %v\n", err)
		logs = "Could not fetch logs."
	}

	// 2. Ask Model for diagnosis
	input := fmt.Sprintf("Context: Resource %s/%s (%s) failed.\nError Message: %s\n\nLogs (last 50 lines):\n%s\n\nPlease diagnose the issue and suggest a fix.", namespace, name, kind, errMessage, logs)

	var diagnosisBuilder strings.Builder
	if r.llm != nil {
		req := &model.LLMRequest{
			Model: "gemini-1.5-pro",
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

	// 3. Notification (with diagnosis)
	subject := fmt.Sprintf("Alert: %s %s/%s Failed", kind, namespace, name)
	body := fmt.Sprintf("Diagnosis:\n%s\n\nLogs (tail):\n%s", diagnosis, logs)

	if err := r.tools.SendEmail(subject, body); err != nil {
		fmt.Printf("Failed to send email: %v\n", err)
	} else {
		fmt.Printf("Email notification sent to engineer regarding %s/%s.\n", namespace, name)
	}
}
