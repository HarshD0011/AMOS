package service

import (
	"fmt"
	"strings"

	"github.com/HarshD0011/AMOS/AMOS/tools"
)

// ContextGenerator builds the prompt context for the LLM
type ContextGenerator struct {
	tools *tools.K8sTools
}

// NewContextGenerator creates a new generator
func NewContextGenerator(tools *tools.K8sTools) *ContextGenerator {
	return &ContextGenerator{tools: tools}
}

// GenerateContext creates structured markdown context for the fault
func (cg *ContextGenerator) GenerateContext(fault UnifiedFault) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Detection Context\n\n"))
	sb.WriteString(fmt.Sprintf("- **Resource**: %s/%s\n", fault.Namespace, fault.Name))
	sb.WriteString(fmt.Sprintf("- **Kind**: %s\n", fault.Type))
	sb.WriteString(fmt.Sprintf("- **Fault**: %s\n", fault.Reason))
	sb.WriteString(fmt.Sprintf("- **Message**: %s\n", fault.Message))
	sb.WriteString(fmt.Sprintf("- **Time**: %s\n\n", fault.DetectedAt.Format("2006-01-02 15:04:05")))

	// Gather additional info based on type
	sb.WriteString("## Diagnostic Data\n\n")

	// 1. Get Details
	desc, err := cg.tools.DescribeResource(string(fault.Type), fault.Name, fault.Namespace)
	if err == nil {
		sb.WriteString("### Resource Status\n")
		sb.WriteString("```yaml\n")
		sb.WriteString(desc)
		sb.WriteString("\n```\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("> Failed to describe resource: %v\n\n", err))
	}

	// 2. Get Logs (if applicable - Pod or possibly Job/Deploy via LabelSelector)
	// For simple MVP we only pull logs if it's a Pod fault or we can find the pod
	if fault.Type == FaultTypePod {
		sb.WriteString("### Pod Logs (Last 50 lines)\n")
		logs, err := cg.tools.GetPodLogs(fault.Name, fault.Namespace, 50)
		if err == nil {
			sb.WriteString("```\n")
			// Truncate if too long (though tailLines helps)
			if len(logs) > 2000 {
				sb.WriteString(logs[len(logs)-2000:] + "\n... (truncated)")
			} else {
				sb.WriteString(logs)
			}
			sb.WriteString("\n```\n")
		} else {
			sb.WriteString(fmt.Sprintf("> Failed to retrieve logs: %v\n", err))
		}
	}

	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Analyze the above context. Determine the root cause of the failure. ")
	sb.WriteString("Suggest and EXECUTE a remediation plan using the available tools. ")
	sb.WriteString("Do not hallucinate tools. If you fix it, verify the fix if possible.")

	return sb.String(), nil
}